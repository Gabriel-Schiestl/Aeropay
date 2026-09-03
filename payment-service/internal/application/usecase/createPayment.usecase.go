package usecase

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
)

type CreatePaymentUseCase struct {
	repository ports.PaymentRepository
}

func NewCreatePaymentUseCase(repository ports.PaymentRepository) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		repository: repository,
	}
}

func (uc *CreatePaymentUseCase) Execute(ctx context.Context, event dto.PaymentAcceptedEvent) error {
	payment, err := domain.NewPayment(event.Amount, event.Currency, event.From, event.To)
	if err != nil {
		observability.RecordPaymentProcessing(event.AcceptedAt, "error")
		return err
	}

	err = uc.repository.Save(ctx, payment, event.IdempotencyKey)
	if err != nil {
		observability.RecordPaymentProcessing(event.AcceptedAt, "error")
		return err
	}

	observability.RecordPaymentProcessing(event.AcceptedAt, "success")
	return nil
}