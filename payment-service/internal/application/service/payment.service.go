package service

import (
	"context"
	"fmt"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/zeebo/xxh3"
)

type PaymentService struct {
	repository ports.PaymentRepository
}

func NewPaymentService(repository ports.PaymentRepository) *PaymentService {
	return &PaymentService{
		repository: repository,
	}
}

func (uc *PaymentService) Create(ctx context.Context, props dto.CreatePaymentDTO) (*domain.Payment, error) {
	requestHash, err := uc.hashRequest(props)
	if err != nil {
		return nil, err
	}

	payment, err := uc.repository.SaveIdempotencyKey(ctx, props, props.IdempotencyKey, requestHash)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (uc *PaymentService) hashRequest(props dto.CreatePaymentDTO) (string, error) {
	body := fmt.Sprintf("%s|%s|%s|%s|%s", props.IdempotencyKey, props.Amount, props.Currency, props.From, props.To)

	hash128 := xxh3.Hash128([]byte(body))
	return fmt.Sprintf("%016x%016x", hash128.Hi, hash128.Lo), nil
}
