package service

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
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

func (uc *PaymentService) Create(ctx context.Context, props dto.CreatePaymentDTO) (*domain.Payment, error) {
	//TODO: Implement hashRequest method
	requestHash, err := uc.hashRequest(props)
	if err != nil {
		return nil, err
	}

	payment, err := uc.repository.SaveIdempotencyKey(ctx, props.IdempotencyKey, requestHash)
	if err != nil {
		return nil, err
	}

	if payment == nil {
		uc.publisher.Publish(props)
		return nil, nil
	}

	return payment, nil
}