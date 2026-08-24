package controller

import "github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"

func mapErrorToHTTPStatus(err error) int {
	switch err {
		case exception.ErrCurrencyMismatch, exception.ErrInvalidAmount, exception.ErrInvalidAccount, exception.ErrSameAccount, exception.ErrAccountNotFound, exception.ErrInsufficientFunds:
			return 400
	}
	return 500
}