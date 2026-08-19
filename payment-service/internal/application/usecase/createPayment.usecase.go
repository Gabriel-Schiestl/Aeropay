package usecase

import (
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
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

func (uc *CreatePaymentUseCase) Execute(props dto.CreatePaymentDTO) error {
	// Implement the logic to create a payment
	return nil
}