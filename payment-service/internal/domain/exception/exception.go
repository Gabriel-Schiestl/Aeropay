package exception

import "errors"

var ErrCurrencyMismatch = errors.New("currency mismatch")
var ErrInvalidAmount = errors.New("invalid amount")