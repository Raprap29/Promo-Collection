package config

import (
	"fmt"
	"log/slog"
	"os"
	"promo-collection-worker/internal/pkg/logger"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds server-level config
type ServerConfig struct {
	Port int `yaml:"port"`
}

type LogConfig struct {
	LogLevel string `yaml:"level"`
}

type HIPConfig struct {
	DecrementWalletSubscriptionURL       string        `yaml:"decrement_wallet_subscription_url"`
	DecrementWalletSubscriptionAPIKey    string        `yaml:"decrement_wallet_subscription_api_key"`
	HTTPTimeout                          time.Duration `yaml:"http_timeout_seconds"`
	RetrieveSubscriberUsageURL           string        `yaml:"retrieve_subscriber_usage_url"`
	RetrieveSubscriberUsageAPIKey        string        `yaml:"retrieve_subscriber_usage_api_key"`
	RetrieveSubscriberUsageSourceChannel string        `yaml:"retrieve_subscriber_usage_source_channel"`
}

// MongoDB connection config
type MongoConfig struct {
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	URI             string        `yaml:"uri"`
	DBName          string        `yaml:"db_name"`
	MaxPoolSize     uint64        `yaml:"max_pool_size"`
	MinPoolSize     uint64        `yaml:"min_pool_size"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_minutes"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout_seconds"`
}

// Redis connection config
type RedisConfig struct {
	Addr           string        `yaml:"addr"`
	Password       string        `yaml:"password"`
	DB             int           `yaml:"db"`
	EnableTLS      bool          `yaml:"enable_tls"`
	ConnectTimeout time.Duration `yaml:"connect_timeout_seconds"`
	CertContent    string        `yaml:"cert_content"`
}

// Kafka connection config
type KafkaConfig struct {
	Server           string `yaml:"server"`
	PromoTopic       string `yaml:"promo_topic"`
	SecurityProtocol string `yaml:"security_protocol"`
	SASLMechanism    string `yaml:"sasl_mechanism"`
	SASLUsername     string `yaml:"sasl_username"`
	SASLPassword     string `yaml:"sasl_password"`
	SessionTimeoutMs int    `yaml:"session_timeout_ms"`
	ClientID         string `yaml:"client_id"`
	GroupID          string `yaml:"group_id"`
}

type PubSubConfig struct {
	ProjectID             string `yaml:"project_id"`
	Topic                 string `yaml:"topic"`
	Subscription          string `yaml:"subscription"`
	NotificationTopic     string `yaml:"notification_topic"`
	MinBackoffSeconds     int    `yaml:"min_backoff_seconds"`
	MaxBackoffSeconds     int    `yaml:"max_backoff_seconds"`
	TopicMessageRetention int    `yaml:"topic_msg_retention"`
	DLQMessageRetention   int    `yaml:"dlq_msg_retention"`
	MaxDeliveryAttempts   int    `yaml:"max_delivery_attempts"`
}

type KafkaRetryServiceConfig struct {
	RetryStartDate string        `yaml:"retry_start_date"`
	WorkerCount    int           `yaml:"worker_count"`
	BufferSize     int           `yaml:"buffer_size"`
	MaxBatchSize   int           `yaml:"max_batch_size"`
	MongoBatchSize int32         `yaml:"mongo_batch_size"`
	FlushInterval  time.Duration `yaml:"flush_interval"`
}

type GCSConfig struct {
	BucketName string `yaml:"bucket_name"`
}

// AppConfig is the main config struct that holds all configs
type AppConfig struct {
	Server            ServerConfig            `yaml:"server"`
	Mongo             MongoConfig             `yaml:"mongo"`
	Redis             RedisConfig             `yaml:"redis"`
	Kafka             KafkaConfig             `yaml:"kafka"`
	PubSub            PubSubConfig            `yaml:"pubsub"`
	Logging           LogConfig               `yaml:"logging"`
	HIP               HIPConfig               `yaml:"hip"`
	KafkaRetryService KafkaRetryServiceConfig `yaml:"kafka_retry_service"`
	GCS               GCSConfig               `yaml:"gcs"`
}

// nolint: funlen
func assignDefaultConfigValues(cfg *AppConfig) *AppConfig {

	// server config defaults
	cfg.Server.Port = GetEnvOrDefaultAsInt("SERVER_PORT", 8080)

	// log config defaults
	cfg.Logging.LogLevel = GetEnvOrDefaultAsString("LOGGING_LEVEL", cfg.Logging.LogLevel)

	// MongoDB config defaults
	cfg.Mongo.URI = GetEnvOrDefaultAsString(
		"MONGO_URI",
		cfg.Mongo.URI,
	)
	cfg.Mongo.DBName = GetEnvOrDefaultAsString("MONGO_DB_NAME", cfg.Mongo.DBName)
	cfg.Mongo.Username = GetEnvOrDefaultAsString("MONGO_USERNAME", cfg.Mongo.Username)
	cfg.Mongo.Password = GetEnvOrDefaultAsString("MONGO_PASSWORD", cfg.Mongo.Password)
	cfg.Mongo.MaxPoolSize = GetEnvOrDefaultAsUint64("MONGO_MAX_POOL_SIZE", cfg.Mongo.MaxPoolSize)
	cfg.Mongo.MinPoolSize = GetEnvOrDefaultAsUint64("MONGO_MIN_POOL_SIZE", cfg.Mongo.MinPoolSize)
	cfg.Mongo.MaxConnIdleTime = time.Duration(GetEnvOrDefaultAsInt("MONGO_MAX_CONN_IDLE_MINUTES", 30)) * time.Minute
	cfg.Mongo.ConnectTimeout = time.Duration(GetEnvOrDefaultAsInt("MONGO_CONNECT_TIMEOUT_SECONDS", 10)) * time.Second

	// Redis config defaults
	cfg.Redis.Addr = GetEnvOrDefaultAsString("REDIS_ADDR", cfg.Redis.Addr)
	cfg.Redis.Password = GetEnvOrDefaultAsString("REDIS_PASSWORD", cfg.Redis.Password)
	cfg.Redis.DB = GetEnvOrDefaultAsInt("REDIS_DB", cfg.Redis.DB)
	cfg.Redis.EnableTLS = GetEnvOrDefaultAsInt("REDIS_ENABLE_TLS", 1) == 1
	cfg.Redis.ConnectTimeout = time.Duration(GetEnvOrDefaultAsInt("REDIS_CONNECT_TIMEOUT_SECONDS", 10)) * time.Second
	cfg.Redis.CertContent = GetEnvOrDefaultAsString("REDIS_TLS_CERT", "")

	// Kafka config defaults
	cfg.Kafka.Server = GetEnvOrDefaultAsString("KAFKA_SERVER", cfg.Kafka.Server)
	cfg.Kafka.PromoTopic = GetEnvOrDefaultAsString("KAFKA_PROMO_TOPIC", cfg.Kafka.PromoTopic)
	cfg.Kafka.SecurityProtocol = GetEnvOrDefaultAsString("KAFKA_SECURITY_PROTOCOL", cfg.Kafka.SecurityProtocol)
	cfg.Kafka.SASLMechanism = GetEnvOrDefaultAsString("KAFKA_SASL_MECHANISM", cfg.Kafka.SASLMechanism)
	cfg.Kafka.SASLUsername = GetEnvOrDefaultAsString("KAFKA_SASL_USERNAME", cfg.Kafka.SASLUsername)
	cfg.Kafka.SASLPassword = GetEnvOrDefaultAsString("KAFKA_SASL_PASSWORD", cfg.Kafka.SASLPassword)
	cfg.Kafka.SessionTimeoutMs = GetEnvOrDefaultAsInt("KAFKA_SESSION_TIMEOUT_MS", 15000)
	cfg.Kafka.ClientID = GetEnvOrDefaultAsString(
		"KAFKA_CLIENT_ID",
		cfg.Kafka.ClientID,
	)
	cfg.Kafka.GroupID = GetEnvOrDefaultAsString("KAFKA_GROUP_ID", cfg.Kafka.GroupID)

	// PubSub config defaults
	cfg.PubSub.ProjectID = GetEnvOrDefaultAsString("PROJECT_ID", cfg.PubSub.ProjectID)
	cfg.PubSub.Topic = GetEnvOrDefaultAsString("PUBSUB_TOPIC", cfg.PubSub.Topic)
	cfg.PubSub.Subscription = GetEnvOrDefaultAsString("PUBSUB_WORKER_SUBSCRIPTION",
		cfg.PubSub.Subscription)
	cfg.PubSub.NotificationTopic = GetEnvOrDefaultAsString("PUBSUB_NOTIFICATION_TOPIC",
		cfg.PubSub.NotificationTopic)
	cfg.PubSub.MinBackoffSeconds = GetEnvOrDefaultAsInt("PUBSUB_MIN_BACKOFF_SECONDS", 10)
	cfg.PubSub.MaxBackoffSeconds = GetEnvOrDefaultAsInt("PUBSUB_MAX_BACKOFF_SECONDS", 300)
	cfg.PubSub.TopicMessageRetention = GetEnvOrDefaultAsInt("PUBSUB_TOPIC_MSG_RETENTION", 7)
	cfg.PubSub.DLQMessageRetention = GetEnvOrDefaultAsInt("PUBSUB_DLQ_MSG_RETENTION", 7)
	cfg.PubSub.MaxDeliveryAttempts = GetEnvOrDefaultAsInt("PUBSUB_MAX_DELIVERY_ATTEMPTS", 15)

	// HIP config defaults
	cfg.HIP.DecrementWalletSubscriptionURL = GetEnvOrDefaultAsString("DECREMENT_WALLET_SUBSCRIPTION_URL",
		cfg.HIP.DecrementWalletSubscriptionURL)
	cfg.HIP.DecrementWalletSubscriptionAPIKey = GetEnvOrDefaultAsString("DECREMENT_WALLET_SUBSCRIPTION_API_KEY",
		cfg.HIP.DecrementWalletSubscriptionAPIKey)
	cfg.HIP.HTTPTimeout = time.Duration(GetEnvOrDefaultAsInt("HTTPTimeout", 10)) * time.Second
	cfg.HIP.RetrieveSubscriberUsageURL = GetEnvOrDefaultAsString("RETRIEVE_SUBSCRIBER_USAGE_URL",
		cfg.HIP.RetrieveSubscriberUsageURL)
	cfg.HIP.RetrieveSubscriberUsageAPIKey = GetEnvOrDefaultAsString("RETRIEVE_SUBSCRIBER_USAGE_API_KEY",
		cfg.HIP.RetrieveSubscriberUsageAPIKey)
	cfg.HIP.RetrieveSubscriberUsageSourceChannel = GetEnvOrDefaultAsString("RETRIEVE_SUBSCRIBER_USAGE_SOURCE_CHANNEL",
		cfg.HIP.RetrieveSubscriberUsageSourceChannel)

	cfg.KafkaRetryService.RetryStartDate = GetEnvOrDefaultAsString("RETRY_START_DATE",
		cfg.KafkaRetryService.RetryStartDate)
	cfg.KafkaRetryService.WorkerCount = GetEnvOrDefaultAsInt("WORKER_COUNT", cfg.KafkaRetryService.WorkerCount)
	cfg.KafkaRetryService.BufferSize = GetEnvOrDefaultAsInt("BUFFER_SIZE", cfg.KafkaRetryService.BufferSize)
	cfg.KafkaRetryService.MaxBatchSize = GetEnvOrDefaultAsInt("MAX_BATCH_SIZE", cfg.KafkaRetryService.MaxBatchSize)
	cfg.KafkaRetryService.MongoBatchSize = GetEnvOrDefaultAsInt32("MONGO_BATCH_SIZE", cfg.KafkaRetryService.MongoBatchSize)
	cfg.KafkaRetryService.FlushInterval = time.Duration(GetEnvOrDefaultAsInt("FLUSH_INTERVAL", 500)) * time.Millisecond

	cfg.GCS.BucketName = GetEnvOrDefaultAsString("GCS_BUCKET_NAME", cfg.GCS.BucketName)
	return cfg
}

// LoadConfig loads and parses config file into AppConfig
func LoadFromConfigFilePath(configPath string) (*AppConfig, error) {

	// #nosec G304: configPath is validated above to prevent path traversal
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("Failed to read config file", err, slog.String("path", configPath))
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("Failed to unmarshal config", err)
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	defaultCfg := assignDefaultConfigValues(&cfg)

	if err := validateConfig(defaultCfg); err != nil {
		logger.Error("Config validation failed", err)
		return nil, err
	}

	logger.Info("Configuration loaded successfully", slog.String("path", configPath))

	return defaultCfg, nil
}

func validateConfig(cfg *AppConfig) error {
	if err := validateMongoConfig(cfg.Mongo); err != nil {
		return err
	}
	if err := validateKafkaConfig(cfg.Kafka); err != nil {
		return err
	}
	if err := validatePubSubConfig(cfg.PubSub); err != nil {
		return err
	}
	return nil
}

func validateMongoConfig(mongo MongoConfig) error {
	if mongo.MinPoolSize < 5 || mongo.MinPoolSize > 10 {
		return fmt.Errorf(
			"mongo.min_pool_size must be between 5 and 10, got %d",
			mongo.MinPoolSize,
		)
	}

	if mongo.MaxPoolSize < 10 || mongo.MaxPoolSize > 50 {
		return fmt.Errorf(
			"mongo.max_pool_size must be between 10 and 50, got %d",
			mongo.MaxPoolSize,
		)
	}

	minIdle := 20 * time.Minute
	maxIdle := 30 * time.Minute
	if mongo.MaxConnIdleTime < minIdle || mongo.MaxConnIdleTime > maxIdle {
		return fmt.Errorf(
			"mongo.max_conn_idle_minutes must be between %v and %v, got %v",
			minIdle,
			maxIdle,
			mongo.MaxConnIdleTime,
		)
	}

	return nil
}

func validateKafkaConfig(kafka KafkaConfig) error {
	if kafka.SessionTimeoutMs < 10000 || kafka.SessionTimeoutMs > 15000 {
		return fmt.Errorf(
			"kafka.session_timeout_ms must be between 10000 and 15000 ms, got %d",
			kafka.SessionTimeoutMs,
		)
	}
	return nil
}

func validatePubSubConfig(pubsub PubSubConfig) error {
	if pubsub.MinBackoffSeconds < 5 || pubsub.MinBackoffSeconds > 10 {
		return fmt.Errorf("pubsub.min_backoff_seconds must be between 5 and 10 seconds, got %d", pubsub.MinBackoffSeconds)
	}
	if pubsub.MaxBackoffSeconds < 60 || pubsub.MaxBackoffSeconds > 300 {
		return fmt.Errorf("pubsub.max_backoff_seconds must be between 60 and 300 seconds, got %d", pubsub.MaxBackoffSeconds)
	}
	// topic_msg_retention and dlq_msg_retention are in days (assumed int)
	if pubsub.TopicMessageRetention < 7 || pubsub.TopicMessageRetention > 14 {
		return fmt.Errorf("pubsub.topic_msg_retention must be between 7 and 14 days, got %d", pubsub.TopicMessageRetention)
	}
	if pubsub.DLQMessageRetention < 7 || pubsub.DLQMessageRetention > 14 {
		return fmt.Errorf("pubsub.dlq_msg_retention must be between 7 and 14 days, got %d", pubsub.DLQMessageRetention)
	}
	if pubsub.MaxDeliveryAttempts < 15 || pubsub.MaxDeliveryAttempts > 20 {
		return fmt.Errorf("pubsub.max_delivery_attempts must be between 15 and 20, got %d", pubsub.MaxDeliveryAttempts)
	}

	return nil
}

// GetEnvOrDefaultAsInt returns the value of the given env variable
// as an int or the default value if not set or invalid.
func GetEnvOrDefaultAsInt(key string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return int(value)
}

// GetEnvOrDefault returns the value of the given env variable or the default value if not set.
func GetEnvOrDefaultAsString(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		if val != "" {
			return val
		}
	}
	return defaultVal
}

// LoadEnv loads environment variables from a .env file and loads the config file path.
func LoadFromConfig() (*AppConfig, error) {
	configPath := GetEnvOrDefaultAsString("CONFIG_PATH", "configs/config.yaml")

	// Load the actual config file
	cfg, err := LoadFromConfigFilePath(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	return cfg, nil
}

func GetEnvOrDefaultAsUint64(key string, defaultValue uint64) uint64 {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func GetEnvOrDefaultAsInt32(key string, defaultValue int32) int32 {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(value)
}
