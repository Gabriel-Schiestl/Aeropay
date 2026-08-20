package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	id     string
	amount decimal.Decimal
	currency Currency
	from   string
	to     string
}

func NewTransaction(amount decimal.Decimal, currency Currency, from, to string) *Transaction {
	return &Transaction{
		id:       uuid.New().String(),
		amount:   amount,
		currency: currency,
		from:     from,
		to:       to,
	}
}

func (t *Transaction) ID() string {return t.id}
func (t *Transaction) Amount() decimal.Decimal {return t.amount}
func (t *Transaction) Currency() Currency {return t.currency}
func (t *Transaction) From() string {return t.from}
func (t *Transaction) To() string {return t.to}