package ports

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
}