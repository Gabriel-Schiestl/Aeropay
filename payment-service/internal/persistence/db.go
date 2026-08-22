package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	_ "github.com/lib/pq"
)

func NewDB(cfg *config.DBConfig) *sql.DB {
	db, err := sql.Open(cfg.Driver, fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name))
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	return db
}

func RunMigrations(db *sql.DB, dbConfig *config.DBConfig) error {
	sqlBytes, err := os.ReadFile(dbConfig.MigrationsFile)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("failed to execute SQL migrations: %w", err)
	}

	fmt.Println("Database migrations completed successfully")

	return nil
}
