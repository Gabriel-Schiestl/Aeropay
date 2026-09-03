package controller

import "github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/exception"

func mapErrorToHTTPStatus(err error) int {
	switch err {
		case exception.ErrCurrencyMismatch, exception.ErrInvalidAmount, exception.ErrInvalidAccount, exception.ErrSameAccount, exception.ErrAccountNotFound, exception.ErrInsufficientFunds:
			return 400
		// Same key, different payload: still in flight, or already used - client conflict.
		case exception.ErrIdempotencyKeyProcessing, exception.ErrIdempotencyKeyAlreadyUsed:
			return 409
		// Same key/payload as a previous attempt that ended in error.
		case exception.ErrIdempotencyKeyError:
			return 422
	}
	return 500
}