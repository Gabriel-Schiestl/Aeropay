package persistence

import (
	"database/sql"
	"fmt"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	_ "github.com/lib/pq"
)

func NewDB(cfg *config.DBConfig) *sql.DB {
	db, err := sql.Open(cfg.Driver, fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name))
	if err != nil {
		panic(err)
	}
	return db
}