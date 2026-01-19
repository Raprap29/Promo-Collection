package router

import (
	"context"
	"notificationservice/internal/app/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter(ctx context.Context) *gin.Engine {
	server := gin.Default()

	healthCheckHandler := handlers.NewHealthCheckHandler()
	server.GET("/IntegrationServices/Dodrio/NotificationService/HealthCheck", healthCheckHandler.HealthCheck)

	return server
}
