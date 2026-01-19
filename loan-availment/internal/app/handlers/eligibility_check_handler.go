package handlers

import (
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EligibilityCheckHandler struct {
	service services.EligibilityCheckServiceInterface
}

func NewEligibilityCheckHandler(service services.EligibilityCheckServiceInterface) *EligibilityCheckHandler {
	return &EligibilityCheckHandler{service: service}
}

func (h *EligibilityCheckHandler) EligibilityCheck(c *gin.Context) {
	var body models.EligibilityCheckRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// msisdn, err := utils.ValidateAndFormat(body.MSISDN)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
	err := h.service.EligibilityCheck(c, body.MSISDN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}
