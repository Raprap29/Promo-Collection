package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRedisKeyBuilder_LoanKey(t *testing.T) {
	tests := []struct {
		name     string
		msisdn   string
		expected string
	}{
		{
			name:     "valid msisdn",
			msisdn:   "1234567890",
			expected: "skipTimestamps:1234567890",
		},
		{
			name:     "empty msisdn",
			msisdn:   "",
			expected: "skipTimestamps:",
		},
		{
			name:     "msisdn with special characters",
			msisdn:   "123-456-7890",
			expected: "skipTimestamps:123-456-7890",
		},
	}

	builder := NewRedisKeyBuilder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.LoanKey(tt.msisdn)
			if result != tt.expected {
				t.Errorf("LoanKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSessionData_JSONSerialization(t *testing.T) {
	now := time.Now()
	session := SessionData{
		UserID:      "user123",
		LoginTime:   now,
		Permissions: []string{"read", "write"},
		Metadata: map[string]interface{}{
			"role":  "admin",
			"level": 5,
		},
		ExpiresAt: now.Add(24 * time.Hour),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal SessionData: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SessionData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SessionData: %v", err)
	}

	// Verify data integrity
	if unmarshaled.UserID != session.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, session.UserID)
	}
	if len(unmarshaled.Permissions) != len(session.Permissions) {
		t.Errorf("Permissions length mismatch: got %v, want %v", len(unmarshaled.Permissions), len(session.Permissions))
	}
}

func TestRateLimitData_JSONSerialization(t *testing.T) {
	now := time.Now()
	rateLimit := RateLimitData{
		MSISDN:       "1234567890",
		CurrentCount: 5,
		Limit:        10,
		WindowStart:  now,
		WindowEnd:    now.Add(time.Hour),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(rateLimit)
	if err != nil {
		t.Fatalf("Failed to marshal RateLimitData: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled RateLimitData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal RateLimitData: %v", err)
	}

	// Verify data integrity
	if unmarshaled.MSISDN != rateLimit.MSISDN {
		t.Errorf("MSISDN mismatch: got %v, want %v", unmarshaled.MSISDN, rateLimit.MSISDN)
	}
	if unmarshaled.CurrentCount != rateLimit.CurrentCount {
		t.Errorf("CurrentCount mismatch: got %v, want %v", unmarshaled.CurrentCount, rateLimit.CurrentCount)
	}
}

func TestCacheMetadata_JSONSerialization(t *testing.T) {
	now := time.Now()
	cache := CacheMetadata{
		Key:          "test-key",
		Type:         "loan",
		CachedAt:     now,
		ExpiresAt:    now.Add(time.Hour),
		HitCount:     42,
		LastAccessed: now.Add(30 * time.Minute),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal CacheMetadata: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled CacheMetadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CacheMetadata: %v", err)
	}

	// Verify data integrity
	if unmarshaled.Key != cache.Key {
		t.Errorf("Key mismatch: got %v, want %v", unmarshaled.Key, cache.Key)
	}
	if unmarshaled.HitCount != cache.HitCount {
		t.Errorf("HitCount mismatch: got %v, want %v", unmarshaled.HitCount, cache.HitCount)
	}
}

func TestRedisKeyPattern(t *testing.T) {
	expectedPattern := "skipTimestamp:%s"
	if LoanKeyPattern != expectedPattern {
		t.Errorf("LoanKeyPattern = %v, want %v", LoanKeyPattern, expectedPattern)
	}
}

func TestNewRedisKeyBuilder(t *testing.T) {
	builder := NewRedisKeyBuilder()
	if builder == nil {
		t.Error("NewRedisKeyBuilder() returned nil")
	}
}

func TestRedisOperations_Interface(t *testing.T) {
	// Test that RedisOperations struct can be used as an interface
	var ops RedisOperations

	// Mock implementations
	ops.Set = func(key string, value interface{}, expiration time.Duration) error {
		return nil
	}
	ops.Get = func(key string) (interface{}, error) {
		return "test-value", nil
	}
	ops.Delete = func(key string) error {
		return nil
	}
	ops.Exists = func(key string) (bool, error) {
		return true, nil
	}
	ops.Expire = func(key string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ops.TTL = func(key string) (time.Duration, error) {
		return time.Hour, nil
	}

	// Test the operations
	err := ops.Set("test-key", "test-value", time.Hour)
	if err != nil {
		t.Errorf("Set operation failed: %v", err)
	}

	value, err := ops.Get("test-key")
	if err != nil {
		t.Errorf("Get operation failed: %v", err)
	}
	if value != "test-value" {
		t.Errorf("Get returned %v, want test-value", value)
	}

	exists, err := ops.Exists("test-key")
	if err != nil {
		t.Errorf("Exists operation failed: %v", err)
	}
	if !exists {
		t.Error("Exists returned false, want true")
	}
}
