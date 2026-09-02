package ports

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
	SaveIdempotencyKey(ctx context.Context, payload dto.CreatePaymentDTO, key, requestHash string) (*domain.Payment, error)
}