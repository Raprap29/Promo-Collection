package configs

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	SERVER_PORT                              string
	WORKER_POOL                              string
	DB_URI                                   string
	DB_NAME                                  string
	UUP_SERVICE_URI_GET_DETAILS_BY_ATTRIBUTE string
	AMAX_CREATE_SESSION_URL                  string
	AMAX_LOGIN_SESSION_URL                   string
	AMAX_TOPUP_URL                           string
	AMAX_X_API_KEY                           string
	LOGIN_SESSION_USER                       string
	LOGIN_SESSION_PASSWORD                   string
	SESSION_KEY                              string
	SESSION_NAME                             string
	KAFKA_SERVER                             string
	KAFKA_SECURITY_PROTOCOL                  string
	KAFKA_SASL_MECHANISM                     string
	KAFKA_SASL_USERNAME                      string
	KAFKA_SASL_PASSWORD                      string
	KAFKA_SESSION_TIMEOUT_MS                 int
	KAFKA_CLIENT_ID                          string
	KAFKA_TOPIC                              string
	DB_MAXPOOLSIZE                           uint64
	DB_MINPOOLSIZE                           uint64
	DB_MAXIDLETIME_INMINUTES                 int
	SFTP_USER                                string
	SFTP_PASSWORD                            string
	SFTP_HOST                                string
	SFTP_PORT                                string
	BUCKET_NAME                              string
	JSON_FILE_PATH                           string
	DIRECTORY_PATH                           string
	SFTP_REMOTE_FILE_PATH                    string
	REPORT_FOR_LAST_X_HOURS                  string
	UUP_CLIENT_ID                            string
	UUP_CLIENT_SECRET                        string
	UUP_API_KEY                              string
	UUP_USERNAME                             string
	UUP_RESOURCE_TYPE                        string
	SMS_SOURCE_ADDRESS                       string
	KAFKA_RETRY_DURATION                     int
	AVAILMENT_TRANSACTION_TTL_IN_HOURS       int
	UUP_ACCESSTOKEN_URL                      string
	UUP_X_API_KEY                            string
	LOAN_STATUS_KEYWORD                      []string
	LOAN_FOUND_MESSAGE_ID                    string
	NO_LOAN_FOUND_MESSAGE_ID                 string
	TIMEOUT_IN_SECONDS                       int
	SERVICE_NAME                             string
	OTEL_URL                                 string
	LOG_LEVEL                                string
	ASSURANCE_REPORT_DESTINATION_FOLDER      string
	REPORT_EVERY_X_HOURS                     int
	REPORT_CHUNKS                            int
	AMAX_CA_CERTIFICATE                      string
	AMAX_CA_CERTIFICATE_REQUIRED             bool
	HIP_CA_CERTIFICATE                       string
	HIP_CA_CERTIFICATE_REQUIRED              bool
	REDIS_ADDR                               string
	REDIS_PASSWORD                           string
	REDIS_DB                                 int
	REDIS_ENABLE_TLS                         bool
	REDIS_CONNECT_TIMEOUT_SECONDS            int
	REDIS_CERT_CONTENT                       string
	PROJECT_ID                               string
	PUBSUB_TOPIC                             string
	PUBSUB_ENABLED                           bool
)

type RedisConfig struct {
	Addr           string        `yaml:"addr"`
	Password       string        `yaml:"password"`
	DB             int           `yaml:"db"`
	EnableTLS      bool          `yaml:"enable_tls"`
	ConnectTimeout time.Duration `yaml:"connect_timeout_seconds"`
	CertContent    string        `yaml:"cert_content"`
}

// PubSubConfig represents the Pub/Sub configuration
type PubSubConfig struct {
	ProjectID string `yaml:"project_id"`
	Topic     string `yaml:"topic"`
	Enabled   bool   `yaml:"enabled"`
}

