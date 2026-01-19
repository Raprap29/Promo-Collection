package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var baseValidConfig = AppConfig{
	Server:  ServerConfig{Port: 8080},
	Logging: LogConfig{LogLevel: "info"},
	PubSub: PubSubConfig{
		ProjectID:    "pid",
		Subscription: "subscription",
	},
}

// Helper to write a config file to temp path
func writeTempConfig(t *testing.T, cfg AppConfig) string {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmp, data, 0644))
	return tmp
}

func TestLoadFromConfigFilePath_FileNotFound(t *testing.T) {
	_, err := LoadFromConfigFilePath("/does/not/exist.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadFromConfigFilePath_InvalidYAML(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(tmp, []byte("invalid: [unclosed"), 0644))

	_, err := LoadFromConfigFilePath(tmp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestAssignDefaultConfigValues(t *testing.T) {
	cfg := AppConfig{}

	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOGGING_LEVEL", "warn")
	t.Setenv("PROJECT_ID", "my-project")
	t.Setenv("PUBSUB_SUBSCRIPTION", "test-subscription")

	assignDefaultConfigValues(&cfg)

	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "warn", cfg.Logging.LogLevel)
	assert.Equal(t, "my-project", cfg.PubSub.ProjectID)
	assert.Equal(t, "test-subscription", cfg.PubSub.Subscription)
}
func TestGetEnvOrDefaultAsInt(t *testing.T) {
	t.Setenv("INT_KEY", "42")
	assert.Equal(t, 42, GetEnvOrDefaultAsInt("INT_KEY", 5))

	t.Setenv("INT_KEY", "invalid")
	assert.Equal(t, 5, GetEnvOrDefaultAsInt("INT_KEY", 5))

	os.Unsetenv("INT_KEY")
	assert.Equal(t, 5, GetEnvOrDefaultAsInt("INT_KEY", 5))
}

func TestGetEnvOrDefaultAsString(t *testing.T) {
	t.Setenv("STRING_KEY", "value")
	assert.Equal(t, "value", GetEnvOrDefaultAsString("STRING_KEY", "default"))

	os.Unsetenv("STRING_KEY")
	assert.Equal(t, "default", GetEnvOrDefaultAsString("STRING_KEY", "default"))
}
func TestLoadFromConfig(t *testing.T) {
	path := writeTempConfig(t, baseValidConfig)
	t.Setenv("CONFIG_PATH", path)

	cfg, err := LoadFromConfig()
	require.NoError(t, err)
	assert.Equal(t, baseValidConfig.Server.Port, cfg.Server.Port)
}

func TestLoadFromConfig_BadPath(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/bad/path.yaml")
	_, err := LoadFromConfig()
	assert.Error(t, err)
}
