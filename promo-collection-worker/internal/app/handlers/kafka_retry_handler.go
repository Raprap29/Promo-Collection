package handlers

import (
	"net/http"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/service"

	"github.com/gin-gonic/gin"
)

type KafkaRetryHandler struct {
	service service.KafkaRetryServiceInterface
}

func NewKafkaRetryHandler(service service.KafkaRetryServiceInterface) *KafkaRetryHandler {
	return &KafkaRetryHandler{
		service: service,
	}
}

func (h *KafkaRetryHandler) RetryKafkaCollectionMessages(c *gin.Context) {
	response := h.service.RetryKafkaCollectionMessages(c.Request.Context())

	if response.ErrorMsg == "" {
		if len(response.SuccessIDs) == 0 && len(response.FailedIDs) == 0 {
			response.Message = log_messages.NoCollectionInDuration
		}

		c.JSON(http.StatusOK, response)
		return
	}

	if len(response.SuccessIDs) > 0 {
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusInternalServerError, response)
}
