package controller

import (
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/usecase"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/server"
	"github.com/gin-gonic/gin"
)

type CoreController struct {
	createPaymentUseCase *usecase.CreatePaymentUseCase
}

func NewCoreController(createPaymentUseCase *usecase.CreatePaymentUseCase) *CoreController {
	return &CoreController{
		createPaymentUseCase: createPaymentUseCase,
	}
}

func (cc *CoreController) createPaymentHandler(ctx *gin.Context) {
	var body dto.CreatePaymentDTO

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	if err := cc.createPaymentUseCase.Execute(body); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(201, gin.H{"message": "Payment created successfully"})
}

func (cc *CoreController) RegisterRoutes(srv *server.Server) {
	srv.Instance.POST("/payments", cc.createPaymentHandler)
}