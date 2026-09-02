package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence/model"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/queue"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	app := fx.New(
		fx.Provide(config.LoadDBConfig, config.LoadQueueConfig),
		fx.Provide(persistence.NewDB),
		fx.Invoke(func(db *sql.DB, dbConfig *config.DBConfig) {
			if err := persistence.RunMigrations(db, dbConfig); err != nil {
				panic(err)
			}
		}),
		fx.Provide(
			fx.Annotate(
				queue.NewPublisher,
				fx.As(new(ports.Publisher)),
			),
		),
		fx.Invoke(observability.RegisterDBCollector),
		fx.Invoke(func(lc fx.Lifecycle, db *sql.DB, queueConfig *config.QueueConfig, publisher ports.Publisher) {
			ctx, cancel := context.WithCancel(context.Background())

			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					if err := publisher.CreateTopic(); err != nil {
						panic(err)
					}

					go func() {
						publishOutboxMessages(ctx, publisher, db, queueConfig)
					}()
					return nil
				},
				OnStop: func(context.Context) error {
					cancel()
					publisher.Close()
					return nil
				},
			})
		}),
	)

	app.Run()
}

func publishOutboxMessages(ctx context.Context, publisher ports.Publisher, db *sql.DB, cfg *config.QueueConfig) {
	selectStmt, err := db.PrepareContext(ctx, `SELECT id, event_type, payload FROM payments_outbox WHERE status = 'pending' ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		panic(err)
	}
	defer selectStmt.Close()

	updateStmt, err := db.PrepareContext(ctx, `UPDATE payments_outbox SET status = 'processed' WHERE id = ANY($1)`)
	if err != nil {
		panic(err)
	}
	defer updateStmt.Close()

	ticker := time.NewTicker(time.Duration(cfg.OutboxPollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping outbox publisher...")
			return
		case <-ticker.C:
			publishPendingMessages(ctx, publisher, db, cfg, selectStmt, updateStmt)
		}
	}

	
}

func publishPendingMessages(ctx context.Context, publisher ports.Publisher, db *sql.DB, cfg *config.QueueConfig, selectStmt, updateStmt *sql.Stmt) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %v", err)
		return
	}

	rows, err := tx.StmtContext(ctx, selectStmt).QueryContext(ctx, cfg.OutboxBatchSize)
	if err != nil {
		log.Printf("failed to query outbox messages: %v", err)
		tx.Rollback()
		return
	}
	defer rows.Close()

	var messages []model.OutboxModel
	for rows.Next() {
		var message model.OutboxModel
		if err := rows.Scan(&message.ID, &message.EventType, &message.Payload); err != nil {
			log.Printf("failed to scan outbox message: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	// TODO: handle each event type in a separate function to avoid a large switch statement
	processed := make([]int, len(messages))
	for _, message := range messages {
		switch message.EventType {
		case "create_payment":
			var body dto.CreatePaymentDTO
			err := json.Unmarshal([]byte(message.Payload), &body)
			if err != nil {
				log.Printf("failed to unmarshal message payload with ID %d: %v", message.ID, err)
				continue
			}

			if err := publisher.Publish(message.Payload, body.From); err != nil {
				log.Printf("failed to publish message with ID %d: %v", message.ID, err)
				continue
			}

			processed = append(processed, message.ID)
		}
	}

	if len(processed) > 0 {
		_, err := tx.StmtContext(ctx, updateStmt).ExecContext(ctx, processed)
		if err != nil {
			log.Printf("failed to update outbox messages: %v", err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit transaction: %v", err)
		tx.Rollback()
		return
	}
}