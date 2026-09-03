package controller

import (
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/dto"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/application/service"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/presentation/server"
	"github.com/gin-gonic/gin"
)

type CoreController struct {
	paymentService *service.PaymentService
}

func NewCoreController(paymentService *service.PaymentService) *CoreController {
	return &CoreController{
		paymentService: paymentService,
	}
}

func (cc *CoreController) createPaymentHandler(ctx *gin.Context) {
	var body dto.CreatePaymentDTO

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	payment, err := cc.paymentService.Create(ctx.Request.Context(), body)
	if err != nil {
		ctx.JSON(mapErrorToHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}


	if payment != nil {
		ctx.JSON(200, gin.H{"message": "Payment already processed", "payment": payment})
		return
	}

	ctx.JSON(202, gin.H{"message": "Payment accepted for processing"})
}

func (cc *CoreController) RegisterRoutes(srv *server.Server) {
	srv.Instance.POST("/payments", cc.createPaymentHandler)
}