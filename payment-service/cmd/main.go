package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/usecase"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/persistence/repository"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/controller"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/server"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	app := fx.New(
		fx.Provide(server.NewServer, config.LoadHTTPConfig,
			config.LoadDBConfig, persistence.NewDB),
		fx.Provide(usecase.NewCreatePaymentUseCase),
		fx.Provide(controller.NewCoreController),
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
		fx.Invoke(func(srv *server.Server, httpConfig *config.HTTPConfig, coreController *controller.CoreController) {
			coreController.RegisterRoutes(srv)

			err := srv.Start(httpConfig.Port)
			if err != nil {
				panic(err)
			}
			fmt.Println("Server started on port 8080")
		}),
	)

	app.Run()
}
