// redis_test.go

package redis

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	"promocollection/internal/pkg/config"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Corp"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certOut := &bytes.Buffer{}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, nil, err
	}

	keyOut := &bytes.Buffer{}
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		return nil, nil, err
	}

	return certOut.Bytes(), keyOut.Bytes(), nil
}

func TestBuildTLSConfig(t *testing.T) {
	ctx := context.Background()
	t.Run("should return default config when no cert content is provided", func(t *testing.T) {
		cfg := config.RedisConfig{}
		tlsConfig, err := buildTLSConfig(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.Nil(t, tlsConfig.RootCAs)
		assert.Nil(t, tlsConfig.Certificates)
	})

	t.Run("should load CA certificate from content", func(t *testing.T) {
		caCert, _, err := generateSelfSignedCert()
		require.NoError(t, err)

		cfg := config.RedisConfig{CertContent: string(caCert)}
		tlsConfig, err := buildTLSConfig(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig.RootCAs)
		assert.Nil(t, tlsConfig.Certificates) // A CA cert is not a client cert
	})

	t.Run("should load client certificate from content", func(t *testing.T) {
		cert, key, err := generateSelfSignedCert()
		require.NoError(t, err)
		// Client cert content is typically the cert followed by the key
		clientCertContent := string(cert) + "\n" + string(key)

		cfg := config.RedisConfig{CertContent: clientCertContent}
		tlsConfig, err := buildTLSConfig(ctx, cfg)
		require.NoError(t, err)
		// The new logic correctly populates the client certificate
		assert.NotNil(t, tlsConfig.Certificates)
		assert.Len(t, tlsConfig.Certificates, 1)
	})

	t.Run("should return error for invalid certificate content", func(t *testing.T) {
		cfg := config.RedisConfig{CertContent: "this is not a valid cert"}
		_, err := buildTLSConfig(ctx, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse PEM content")
	})
}

func TestConnectToRedis(t *testing.T) {
	ctx := context.Background()

	t.Run("should connect successfully without TLS", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		mockNewClient := func(opt *redis.Options) *redis.Client {
			assert.Nil(t, opt.TLSConfig)
			return db
		}

		mock.ExpectPing().SetVal("PONG")

		cfg := config.RedisConfig{Addr: "localhost:6379"}
		redisClient, err := ConnectToRedis(ctx, cfg, mockNewClient)
		require.NoError(t, err)
		assert.NotNil(t, redisClient)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should fail if ping fails", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		mockNewClient := func(opt *redis.Options) *redis.Client {
			return db
		}

		expectedErr := errors.New("redis is down")
		mock.ExpectPing().SetErr(expectedErr)

		cfg := config.RedisConfig{Addr: "localhost:6379"}
		_, err := ConnectToRedis(ctx, cfg, mockNewClient)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should connect successfully with TLS", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		mockNewClient := func(opt *redis.Options) *redis.Client {
			assert.NotNil(t, opt.TLSConfig, "TLSConfig should be set")
			assert.NotNil(t, opt.TLSConfig.RootCAs, "RootCAs should be set")
			return db
		}

		mock.ExpectPing().SetVal("PONG")

		caCert, _, err := generateSelfSignedCert()
		require.NoError(t, err)
		cfg := config.RedisConfig{EnableTLS: true, CertContent: string(caCert)}

		redisClient, err := ConnectToRedis(ctx, cfg, mockNewClient)
		require.NoError(t, err)
		assert.NotNil(t, redisClient)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should fail if TLS config build fails", func(t *testing.T) {
		cfg := config.RedisConfig{EnableTLS: true, CertContent: "invalid cert"}
		_, err := ConnectToRedis(ctx, cfg, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build TLS config")
	})

	t.Run("should use default client constructor when nil is provided", func(t *testing.T) {
		// Use a non-existent port to guarantee a connection failure
		cfg := config.RedisConfig{Addr: "localhost:9999"}
		_, err := ConnectToRedis(ctx, cfg, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dial tcp")
	})
}

func TestDisconnect(t *testing.T) {
	db, _ := redismock.NewClientMock()
	err := Disconnect(db)
	assert.NoError(t, err)
}