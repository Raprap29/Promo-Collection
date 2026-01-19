package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var baseValidConfig = AppConfig{
	Server: ServerConfig{Port: 8080},
	Mongo: MongoConfig{
		URI:             "mongodb://localhost:27017",
		DBName:          "Loan_Prod",
		MinPoolSize:     5,
		MaxPoolSize:     20,
		MaxConnIdleTime: 25 * time.Minute,
		ConnectTimeout:  10 * time.Second,
	},
	Redis: RedisConfig{
		Addr:           "localhost:6379",
		Password:       "pass",
		DB:             1,
		EnableTLS:      true,
		ConnectTimeout: 5 * time.Second,
	},
	Kafka: KafkaConfig{
		Server:           "localhost:9092",
		PromoTopic:       "promo",
		SecurityProtocol: "PLAINTEXT",
		SASLMechanism:    "PLAIN",
		SASLUsername:     "user",
		SASLPassword:     "pass",
		SessionTimeoutMs: 12000,
		ClientID:         "client",
		GroupID:          "group",
	},
	PubSub: PubSubConfig{
		ProjectID:             "pid",
		Topic:                 "topic",
		MinBackoffSeconds:     6,
		MaxBackoffSeconds:     120,
		TopicMessageRetention: 10,
		DLQMessageRetention:   10,
		MaxDeliveryAttempts:   16,
	},
}

func writeTempConfig(t *testing.T, cfg AppConfig) string {
	t.Helper()
	data, _ := yaml.Marshal(cfg)
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmp, data, 0644))
	return tmp
}

func TestValidateConfigErrors(t *testing.T) {
	t.Run("min pool size too low", func(t *testing.T) {
		c := baseValidConfig
		c.Mongo.MinPoolSize = 1
		assert.Error(t, validateConfig(&c))
	})

	t.Run("max pool size too high", func(t *testing.T) {
		c := baseValidConfig
		c.Mongo.MaxPoolSize = 100
		assert.Error(t, validateConfig(&c))
	})

	t.Run("max conn idle time out of range", func(t *testing.T) {
		c := baseValidConfig
		c.Mongo.MaxConnIdleTime = 5 * time.Minute
		assert.Error(t, validateConfig(&c))
	})

	t.Run("kafka session timeout out of range", func(t *testing.T) {
		c := baseValidConfig
		c.Kafka.SessionTimeoutMs = 5000
		assert.Error(t, validateConfig(&c))
	})

	t.Run("pubsub min backoff invalid", func(t *testing.T) {
		c := baseValidConfig
		c.PubSub.MinBackoffSeconds = 1
		assert.Error(t, validateConfig(&c))
	})

	t.Run("pubsub max backoff invalid", func(t *testing.T) {
		c := baseValidConfig
		c.PubSub.MaxBackoffSeconds = 10
		assert.Error(t, validateConfig(&c))
	})

	t.Run("pubsub topic retention invalid", func(t *testing.T) {
		c := baseValidConfig
		c.PubSub.TopicMessageRetention = 1
		assert.Error(t, validateConfig(&c))
	})

	t.Run("pubsub dlq retention invalid", func(t *testing.T) {
		c := baseValidConfig
		c.PubSub.DLQMessageRetention = 1
		assert.Error(t, validateConfig(&c))
	})

	t.Run("pubsub max delivery attempts invalid", func(t *testing.T) {
		c := baseValidConfig
		c.PubSub.MaxDeliveryAttempts = 1
		assert.Error(t, validateConfig(&c))
	})
}

func TestGetEnvAsInt(t *testing.T) {
    t.Setenv("INT_KEY", "42")
    assert.Equal(t, 42, GetEnvOrDefaultAsInt("INT_KEY", 5))

    t.Setenv("INT_KEY", "invalid")
    assert.Equal(t, 5, GetEnvOrDefaultAsInt("INT_KEY", 5))

    os.Unsetenv("INT_KEY")
    assert.Equal(t, 5, GetEnvOrDefaultAsInt("INT_KEY", 5))
}

func TestLoadFromConfig(t *testing.T) {
	t.Run("valid config from env", func(t *testing.T) {
		path := writeTempConfig(t, baseValidConfig)
		t.Setenv("CONFIG_PATH", path)
		cfg, err := LoadFromConfig()
		require.NoError(t, err)
		assert.Equal(t, "Loan_Prod", cfg.Mongo.DBName)
	})

	t.Run("valid config from default path", func(t *testing.T) {
		path := writeTempConfig(t, baseValidConfig)
		// Create default path in a temp dir
		defaultDir := t.TempDir()
		defaultPath := filepath.Join(defaultDir, "configs", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(defaultPath), 0o755))
		require.NoError(t, os.Rename(path, defaultPath))

		// Temporarily change working dir
		oldWD, _ := os.Getwd()
		require.NoError(t, os.Chdir(defaultDir))
		defer os.Chdir(oldWD)

		cfg, err := LoadFromConfig()
		require.NoError(t, err)
		assert.Equal(t, "Loan_Prod", cfg.Mongo.DBName)
	})

	t.Run("nonexistent config file", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "/nonexistent/path/config.yaml")
		_, err := LoadFromConfig()
		assert.Error(t, err)
	})
}
