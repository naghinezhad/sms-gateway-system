package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"go.uber.org/zap"
)

type WalletHandler struct {
	service *service.WalletService
	logger  *zap.Logger
}

func NewWalletHandler(s *service.WalletService, logger *zap.Logger) *WalletHandler {
	return &WalletHandler{service: s, logger: logger}
}

func (h *WalletHandler) TopUp(c *gin.Context) {
	customerID, ok := getCustomerID(c)
	if !ok {
		return
	}

	var req model.TopUpInput
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid topup request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance, err := h.service.TopUp(c, customerID, req.Amount)
	if err != nil {
		switch err {
		case service.ErrCustomerNotFound, service.ErrWalletNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrInvalidAmount:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("topup failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "topup failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	customerID, ok := getCustomerID(c)
	if !ok {
		return
	}

	balance, err := h.service.GetBalance(c, customerID)
	if err != nil {
		switch err {
		case service.ErrCustomerNotFound, service.ErrWalletNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			h.logger.Error("failed to get balance", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get balance"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}
