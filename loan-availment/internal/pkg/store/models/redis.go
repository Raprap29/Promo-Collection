package models

import (
	"fmt"
	"time"
)

type RedisSchema struct{}

// Key Patterns for different data types
const (
	LoanKeyPattern = "skipTimestamps:%s" // SkipTimestamp:MSISDN

)

type RedisKeyBuilder struct{}

func NewRedisKeyBuilder() *RedisKeyBuilder {
	return &RedisKeyBuilder{}
}

func (rkb *RedisKeyBuilder) LoanKey(msisdn string) string {
	return fmt.Sprintf(LoanKeyPattern, msisdn)
}

type RedisDataStructures struct{}

type SessionData struct {
	UserID      string                 `json:"user_id"`
	LoginTime   time.Time              `json:"login_time"`
	Permissions []string               `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

type RateLimitData struct {
	MSISDN       string    `json:"msisdn"`
	CurrentCount int64     `json:"current_count"`
	Limit        int64     `json:"limit"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
}
type CacheMetadata struct {
	Key          string    `json:"key"`
	Type         string    `json:"type"`
	CachedAt     time.Time `json:"cached_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	HitCount     int64     `json:"hit_count"`
	LastAccessed time.Time `json:"last_accessed"`
}

type RedisOperations struct {
	Set    func(key string, value interface{}, expiration time.Duration) error
	Get    func(key string) (interface{}, error)
	Delete func(key string) error
	Exists func(key string) (bool, error)

	// Time-based operations
	Expire func(key string, expiration time.Duration) (bool, error)
	TTL    func(key string) (time.Duration, error)
}
