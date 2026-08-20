package domain

import (
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Payment struct {
	id	   string
	amount   decimal.Decimal
	currency Currency
	from     string
	to       string
	transactions []*Transaction
}

func NewPayment(amount decimal.Decimal, currency, from, to string) (*Payment, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, exception.ErrInvalidAmount
	}

	currencyEnum, err := ParseCurrency(currency)
	if err != nil {
		return nil, err
	}

	transactionFrom := NewTransaction(amount, currencyEnum, from, to)
	transactionTo := NewTransaction(amount, currencyEnum, to, from)

	return &Payment{
		id:       uuid.New().String(),
		amount:   amount,
		currency: currencyEnum,
		from:     from,
		to:       to,
		transactions: []*Transaction{transactionFrom, transactionTo},
	}, nil
}

func (p *Payment) ID() string { return p.id }
func (p *Payment) Amount() decimal.Decimal { return p.amount }
func (p *Payment) Currency() Currency { return p.currency }
func (p *Payment) From() string { return p.from }
func (p *Payment) To() string { return p.to }
func (p *Payment) Transactions() []*Transaction { return p.transactions }
