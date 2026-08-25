package repository

import (
	"context"
	"database/sql"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type paymentRepository struct {
	db                    *sql.DB
	paymentStmt           *sql.Stmt
	ledgerStmt            *sql.Stmt
	debitStmt             *sql.Stmt
	creditStmt            *sql.Stmt
	selectAccountLockStmt *sql.Stmt
}

func NewPaymentRepository(db *sql.DB) ports.PaymentRepository {
	ctx := context.Background()

	paymentStmt, err := db.PrepareContext(ctx, `INSERT INTO payments (id, amount, currency, from_account, to_account) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		panic(err)
	}

	ledgerStmt, err := db.PrepareContext(ctx, `INSERT INTO ledger (amount, currency, account, payment_id) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		panic(err)
	}

	debitStmt, err := db.PrepareContext(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1`)
	if err != nil {
		panic(err)
	}

	creditStmt, err := db.PrepareContext(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`)
	if err != nil {
		panic(err)
	}

	selectAccountLockStmt, err := db.PrepareContext(ctx, `SELECT id, balance FROM accounts WHERE id = $1 FOR UPDATE`)
	if err != nil {
		panic(err)
	}

	return &paymentRepository{
		db:                    db,
		paymentStmt:           paymentStmt,
		ledgerStmt:            ledgerStmt,
		debitStmt:             debitStmt,
		creditStmt:            creditStmt,
		selectAccountLockStmt: selectAccountLockStmt,
	}
}

func (r *paymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
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

	_, err = tx.StmtContext(ctx, r.paymentStmt).ExecContext(ctx, payment.ID(), payment.Amount().String(), payment.Currency().String(), payment.From(), payment.To())
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
	var id, balance string
	err := tx.StmtContext(ctx, r.selectAccountLockStmt).QueryRowContext(ctx, accountID).Scan(&id, &balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return exception.ErrAccountNotFound
		}
		return err
	}

	return nil
}

func (r *paymentRepository) debitAccount(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error {
	res, err := tx.StmtContext(ctx, r.debitStmt).ExecContext(ctx, payment.Amount().String(), payment.From())
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

	_, err = tx.StmtContext(ctx, r.ledgerStmt).ExecContext(ctx, payment.Amount().Neg().String(), payment.Currency().String(), payment.From(), payment.ID())
	if err != nil {
		return err
	}

	return nil
}

func (r *paymentRepository) creditAccount(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error {
	_, err := tx.StmtContext(ctx, r.creditStmt).ExecContext(ctx, payment.Amount().String(), payment.To())
	if err != nil {
		return err
	}

	_, err = tx.StmtContext(ctx, r.ledgerStmt).ExecContext(ctx, payment.Amount().String(), payment.Currency().String(), payment.To(), payment.ID())
	if err != nil {
		return err
	}

	return nil
}
