package handlers

import (
	"globe/dodrio_loan_availment/internal/pkg/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ArrowLoansCacheHandler struct {
	service *services.ArrowLoansCacheService
}

// NewAvailmentHandler creates a new instance of AvailmentHandler
func NewArrowLoansCacheHandler() *ArrowLoansCacheHandler {
	return &ArrowLoansCacheHandler{
		service: services.NewArrowLoansCacheService(),
	}
}

func (h *ArrowLoansCacheHandler) ArrowLoansCache(c *gin.Context) {
	var jsonData map[string]time.Time

	if err := c.BindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Date Format"})
		return
	}

	results, err := h.service.ArrowLoansCacheTable(c,jsonData["endDatetime"])

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
