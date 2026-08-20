package model

type PaymentModel struct {
	ID 		 string `db:"id"`
	Amount   string `db:"amount"`
	Currency string `db:"currency"`
	From     string `db:"from_account"`
	To       string `db:"to_account"`
}