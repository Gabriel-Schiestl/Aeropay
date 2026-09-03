package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type CreatePaymentDTO struct {
	IdempotencyKey string  `json:"idempotency_key" binding:"required"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
	From    string  `json:"from" binding:"required"`
	To      string  `json:"to" binding:"required"`
}

type PaymentAcceptedEvent struct {
	CreatePaymentDTO
	AcceptedAt time.Time `json:"accepted_at"`
}