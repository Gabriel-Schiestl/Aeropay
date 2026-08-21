package repository

import (
	"context"
	"database/sql"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type paymentRepository struct {
	db *sql.DB
	paymentQuery     string
	ledgerQuery string
	debitQuery	  string
	creditQuery	  string
	selectAccountLockQuery string
}

func NewPaymentRepository(db *sql.DB) ports.PaymentRepository {
	return &paymentRepository{
		db: db,
		paymentQuery: `INSERT INTO payments (id, amount, currency, from_account, to_account) VALUES ($1, $2, $3, $4, $5)`,
		ledgerQuery: `INSERT INTO ledger (id, amount, currency, account, payment_id) VALUES ($1, $2, $3, $4, $5)`,
		debitQuery: `UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1`,
		creditQuery: `UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		selectAccountLockQuery: `SELECT id, balance FROM accounts WHERE id = $1 FOR UPDATE`,
	}
}

func (r *paymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txPaymentStmt, err := tx.PrepareContext(ctx, r.paymentQuery)
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

	first, second := payment.From(), payment.To()
	if second < first {
		first, second = second, first
	}

	err = r.lockAccount(ctx, tx, first)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = r.lockAccount(ctx, tx, second)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = r.debitAccount(ctx, tx, payment)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = r.creditAccount(ctx, tx, payment)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (r *paymentRepository) lockAccount(ctx context.Context, tx *sql.Tx, accountID string) error {
	txSelectStmt, err := tx.PrepareContext(ctx, r.selectAccountLockQuery)
	if err != nil {
		return err
	}
	defer txSelectStmt.Close()

	var id, balance string
	err = txSelectStmt.QueryRowContext(ctx, accountID).Scan(&id, &balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return exception.ErrAccountNotFound
		}
		return err
	}

	return nil
}

func (r *paymentRepository) debitAccount(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error {
	txDebitStmt, err := tx.PrepareContext(ctx, r.debitQuery)
	if err != nil {
		return err
	}
	defer txDebitStmt.Close()

	res, err := txDebitStmt.ExecContext(ctx, payment.Amount().String(), payment.From())
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return exception.ErrInsufficientFunds
	}

	txLedgerStmt, err := tx.PrepareContext(ctx, r.ledgerQuery)
	if err != nil {
		return err
	}
	defer txLedgerStmt.Close()

	_, err = txLedgerStmt.ExecContext(ctx, payment.ID(), payment.Amount().Neg().String(), payment.Currency().String(), payment.From(), payment.ID())
	if err != nil {
		return err
	}

	return nil
}

func (r *paymentRepository) creditAccount(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error {
	txCreditStmt, err := tx.PrepareContext(ctx, r.creditQuery)
	if err != nil {
		return err
	}
	defer txCreditStmt.Close()

	_, err = txCreditStmt.ExecContext(ctx, payment.Amount().String(), payment.To())
	if err != nil {
		return err
	}

	txLedgerStmt, err := tx.PrepareContext(ctx, r.ledgerQuery)
	if err != nil {
		return err
	}
	defer txLedgerStmt.Close()

	_, err = txLedgerStmt.ExecContext(ctx, payment.ID(), payment.Amount().String(), payment.Currency().String(), payment.To(), payment.ID())
	if err != nil {
		return err
	}

	return nil
}