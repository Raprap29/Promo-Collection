package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisClientConstructor func(opt *redis.Options) *redis.Client

type RedisClient struct {
	Client *redis.Client
}

func ConnectToRedis(
	ctx context.Context,
	cfg configs.RedisConfig,

	newClientFunc RedisClientConstructor,
) (*RedisClient, error) {

	logger.Info(ctx, "Connecting to Redis %s %d %t", slog.String("addr", cfg.Addr), slog.Int("db", cfg.DB), slog.Bool("enable_tls", cfg.EnableTLS))

	options := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.EnableTLS {
		tlsConfig, err := buildTLSConfig(ctx, cfg)
		if err != nil {
			logger.Error(ctx, "Failed to build TLS config %s", err)
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		options.TLSConfig = tlsConfig
	}

	if newClientFunc == nil {
		newClientFunc = redis.NewClient
	}
	client := newClientFunc(options)

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error(ctx, "Redis ping failed", err)
		return nil, err
	}

	logger.Info(ctx, "Successfully connected to Redis")

	logger.Debug(ctx, "Redis client initialized %s %d",
		slog.String("addr", cfg.Addr),
		slog.Int("db", cfg.DB),
	)

	return &RedisClient{
		Client: client,
	}, nil
}

func buildTLSConfig(ctx context.Context, cfg configs.RedisConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.CertContent == "" {
		return tlsConfig, nil
	}

	certContentBytes := []byte(cfg.CertContent)
	var loadedAny bool

	// Attempt to load as a client certificate and key pair first.
	if cert, err := tls.X509KeyPair(certContentBytes, certContentBytes); err == nil {
		tlsConfig.Certificates = []tls.Certificate{cert}
		logger.Info(ctx, "Loaded client certificate from PEM content")
		loadedAny = true
	}

	// Separately, attempt to load any root CA certificates from the same content.
	caCertPool := x509.NewCertPool()
	if caCertPool.AppendCertsFromPEM(certContentBytes) {
		tlsConfig.RootCAs = caCertPool
		logger.Info(ctx, "Loaded CA certificate(s) from PEM content")
		loadedAny = true
	}

	// If neither parsing attempt worked, the content is invalid.
	if !loadedAny {
		return nil, fmt.Errorf("failed to parse PEM content as a valid CA certificate or client key pair")
	}

	return tlsConfig, nil
}

func Disconnect(client *redis.Client) error {
	return client.Close()
}
