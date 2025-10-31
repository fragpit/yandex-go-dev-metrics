package config

import (
	"flag"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"empty", "", false},
		{"no scheme", "example.com", false},
		{"scheme only", "http://", false},
		{"scheme_without_slashes", "http:example.com", false},
		{"invalid_scheme", "ftp://example.com", false},
		{"malformed", "://example.com", false},
		{"http_simple", "http://example.com", true},
		{"https_with_path", "https://example.com/path?x=1", true},
		{"http_with_port", "http://localhost:8080", true},
		{"ipv4", "http://127.0.0.1", true},
		{"ipv6", "http://[::1]", true},
		{"userinfo", "http://user:pass@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateURL(tt.raw); got != tt.want {
				t.Fatalf("validateURL(%q) = %v; want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestAgentConfig_Debug(t *testing.T) {
	cfg := &AgentConfig{
		LogLevel:       "debug",
		ServerURL:      "http://localhost:8080",
		PollInterval:   2,
		ReportInterval: 10,
		SecretKey:      []byte("test"),
		RateLimit:      1,
	}

	cfg.Debug()
}

func TestServerConfig_Debug(t *testing.T) {
	cfg := &ServerConfig{
		LogLevel:      "debug",
		Address:       "localhost:8080",
		FileStorePath: "/tmp/metrics.json",
		Restore:       true,
		DatabaseDSN:   "",
		SecretKey:     []byte("test"),
	}

	cfg.Debug()
}

func TestNewAgentConfig_Defaults(t *testing.T) {
	// Сбрасываем флаги и переменные окружения
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	cfg, err := NewAgentConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "http://localhost:8080", cfg.ServerURL)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 1, cfg.RateLimit)
}

func TestNewAgentConfig_WithFlags(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"-log-level", "debug",
		"-a", "localhost:9090",
		"-p", "5",
		"-r", "20",
		"-k", "secret123",
		"-l", "3",
	}

	cfg, err := NewAgentConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "http://localhost:9090", cfg.ServerURL)
	assert.Equal(t, 5, cfg.PollInterval)
	assert.Equal(t, 20, cfg.ReportInterval)
	assert.Equal(t, []byte("secret123"), cfg.SecretKey)
	assert.Equal(t, 3, cfg.RateLimit)
}

func TestNewAgentConfig_WithEnvVars(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	// Устанавливаем переменные окружения
	_ = os.Setenv("LOG_LEVEL", "warn")
	_ = os.Setenv("ADDRESS", "example.com:8080")
	_ = os.Setenv("POLL_INTERVAL", "7")
	_ = os.Setenv("REPORT_INTERVAL", "15")
	_ = os.Setenv("KEY", "envkey456")
	_ = os.Setenv("RATE_LIMIT", "5")

	defer func() {
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("ADDRESS")
		_ = os.Unsetenv("POLL_INTERVAL")
		_ = os.Unsetenv("REPORT_INTERVAL")
		_ = os.Unsetenv("KEY")
		_ = os.Unsetenv("RATE_LIMIT")
	}()

	cfg, err := NewAgentConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, "http://example.com:8080", cfg.ServerURL)
	assert.Equal(t, 7, cfg.PollInterval)
	assert.Equal(t, 15, cfg.ReportInterval)
	assert.Equal(t, []byte("envkey456"), cfg.SecretKey)
	assert.Equal(t, 5, cfg.RateLimit)
}

func TestNewAgentConfig_EnvOverridesFlags(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"-a", "localhost:9090",
		"-p", "5",
	}

	_ = os.Setenv("ADDRESS", "override.com:7070")
	_ = os.Setenv("POLL_INTERVAL", "12")
	defer func() {
		_ = os.Unsetenv("ADDRESS")
		_ = os.Unsetenv("POLL_INTERVAL")
	}()

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://override.com:7070", cfg.ServerURL)
	assert.Equal(t, 12, cfg.PollInterval)
}

func TestNewAgentConfig_InvalidPollInterval(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("POLL_INTERVAL", "invalid")
	defer func() { _ = os.Unsetenv("POLL_INTERVAL") }()

	cfg, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "POLL_INTERVAL")
}

func TestNewAgentConfig_InvalidReportInterval(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("REPORT_INTERVAL", "invalid")
	defer func() { _ = os.Unsetenv("REPORT_INTERVAL") }()

	cfg, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "REPORT_INTERVAL")
}

func TestNewAgentConfig_InvalidRateLimit(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("RATE_LIMIT", "not_a_number")
	defer func() { _ = os.Unsetenv("RATE_LIMIT") }()

	cfg, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "RATE_LIMIT")
}

func TestNewServerConfig_Defaults(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	cfg, err := NewServerConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 300*time.Second, cfg.StoreInterval)
	assert.Equal(t, "/tmp/metrics.json", cfg.FileStorePath)
	assert.False(t, cfg.Restore)
	assert.Empty(t, cfg.DatabaseDSN)
}

func TestNewServerConfig_WithFlags(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"-log-level", "debug",
		"-a", "0.0.0.0:9090",
		"-i", "60",
		"-f", "/var/metrics.json",
		"-r",
		"-d", "postgres://user:pass@localhost/db",
		"-k", "serverkey",
		"-audit-file", "/tmp/audit.log",
		"-audit-url", "http://audit.example.com",
	}

	cfg, err := NewServerConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "0.0.0.0:9090", cfg.Address)
	assert.Equal(t, 60*time.Second, cfg.StoreInterval)
	assert.Equal(t, "/var/metrics.json", cfg.FileStorePath)
	assert.True(t, cfg.Restore)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, []byte("serverkey"), cfg.SecretKey)
	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.example.com", cfg.AuditURL)
}

