package usecase

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type CreatePaymentUseCase struct {
	repository ports.PaymentRepository
}

func NewCreatePaymentUseCase(repository ports.PaymentRepository) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		repository: repository,
	}
}

func (uc *CreatePaymentUseCase) Execute(ctx context.Context, props dto.CreatePaymentDTO) error {
	payment, err := domain.NewPayment(props.Amount, props.Currency, props.From, props.To)
	if err != nil {
		return err
	}

	err = uc.repository.Save(ctx, payment, props.IdempotencyKey)
	if err != nil {
		return err
	}

	return nil
}