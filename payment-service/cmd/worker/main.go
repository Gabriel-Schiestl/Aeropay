package main

import (
	"context"
	"log"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/usecase"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence/repository"
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
		fx.Provide(usecase.NewCreatePaymentUseCase),
		fx.Provide(
			fx.Annotate(
				repository.NewPaymentRepository,
				fx.As(new(ports.PaymentRepository)),
			),
		),
		fx.Provide(
			fx.Annotate(
				queue.NewConsumer[dto.CreatePaymentDTO],
				fx.As(new(ports.Consumer)),
			),
		),
		fx.Invoke(observability.RegisterDBCollector),
		fx.Invoke(func(lc fx.Lifecycle, consumer ports.Consumer, useCase *usecase.CreatePaymentUseCase, queueConfig *config.QueueConfig) {
			ctx, cancel := context.WithCancel(context.Background())

			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go func() {
						if err := consumer.Consume(ctx); err != nil {
							log.Printf("consumer error: %v", err)
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