func TestNewServerConfig_WithEnvVars(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("LOG_LEVEL", "error")
	_ = os.Setenv("ADDRESS", "0.0.0.0:3000")
	_ = os.Setenv("STORE_INTERVAL", "120")
	_ = os.Setenv("FILE_STORAGE_PATH", "/data/metrics.json")
	_ = os.Setenv("RESTORE", "true")
	_ = os.Setenv("DATABASE_DSN", "postgres://localhost/mydb")
	_ = os.Setenv("KEY", "envserverkey")
	_ = os.Setenv("AUDIT_FILE", "/var/log/audit.log")
	_ = os.Setenv("AUDIT_URL", "https://audit.example.com/api")

	defer func() {
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("ADDRESS")
		_ = os.Unsetenv("STORE_INTERVAL")
		_ = os.Unsetenv("FILE_STORAGE_PATH")
		_ = os.Unsetenv("RESTORE")
		_ = os.Unsetenv("DATABASE_DSN")
		_ = os.Unsetenv("KEY")
		_ = os.Unsetenv("AUDIT_FILE")
		_ = os.Unsetenv("AUDIT_URL")
	}()

	cfg, err := NewServerConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "error", cfg.LogLevel)
	assert.Equal(t, "0.0.0.0:3000", cfg.Address)
	assert.Equal(t, 120*time.Second, cfg.StoreInterval)
	assert.Equal(t, "/data/metrics.json", cfg.FileStorePath)
	assert.True(t, cfg.Restore)
	assert.Equal(t, "postgres://localhost/mydb", cfg.DatabaseDSN)
	assert.Equal(t, []byte("envserverkey"), cfg.SecretKey)
	assert.Equal(t, "/var/log/audit.log", cfg.AuditFile)
	assert.Equal(t, "https://audit.example.com/api", cfg.AuditURL)
}

func TestNewServerConfig_InvalidStoreInterval(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("STORE_INTERVAL", "not_a_number")
	defer func() { _ = os.Unsetenv("STORE_INTERVAL") }()

	cfg, err := NewServerConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "STORE_INTERVAL")
}

func TestNewServerConfig_InvalidRestore(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("RESTORE", "maybe")
	defer func() { _ = os.Unsetenv("RESTORE") }()

	cfg, err := NewServerConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "RESTORE")
}

func TestNewServerConfig_InvalidAuditURL(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	_ = os.Setenv("AUDIT_URL", "ftp://invalid.com")
	defer func() { _ = os.Unsetenv("AUDIT_URL") }()

	cfg, err := NewServerConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid audit URL")
}

func TestNewServerConfig_EmptyAuditURL(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	cfg, err := NewServerConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.AuditURL)
}

func TestGetEnvOrDefault(t *testing.T) {
	l := slog.New(slog.DiscardHandler)
	slog.SetDefault(l)

	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue any
		parser       any
		wantValue    any
		wantErr      bool
	}{
		{
			name:         "string_default_no_env",
			envKey:       "TEST_STRING_EMPTY",
			envValue:     "",
			defaultValue: "default",
			parser:       parseString,
			wantValue:    "default",
			wantErr:      false,
		},
		{
			name:         "string_from_env",
			envKey:       "TEST_STRING",
			envValue:     "value",
			defaultValue: "default",
			parser:       parseString,
			wantValue:    "value",
			wantErr:      false,
		},
		{
			name:         "int_default_no_env",
			envKey:       "TEST_INT_EMPTY",
			envValue:     "",
			defaultValue: 42,
			parser:       parseInt,
			wantValue:    42,
			wantErr:      false,
		},
		{
			name:         "int_from_env",
			envKey:       "TEST_INT_VALID",
			envValue:     "100",
			defaultValue: 42,
			parser:       parseInt,
			wantValue:    100,
			wantErr:      false,
		},
		{
			name:         "int_invalid_env",
			envKey:       "TEST_INT_INVALID",
			envValue:     "not_a_number",
			defaultValue: 42,
			parser:       parseInt,
			wantValue:    42,
			wantErr:      true,
		},
		{
			name:         "bool_default_no_env",
			envKey:       "TEST_BOOL_EMPTY",
			envValue:     "",
			defaultValue: false,
			parser:       parseBool,
			wantValue:    false,
			wantErr:      false,
		},
		{
			name:         "bool_true_from_env",
			envKey:       "TEST_BOOL_TRUE",
			envValue:     "true",
			defaultValue: false,
			parser:       parseBool,
			wantValue:    true,
			wantErr:      false,
		},
		{
			name:         "bool_invalid_env",
			envKey:       "TEST_BOOL_INVALID",
			envValue:     "maybe",
			defaultValue: false,
			parser:       parseBool,
			wantValue:    false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envKey, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envKey)
			}

			var got any
			var err error

			switch def := tt.defaultValue.(type) {
			case string:
				got, err = getEnvOrDefault(tt.envKey, def, parseString)
			case int:
				got, err = getEnvOrDefault(tt.envKey, def, parseInt)
			case bool:
				got, err = getEnvOrDefault(tt.envKey, def, parseBool)
			default:
				t.Fatalf("Unsupported type: %T", tt.defaultValue)
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantValue, got)
		})
	}
}
