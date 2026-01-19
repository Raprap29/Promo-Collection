package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/downstreams/models"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)


type contextKey string

const LogDetailsKey contextKey = "logDetails"


func extractHeaders(headers map[string][]string) map[string]interface{} {
    result := make(map[string]interface{})
    for key, values := range headers {
        result[key] = values[0]
    }
    return maskSensitiveData(result, consts.SensitiveKeys)
}

func AttachRequestDetails() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestTime := time.Now().UTC()
        
        details := models.RequestDetails{
            RequestID:     uuid.New().String(),
            IP:           c.ClientIP(),
            UserAgent:    c.Request.UserAgent(),
            HTTPMethod:   c.Request.Method,
            Path:         c.Request.URL.String(),
            OperationName: extractFirstTwoSegments(c.HandlerName()),
            RequestTime:  requestTime.Format(time.RFC3339Nano),
            RequestParams: map[string]interface{}{
                "headers": extractHeaders(c.Request.Header),
                "payload": map[string]interface{}{},
                "query":   c.Request.URL.Query(),
            },
        }

        c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), LogDetailsKey, details))
        c.Next()

        details.Status = c.Writer.Status()
        details.ResponseTime = time.Now().UTC().Format(time.RFC3339Nano)
        details.ResponseParams = map[string]interface{}{
            "headers": extractHeaders(c.Writer.Header()),
            "body":    map[string]interface{}{},
        }

        logMessage, err := json.Marshal(details)
        if err != nil {
            log.Fatalf("Error encoding log message to JSON: %v", err)
        }
        fmt.Println(string(logMessage))
    }
}



func maskSensitiveData(data map[string]interface{}, keysToMask []string) map[string]interface{} {
	maskedData := make(map[string]interface{})
	for key, value := range data {
		if contains(keysToMask, key) {
			maskedData[key] = "*****" 
		} else {
			if reflect.TypeOf(value).Kind() == reflect.Map {
				if nestedMap, ok := value.(map[string]interface{}); ok {
					maskedData[key] = maskSensitiveData(nestedMap, keysToMask)
				} else {
					maskedData[key] = value
				}
			} else {
				maskedData[key] = value
			}
		}
	}
	return maskedData
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}



func extractFirstTwoSegments(handlerName string) string {
	segments := strings.Split(handlerName, "/")
	if len(segments) > 2 {
		return strings.Join(segments[:2], "/")
	}
	return handlerName
}