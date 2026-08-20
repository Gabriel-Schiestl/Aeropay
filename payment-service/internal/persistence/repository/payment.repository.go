package repository

import (
	"context"
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

func (r *paymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	paymentQuery := `INSERT INTO payments (id, amount, currency, from_account, to_account) VALUES ($1, $2, $3, $4, $5)`
	transactionQuery := `INSERT INTO transactions (id, amount, currency, from_account, to_account, payment_id) VALUES ($1, $2, $3, $4, $5, $6)`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txPaymentStmt, err := tx.PrepareContext(ctx, paymentQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer txPaymentStmt.Close()

	_, err = txPaymentStmt.ExecContext(ctx, payment.ID(), payment.Amount().String(), payment.Currency().String(), payment.From(), payment.To())
	if err != nil {
		tx.Rollback()
		return err
	}

	txTransactionStmt, err := tx.PrepareContext(ctx, transactionQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer txTransactionStmt.Close()

	//TODO: Save concurrently
	for _, transaction := range payment.Transactions() {
		_, err = txTransactionStmt.ExecContext(ctx, transaction.ID(), transaction.Amount().String(), transaction.Currency().String(), transaction.From(), transaction.To(), payment.ID())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}