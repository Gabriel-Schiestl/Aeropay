package exception

import "errors"

var ErrCurrencyMismatch = errors.New("currency mismatch")
var ErrInvalidAmount = errors.New("invalid amount")
var ErrInvalidAccount = errors.New("invalid account")
var ErrSameAccount = errors.New("from and to accounts cannot be the same")
var ErrAccountNotFound = errors.New("account not found")
var ErrInsufficientFunds = errors.New("insufficient funds")