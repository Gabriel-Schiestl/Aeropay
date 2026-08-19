package domain

import "github.com/shopspring/decimal"

type Payment struct {
	id	   string
	amount   decimal.Decimal
	currency Currency
	from     string
	to       string
}

func NewPayment(id string, amount decimal.Decimal, currency Currency, from string, to string) *Payment {
	return &Payment{
		id:       id,
		amount:   amount,
		currency: currency,
		from:     from,
		to:       to,
	}
}

func (p *Payment) ID() string {
	return p.id
}

func (p *Payment) Amount() decimal.Decimal {
	return p.amount
}

func (p *Payment) Currency() Currency {
	return p.currency
}

func (p *Payment) From() string {
	return p.from
}

func (p *Payment) To() string {
	return p.to
}
