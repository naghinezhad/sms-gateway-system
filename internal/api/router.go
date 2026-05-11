package api

import (
	"github.com/gin-gonic/gin"
	"github.com/naghinezhad/sms-gateway-system/internal/api/handler"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"go.uber.org/zap"
)

func SetupRouter(
	smsService *service.SmsService,
	walletService *service.WalletService,
	customerService *service.CustomerService,
	logger *zap.Logger,
) *gin.Engine {
	r := gin.Default()

	customerHandler := handler.NewCustomerHandler(customerService, logger)
	walletHandler := handler.NewWalletHandler(walletService, logger)
	smsHandler := handler.NewSmsHandler(smsService, logger)

	r.POST("/api/customers", customerHandler.CreateCustomer)

	r.POST("/api/wallet/topup", walletHandler.TopUp)
	r.GET("/api/wallet/balance", walletHandler.GetBalance)

	r.POST("/api/sms", smsHandler.SendStandard)
	r.POST("/api/sms/express", smsHandler.SendExpress)
	r.GET("/api/sms/:messageId", smsHandler.GetMessage)
	r.GET("/api/sms", smsHandler.ListMessages)

	return r
}
