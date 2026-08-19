package dto

import "github.com/shopspring/decimal"

type CreatePaymentDTO struct {
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
	From    string  `json:"from" binding:"required"`
	To      string  `json:"to" binding:"required"`
}