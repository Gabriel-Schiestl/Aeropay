package controller

import "github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"

func mapErrorToHTTPStatus(err error) int {
	switch err {
		case exception.ErrCurrencyMismatch:
			return 400
	}
	return 500
}