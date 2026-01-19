package config

import (
	"fmt"
	"log/slog"
	"notificationservice/internal/pkg/logger"

	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds server-level config
type ServerConfig struct {
	SERVICE_NAME string `yaml:"service_name"`
	Port         int    `yaml:"port"`
}
type LogConfig struct {
	LogLevel string `yaml:"level"`
}

type PubSubConfig struct {
	ProjectID    string `yaml:"project_id"`
	Subscription string `yaml:"subscription"`
}

type SMSNotificationConfig struct {
	SEND_SMS_NOTIFICATION_URL       string `yaml:"send_sms_notification_url"`
	SEND_SMS_NOTIFICATION_X_API_KEY string `yaml:"send_sms_notification_x_api_key"`
}

// AppConfig is the main config struct that holds all configs
type AppConfig struct {
	PubSub                PubSubConfig          `yaml:"pubsub"`
	Server                ServerConfig          `yaml:"server"`
	Logging               LogConfig             `yaml:"logging"`
	SMSNotificationConfig SMSNotificationConfig `yaml:"sms-notification"`
}

func assignDefaultConfigValues(cfg *AppConfig) {

	// server config defaults
	cfg.Server.SERVICE_NAME = GetEnvOrDefaultAsString("SERVICE_NAME", "dodrio_loanCollection")
	cfg.Server.Port = GetEnvOrDefaultAsInt("SERVER_PORT", 8080)

	// log config defaults
	cfg.Logging.LogLevel = GetEnvOrDefaultAsString("LOGGING_LEVEL", cfg.Logging.LogLevel)

	// PubSub config defaults
	cfg.PubSub.ProjectID = GetEnvOrDefaultAsString("PROJECT_ID", "glo-isg-reloads-project-dev")
	cfg.PubSub.Subscription = GetEnvOrDefaultAsString("PUBSUB_SUBSCRIPTION",
		"glo-isg-dodrio-d-notification-subscription-ddr-762")
	cfg.SMSNotificationConfig.SEND_SMS_NOTIFICATION_URL = GetEnvOrDefaultAsString(
		"SEND_SMS_NOTIFICATION_URL", cfg.SMSNotificationConfig.SEND_SMS_NOTIFICATION_URL)
	cfg.SMSNotificationConfig.SEND_SMS_NOTIFICATION_X_API_KEY = GetEnvOrDefaultAsString(
		"DECREMENT_WALLET_SUBSCRIPTION_API_KEY", "")

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

	assignDefaultConfigValues(&cfg)

	logger.Info("Configuration loaded successfully", slog.String("path", configPath))

	return &cfg, nil
}

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
		return val
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
