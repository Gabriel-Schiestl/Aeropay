package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/usecase"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence/repository"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/queue"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/server"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	app := fx.New(
		fx.Provide(server.NewServer, config.LoadHTTPConfig, config.LoadDBConfig, config.LoadQueueConfig),
		fx.Provide(persistence.NewDB),
		fx.Invoke(func(db *sql.DB, dbConfig *config.DBConfig) {
			if err := persistence.RunMigrations(db, dbConfig); err != nil {
				panic(err)
			}
		}),
		fx.Provide(
			fx.Annotate(
				usecase.NewCreatePaymentUseCase,
				fx.As(new(queue.Handler[dto.PaymentAcceptedEvent])),
			),
		),
		fx.Provide(
			fx.Annotate(
				repository.NewPaymentRepository,
				fx.As(new(ports.PaymentRepository)),
			),
		),
		fx.Provide(
			fx.Annotate(
				queue.NewConsumer[dto.PaymentAcceptedEvent],
				fx.As(new(ports.Consumer)),
			),
		),
		fx.Invoke(observability.RegisterDBCollector),
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server, httpConfig *config.HTTPConfig) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go func() {
						if err := srv.Start(httpConfig.Port); err != nil {
							log.Printf("metrics server error: %v", err)
						}
					}()
					log.Printf("Metrics server started on port %s", httpConfig.Port)
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return srv.Shutdown(ctx)
				},
			})
		}),
		fx.Invoke(func(lc fx.Lifecycle, consumer ports.Consumer, queueConfig *config.QueueConfig) {
			ctx, cancel := context.WithCancel(context.Background())

			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go func() {
						if err := consumer.Consume(ctx); err != nil {
							log.Fatalf("consumer error: %v", err)
						}
					}()
					return nil
				},
				OnStop: func(context.Context) error {
					cancel()
					return nil
				},
			})
		}),
	)

	app.Run()
}
