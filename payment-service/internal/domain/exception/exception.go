package exception

import "errors"

var ErrCurrencyMismatch = errors.New("currency mismatch")
var ErrInvalidAmount = errors.New("invalid amount")
var ErrInvalidAccount = errors.New("invalid account")
var ErrSameAccount = errors.New("from and to accounts cannot be the same")
var ErrAccountNotFound = errors.New("account not found")
var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrIdempotencyKeyProcessing = errors.New("idempotency key is already being processed")
var ErrIdempotencyKeyAlreadyUsed = errors.New("idempotency key has already been used")
var ErrIdempotencyKeyError = errors.New("idempotency key has an error status")
var ErrPaymentNotFound = errors.New("payment not found")