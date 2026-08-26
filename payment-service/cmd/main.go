package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/service"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence/repository"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/controller"
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
		fx.Provide(server.NewServer, config.LoadHTTPConfig,
			config.LoadDBConfig, config.LoadQueueConfig),
		fx.Provide(persistence.NewDB),
		fx.Provide(service.NewPaymentService),
		fx.Provide(controller.NewCoreController),
		fx.Provide(
			fx.Annotate(
				queue.NewPublisher[dto.CreatePaymentDTO],
				fx.As(new(ports.Publisher[dto.CreatePaymentDTO])),
			),
		),
		fx.Provide(
			fx.Annotate(
				repository.NewPaymentRepository,
				fx.As(new(ports.PaymentRepository)),
			),
		),
		fx.Invoke(func(db *sql.DB, dbConfig *config.DBConfig) {
			err := persistence.RunMigrations(db, dbConfig)
			if err != nil {
				panic(err)
			}
		}),
		fx.Invoke(observability.RegisterDBCollector),
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server, httpConfig *config.HTTPConfig, coreController *controller.CoreController, publisher ports.Publisher[dto.CreatePaymentDTO]) {
			coreController.RegisterRoutes(srv)

			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go func() {
						if err := srv.Start(httpConfig.Port); err != nil {
							log.Printf("server error: %v", err)
						}
					}()
					log.Printf("Server started on port %s", httpConfig.Port)
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Println("Shutting down gracefully...")
					publisher.Close()
					return srv.Shutdown(ctx)
				},
			})
		}),
	)

	app.Run()
}
