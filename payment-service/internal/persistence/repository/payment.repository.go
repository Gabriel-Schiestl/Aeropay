package repository

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/shopspring/decimal"
)

type paymentRepository struct {
	db                    *sql.DB
	paymentStmt           *sql.Stmt
	ledgerStmt            *sql.Stmt
	debitStmt             *sql.Stmt
	creditStmt            *sql.Stmt
	selectAccountLockStmt *sql.Stmt
	insertIdempotencyKeyStmt *sql.Stmt
	selectIdempotencyKeyStmt *sql.Stmt
	getPaymentByIdStmt     *sql.Stmt
	ttl				int
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

	insertIdempotencyKeyStmt, err := db.PrepareContext(ctx, `INSERT INTO idempotency_keys (key, request_hash, status, expires_at) VALUES ($1, $2, 'processing', $4)`)
	if err != nil {
		panic(err)
	}

	selectIdempotencyKeyStmt, err := db.PrepareContext(ctx, `SELECT key, request_hash, status, expires_at, payment_id FROM idempotency_keys WHERE key = $1`)
	if err != nil {
		panic(err)
	}

	getPaymentByIdStmt, err := db.PrepareContext(ctx, `SELECT id, amount, currency, from_account, to_account FROM payments WHERE id = $1`)
	if err != nil {
		panic(err)
	}

	ttl := 0
	if ttlStr := os.Getenv("IDEMPOTENCY_KEY_TTL"); ttlStr != "" {
		if t, err := strconv.Atoi(ttlStr); err == nil {
			ttl = t
		}
	}

	return &paymentRepository{
		db:                    db,
		paymentStmt:           paymentStmt,
		ledgerStmt:            ledgerStmt,
		debitStmt:             debitStmt,
		creditStmt:            creditStmt,
		selectAccountLockStmt: selectAccountLockStmt,
		insertIdempotencyKeyStmt: insertIdempotencyKeyStmt,
		selectIdempotencyKeyStmt: selectIdempotencyKeyStmt,
		getPaymentByIdStmt: getPaymentByIdStmt,
		ttl: ttl,
	}
}

func (r *paymentRepository) SaveIdempotencyKey(ctx context.Context, key, requestHash string) (*domain.Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	var (
		existent, status, paymentID, hash string
		expiresAt time.Time
	)

	err = tx.StmtContext(ctx, r.selectIdempotencyKeyStmt).QueryRowContext(ctx, key).Scan(&existent, &hash, &status, &expiresAt, &paymentID)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	}

	//TODO: refactor this logic to be more readable and maintainable
	if existent != "" {
		switch status {
		case "processing":
			return nil, exception.ErrIdempotencyKeyProcessing
		case "completed":
			if hash != requestHash && expiresAt.After(time.Now()) {
				return nil, exception.ErrIdempotencyKeyAlreadyUsed
			}

			if hash != requestHash && expiresAt.Before(time.Now()) {
				err := r.updateIdempotencyKey(ctx, tx, key, requestHash)
				if err != nil {
					tx.Rollback()
					return nil, err
				}
				err = tx.Commit()
				if err != nil {
					tx.Rollback()
					return nil, err
				}
				return nil, nil
			}
			payment, err := r.getPaymentById(ctx, tx, paymentID)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			return payment, nil
		case "error":
			if hash != requestHash {
				return nil, exception.ErrIdempotencyKeyAlreadyUsed
			}
			return nil, exception.ErrIdempotencyKeyError
		}
	}
	_, err = tx.StmtContext(ctx, r.insertIdempotencyKeyStmt).ExecContext(ctx, key, requestHash, time.Now().Add(time.Duration(r.ttl)*time.Second))
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return nil, nil
}

func (r *paymentRepository) updateIdempotencyKey(ctx context.Context, tx *sql.Tx, key, requestHash string) error {
	_, err := tx.ExecContext(ctx, `UPDATE idempotency_keys SET request_hash = $1, status = 'processing', expires_at = $2 WHERE key = $3`, requestHash, time.Now().Add(time.Duration(r.ttl)*time.Second), key)
	if err != nil {
		return err
	}
	return nil
}

func (r *paymentRepository) getPaymentById(ctx context.Context, tx *sql.Tx, paymentID string) (*domain.Payment, error) {
	var (
		id, amount, currency, from, to string
	)
	err := tx.QueryRowContext(ctx, `SELECT id, amount, currency, from_account, to_account FROM payments WHERE id = $1`, paymentID).Scan(&id, &amount, &currency, &from, &to)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, exception.ErrPaymentNotFound
		}
		return nil, err
	}

	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, err
	}

	payment, err := domain.LoadPayment(id, dec, currency, from, to)
	if err != nil {
		return nil, err
	}
	return payment, nil
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
