
package middleware

import (
    "time"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/semconv/v1.17.0"
    "github.com/gin-gonic/gin"
)

func NewMetricMiddleware(meter metric.Meter) gin.HandlerFunc {

    durationHistogram, _ := meter.Int64Histogram(
        "http.server.latency",
        metric.WithUnit("ms"),
        metric.WithDescription("The latency of HTTP requests."),
    )
    
    // Counter for the total number of requests
    requestCounter, _ := meter.Int64Counter(
        "http.server.requests_total",
        metric.WithDescription("The total number of HTTP requests."),
    )
    
    // Counter for successful requests
    successCounter, _ := meter.Int64Counter(
        "http.server.success_requests_total",
        metric.WithDescription("The total number of successful HTTP requests."),
    )
    
    // Counter for failed requests
    errorCounter, _ := meter.Int64Counter(
        "http.server.error_requests_total",
        metric.WithDescription("The total number of failed HTTP requests."),
    )
    
    // Histogram for request size
    requestSizeHistogram, _ := meter.Int64Histogram(
        "http.server.request_size_bytes",
        metric.WithUnit("bytes"),
        metric.WithDescription("The size of HTTP requests in bytes."),
    )
    
    // Histogram for response size
    responseSizeHistogram, _ := meter.Int64Histogram(
        "http.server.response_size_bytes",
        metric.WithUnit("bytes"),
        metric.WithDescription("The size of HTTP responses in bytes."),
    )

    return func(c *gin.Context) {
        startTime := time.Now()
        
        requestSize := c.Request.ContentLength

        c.Next()

        duration := time.Since(startTime).Milliseconds()

        // Extract route, method, and status code
        path := c.FullPath()
        method := c.Request.Method
        statusCode := c.Writer.Status()

        // Get response size
        responseSize := c.Writer.Size()

        // Create metric attributes for each request
        attributes := []attribute.KeyValue{
            semconv.HTTPRouteKey.String(path),
            semconv.HTTPMethodKey.String(method),
            semconv.HTTPStatusCodeKey.Int(statusCode),
            attribute.String("http.client_ip", c.ClientIP()), 
        }

        // Record the request duration
        durationHistogram.Record(
            c.Request.Context(),
            duration,
            metric.WithAttributes(attributes...),
        )

        // Record the request count
        requestCounter.Add(
            c.Request.Context(),
            1,
            metric.WithAttributes(attributes...),
        )

        // Record the request size
        requestSizeHistogram.Record(
            c.Request.Context(),
            requestSize,
            metric.WithAttributes(attributes...),
        )

        // Record the response size
        responseSizeHistogram.Record(
            c.Request.Context(),
            int64(responseSize),
            metric.WithAttributes(attributes...),
        )

        // Increment success or error counter based on status code
        if statusCode >= 200 && statusCode < 400 {
            successCounter.Add(
                c.Request.Context(),
                1,
                metric.WithAttributes(attributes...),
            )
        } else {
            errorCounter.Add(
                c.Request.Context(),
                1,
                metric.WithAttributes(attributes...),
            )
        }
    }
}
