package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func getCustomerID(c *gin.Context) (string, bool) {
	customerID := strings.TrimSpace(c.GetHeader("X-Customer-ID"))
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Customer-ID header is required"})
		return "", false
	}

	return customerID, true
}
