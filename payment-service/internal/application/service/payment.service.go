package service

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type PaymentService struct {
	repository ports.PaymentRepository
	publisher ports.Publisher[dto.CreatePaymentDTO]
}

func NewPaymentService(repository ports.PaymentRepository, publisher ports.Publisher[dto.CreatePaymentDTO]) *PaymentService {
	return &PaymentService{
		repository: repository,
		publisher: publisher,
	}
}

func (uc *PaymentService) Create(ctx context.Context, props dto.CreatePaymentDTO) error {
	// TODO: check idempotency key and publish to queue

	return nil
}