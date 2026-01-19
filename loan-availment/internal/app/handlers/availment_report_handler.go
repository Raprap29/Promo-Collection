package handlers

import (
	"context"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"

	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/services"
	"globe/dodrio_loan_availment/internal/pkg/store"
)

type AvailmentHandler struct {
	service *services.AvailmentReportService
}

// NewAvailmentHandler creates a new instance of AvailmentHandler
func NewAvailmentHandler(bucketName string) *AvailmentHandler {
	// Create the GCS client
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		logger.Error("Failed to create GCS client: %v", err)
		return nil
	}
	return &AvailmentHandler{
		service: services.NewAvailmentService(gcsClient, bucketName, store.NewAvailmentReportRepository()),
	}
}

func (h *AvailmentHandler) AvailmentsReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": consts.SuccessProcessingMessageRevAvailmentReport})

	go func() {
		dynamicStartDay := c.Query("dynamicStartDay")
		_, err := h.service.AvailmentDetailsReports(context.Background(), dynamicStartDay)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}()

}
