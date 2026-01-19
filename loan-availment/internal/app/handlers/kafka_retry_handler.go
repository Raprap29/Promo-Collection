package handlers

import (
	app "globe/dodrio_loan_availment/internal/app"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KafkaRetryHandler struct {
	service app.KafkaRetryService
}

func NewKafkaRetryHandler(service app.KafkaRetryService) *KafkaRetryHandler {
	return &KafkaRetryHandler{service: service}
}
func (h *KafkaRetryHandler) RetryKafkaAvailmentMessage(c *gin.Context) {

	successMessage, failedMessages, err := h.service.RetryKafkaAvailmentMessage(c.Request.Context())
	if err != nil && len(successMessage) > 0 {
		c.JSON(http.StatusOK, gin.H{"Success Messages": successMessage, "failedMessages": failedMessages, "error": err})
		return
	} else if err != nil && len(successMessage) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"Success Messages": successMessage, "failedMessages": failedMessages})
}
