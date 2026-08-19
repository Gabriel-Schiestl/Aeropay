package usecase

import "github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"

type CreatePaymentUseCase struct {}

func NewCreatePaymentUseCase() *CreatePaymentUseCase {
	return &CreatePaymentUseCase{}
}

func (uc *CreatePaymentUseCase) Execute(props dto.CreatePaymentDTO) error {
	// Implement the logic to create a payment
	return nil
}