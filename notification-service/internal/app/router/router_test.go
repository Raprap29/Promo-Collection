package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test that SetupRouter returns a non-nil router and registers the health check route
func TestSetupRouterHealthCheckRoute(t *testing.T) {
	ctx := context.Background()
	router := SetupRouter(ctx)
	assert.NotNil(t, router)

	// Use httptest to simulate a GET request to the health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/IntegrationServices/Dodrio/NotificationService/HealthCheck", nil)
	router.ServeHTTP(w, req)

	// Expect a 200 OK status code from the health check handler
	assert.Equal(t, 200, w.Code)
}

// Test that SetupRouter works with different context types
func TestSetupRouterWithDifferentContexts(t *testing.T) {
	// With timeout context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	router1 := SetupRouter(timeoutCtx)
	assert.NotNil(t, router1)

	// With value context
	type ctxKey string
	const testKey ctxKey = "testKey"

	valueCtx := context.WithValue(context.Background(), testKey, "value")
	router2 := SetupRouter(valueCtx)
	assert.NotNil(t, router2)
}

// Test that SetupRouter can be called concurrently
func TestSetupRouterConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			router := SetupRouter(ctx)
			assert.NotNil(t, router)
		}()
	}
	wg.Wait()
}
