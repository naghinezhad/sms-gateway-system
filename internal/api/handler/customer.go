package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"go.uber.org/zap"
)

type CustomerHandler struct {
	service *service.CustomerService
	logger  *zap.Logger
}

func NewCustomerHandler(s *service.CustomerService, logger *zap.Logger) *CustomerHandler {
	return &CustomerHandler{service: s, logger: logger}
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req model.CreateCustomerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid create customer request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.CreateCustomer(c, &req)
	if err != nil {
		switch err {
		case service.ErrCustomerExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrInvalidMessage:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("failed to create customer", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		}
		return
	}

	c.JSON(http.StatusCreated, customer)
}
