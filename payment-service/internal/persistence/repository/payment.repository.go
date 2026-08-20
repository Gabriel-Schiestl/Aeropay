package repository

import (
	"database/sql"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) ports.PaymentRepository {
	return &paymentRepository{
		db: db,
	}
}

func (r *paymentRepository) Save(payment *domain.Payment) error {
	return nil
}