package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"go.uber.org/zap"
)

type SmsHandler struct {
	service *service.SmsService
	logger  *zap.Logger
}

func NewSmsHandler(s *service.SmsService, logger *zap.Logger) *SmsHandler {
	return &SmsHandler{service: s, logger: logger}
}

func (h *SmsHandler) SendStandard(c *gin.Context) {
	h.send(c, model.PriorityStandard)
}

func (h *SmsHandler) SendExpress(c *gin.Context) {
	h.send(c, model.PriorityExpress)
}

func (h *SmsHandler) send(c *gin.Context, priority string) {
	customerID, ok := getCustomerID(c)
	if !ok {
		return
	}

	var req model.SendSmsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid sms request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.service.SendSMS(c, customerID, &req, priority)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerNotFound), errors.Is(err, service.ErrWalletNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidMessage), errors.Is(err, service.ErrMessageTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("failed to send sms", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send sms"})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"messageId": msg.MessageID,
		"status":    msg.Status,
		"cost":      msg.Cost,
	})
}

func (h *SmsHandler) GetMessage(c *gin.Context) {
	customerID, ok := getCustomerID(c)
	if !ok {
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messageId is required"})
		return
	}

	msg, err := h.service.GetMessage(c, customerID, messageID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			h.logger.Error("failed to get message", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get message"})
		}
		return
	}

	c.JSON(http.StatusOK, msg)
}

func (h *SmsHandler) ListMessages(c *gin.Context) {
	customerID, ok := getCustomerID(c)
	if !ok {
		return
	}

	filters := &model.SmsReportFilters{CustomerID: customerID}
	filters.Status = strings.ToUpper(c.Query("status"))
	filters.Priority = strings.ToUpper(c.Query("priority"))
	filters.To = c.Query("to")

	fromStr := c.Query("from")
	if fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from timestamp"})
			return
		}
		filters.FromTime = &parsed
	}

	toStr := c.Query("to")
	if toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to timestamp"})
			return
		}
		filters.ToTime = &parsed
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		filters.Limit = parsed
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		filters.Offset = parsed
	}

	messages, err := h.service.ListMessages(c, filters)
	if err != nil {
		h.logger.Error("failed to list messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
