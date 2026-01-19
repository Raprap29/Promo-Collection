// go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckHandlerHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewHealthCheckHandler()
	router.GET("/health", handler.HealthCheck)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Health Check"}`, w.Body.String())
}