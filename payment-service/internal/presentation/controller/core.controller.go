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
	
	if err := cc.paymentService.Create(ctx.Request.Context(), body); err != nil {
		ctx.JSON(mapErrorToHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(201, gin.H{"message": "Payment created successfully"})
}

func (cc *CoreController) RegisterRoutes(srv *server.Server) {
	srv.Instance.POST("/payments", cc.createPaymentHandler)
}