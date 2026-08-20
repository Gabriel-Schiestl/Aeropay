package domain

import (
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"
	"github.com/shopspring/decimal"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	BRL Currency = "BRL"
)

var currencyMap = map[string]Currency{
	"USD": USD,
	"EUR": EUR,
	"BRL": BRL,
}

func ParseCurrency(currencyStr string) (Currency, error) {
	if currency, ok := currencyMap[currencyStr]; ok {
		return currency, nil
	}
	return "", exception.ErrCurrencyMismatch
}

type Money struct {
	amount decimal.Decimal
	currency Currency
}

func NewMoney(amount decimal.Decimal, currency Currency) *Money {
	return &Money{
		amount:   amount,
		currency: currency,
	}
}

func (m *Money) Add(other *Money) error {
	if m.currency != other.currency {
		return exception.ErrCurrencyMismatch
	}

	m.amount = m.amount.Add(other.amount)

	return nil
}

func (m *Money) Subtract(other *Money) error {
	if m.currency != other.currency {
		return exception.ErrCurrencyMismatch
	}

	m.amount = m.amount.Sub(other.amount)

	return nil
}

//-------- GETTERS --------

func (m *Money) Amount() decimal.Decimal {
	return m.amount
}

func (m *Money) Currency() Currency {
	return m.currency
}