// LoadEnv loads the environment variables from a .env file
func LoadEnv() error {
	err := godotenv.Load("./../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	LoadEnvValues()

	return nil
}

func LoadEnvValues() {
	SERVER_PORT = GetEnv("SERVER_PORT", "8080")
	WORKER_POOL = GetEnv("WORKER_POOL", "5")
	DB_URI = GetEnv("DB_URI", "mongodb+srv://dodrio:dodrio@cluster0.2p7dmcv.mongodb.net/db?retryWrites=true&w=majority&appName=Cluster0")
	DB_NAME = GetEnv("DB_NAME", "AvailmentDemo")
	UUP_SERVICE_URI_GET_DETAILS_BY_ATTRIBUTE = GetEnv("UUP_SERVICE_URI_GET_DETAILS_BY_ATTRIBUTE", "")
	UUP_X_API_KEY = GetEnv("UUP_X_API_KEY", "")
	UUP_ACCESSTOKEN_URL = GetEnv("UUP_ACCESSTOKEN_URL", "")
	TIMEOUT_IN_SECONDS_str := GetEnv("TIMEOUT_IN_SECONDS", "20")
	TIMEOUT_IN_SECONDS, _ = strconv.Atoi(TIMEOUT_IN_SECONDS_str)
	SESSION_KEY = GetEnv("SESSION_KEY", "")
	SESSION_NAME = GetEnv("SESSION_NAME", "")
	AMAX_CREATE_SESSION_URL = GetEnv("AMAX_CREATE_SESSION_URL", "")
	AMAX_LOGIN_SESSION_URL = GetEnv("AMAX_LOGIN_SESSION_URL", "")
	AMAX_TOPUP_URL = GetEnv("AMAX_TOPUP_URL", "")
	AMAX_CA_CERTIFICATE = GetEnv("AMAX_CA_CERTIFICATE", "")
	AMAX_CA_CERTIFICATE_REQUIRED, _ = strconv.ParseBool(GetEnv("AMAX_CA_CERTIFICATE_REQUIRED", "false"))
	HIP_CA_CERTIFICATE = GetEnv("HIP_CA_CERTIFICATE", "")
	HIP_CA_CERTIFICATE_REQUIRED, _ = strconv.ParseBool(GetEnv("HIP_CA_CERTIFICATE_REQUIRED", "false"))
	LOGIN_SESSION_USER = GetEnv("LOGIN_SESSION_USER", "")
	LOGIN_SESSION_PASSWORD = GetEnv("LOGIN_SESSION_PASSWORD", "")
	KAFKA_SERVER = GetEnv("KAFKA_SERVER", "")
	KAFKA_SECURITY_PROTOCOL = GetEnv("KAFKA_SECURITY_PROTOCOL", "")
	KAFKA_SASL_MECHANISM = GetEnv("KAFKA_SASL_MECHANISM", "")
	KAFKA_SASL_USERNAME = GetEnv("KAFKA_SASL_USERNAME", "")
	KAFKA_SASL_PASSWORD = GetEnv("KAFKA_SASL_PASSWORD", "")
	KAFKA_SESSION_TIMEOUT_MS, _ = strconv.Atoi(GetEnv("KAFKA_SESSION_TIMEOUT_MS", ""))
	KAFKA_CLIENT_ID = GetEnv("KAFKA_CLIENT_ID", "")
	KAFKA_TOPIC = GetEnv("KAFKA_TOPIC", "")
	KAFKA_RETRY_DURATION_Str := GetEnv("KAFKA_RETRY_DURATION", "12")
	KAFKA_RETRY_DURATION, _ = strconv.Atoi(KAFKA_RETRY_DURATION_Str)
	DB_MAXPOOLSIZE_Str := GetEnv("DB_MAXPOOLSIZE", "100")
	DB_MINPOOLSIZE_Str := GetEnv("DB_MINPOOLSIZE", "10")
	DB_MAXIDLETIME_INMINUTES_Str := GetEnv("DB_MAXIDLETIME_INMINUTES", "5")
	SMS_SOURCE_ADDRESS = GetEnv("SMS_SOURCE_ADDRESS", "")
	DB_MAXPOOLSIZE, _ = strconv.ParseUint(DB_MAXPOOLSIZE_Str, 10, 64)
	DB_MINPOOLSIZE, _ = strconv.ParseUint(DB_MINPOOLSIZE_Str, 10, 64)
	DB_MAXIDLETIME_INMINUTES, _ = strconv.Atoi(DB_MAXIDLETIME_INMINUTES_Str)
	SFTP_USER = GetEnv("SFTP_USER", "")
	SFTP_PASSWORD = GetEnv("SFTP_PASSWORD", "")
	SFTP_HOST = GetEnv("SFTP_HOST", "")
	SFTP_PORT = GetEnv("SFTP_PORT", "")
	SFTP_REMOTE_FILE_PATH = GetEnv("SFTP_REMOTE_FILE_PATH", "")

	REPORT_FOR_LAST_X_HOURS = GetEnv("REPORT_FOR_LAST_X_HOURS", "")

	BUCKET_NAME = GetEnv("BUCKET_NAME", "")
	JSON_FILE_PATH = GetEnv("JSON_FILE_PATH", "")
	ASSURANCE_REPORT_DESTINATION_FOLDER = GetEnv("ASSURANCE_REPORT_DESTINATION_FOLDER", "availmentAssuranceReport")
	DIRECTORY_PATH = GetEnv("DIRECTORY_PATH_FOR_AVAILMENT_REPORT", "/availmentReport")

	UUP_CLIENT_ID = GetEnv("UUP_CLIENT_ID", "")
	UUP_CLIENT_SECRET = GetEnv("UUP_CLIENT_SECRET", "")
	UUP_API_KEY = GetEnv("UUP_API_KEY", "")
	UUP_USERNAME = GetEnv("UUP_USERNAME", "")
	UUP_RESOURCE_TYPE = GetEnv("UUP_RESOURCE_TYPE", "")
	AMAX_X_API_KEY = GetEnv("AMAX_X_API_KEY", "")
	LoanStatusKeywordValues := GetEnv("LOAN_STATUS_KEYWORD", "LOAN STATUS,UTANG STATUS")
	LOAN_STATUS_KEYWORD = strings.Split(LoanStatusKeywordValues, ",")
	AVAILMENT_TRANSACTION_TTL_IN_HOURS_Str := GetEnv("AVAILMENT_TRANSACTION_TTL_IN_HOURS", "24")
	AVAILMENT_TRANSACTION_TTL_IN_HOURS, _ = strconv.Atoi(AVAILMENT_TRANSACTION_TTL_IN_HOURS_Str)
	LOAN_FOUND_MESSAGE_ID = GetEnv("LOAN_FOUND_MESSAGE_ID", "48503")

	NO_LOAN_FOUND_MESSAGE_ID = GetEnv("NO_LOAN_FOUND_MESSAGE_ID", "48504")

	SERVICE_NAME = GetEnv("SERVICE_NAME", "loanavailment")
	OTEL_URL = GetEnv("OTEL_URL", "")
	LOG_LEVEL = GetEnv("LOG_LEVEL", "INFO")
	REPORT_EVERY_X_HOURS, _ = strconv.Atoi(GetEnv("REPORT_EVERY_X_HOURS", "24"))
	REPORT_CHUNKS, _ = strconv.Atoi(GetEnv("REPORT_CHUNKS", "4"))
	REDIS_ADDR = GetEnv("REDIS_ADDR", "localhost:6379")
	REDIS_PASSWORD = GetEnv("REDIS_PASSWORD", "")
	REDIS_DB_Str := GetEnv("REDIS_DB", "0")
	REDIS_DB, _ = strconv.Atoi(REDIS_DB_Str)
	REDIS_ENABLE_TLS, _ = strconv.ParseBool(GetEnv("REDIS_ENABLE_TLS", "false"))
	REDIS_CONNECT_TIMEOUT_SECONDS_Str := GetEnv("REDIS_CONNECT_TIMEOUT_SECONDS", "5")
	REDIS_CONNECT_TIMEOUT_SECONDS, _ = strconv.Atoi(REDIS_CONNECT_TIMEOUT_SECONDS_Str)
	REDIS_CERT_CONTENT = GetEnv("REDIS_CERT_CONTENT", "")

	// Pub/Sub Configuration
	PROJECT_ID = GetEnv("PROJECT_ID", "")
	PUBSUB_TOPIC = GetEnv("PUBSUB_TOPIC", "glo-isg-dodrio-d-notification-topic-ddr-762")
	}

// GetRedisConfig returns a RedisConfig struct populated from environment variables
func GetRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:           REDIS_ADDR,
		Password:       REDIS_PASSWORD,
		DB:             REDIS_DB,
		EnableTLS:      REDIS_ENABLE_TLS,
		ConnectTimeout: time.Duration(REDIS_CONNECT_TIMEOUT_SECONDS) * time.Second,
		CertContent:    REDIS_CERT_CONTENT,
	}
}

// GetPubSubConfig returns a PubSubConfig struct populated from environment variables
func GetPubSubConfig() PubSubConfig {
	return PubSubConfig{
		ProjectID: PROJECT_ID,
		Topic:     PUBSUB_TOPIC,
		}
}

// GetEnv fetches the value of an environment variable or returns a default value
func GetEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
