package ports

import "github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"

type PaymentRepository interface {
	Save(payment domain.Payment) error
}