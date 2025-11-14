package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type envParser[T any] func(string) (T, error)

func getEnvOrDefault[T any](
	envKey string,
	defaultValue T,
	parser envParser[T],
) (T, error) {
	env := os.Getenv(envKey)
	if env == "" {
		return defaultValue, nil
	}

	value, err := parser(env)
	if err != nil {
		slog.Error(
			"error converting parameter",
			slog.String("parameter", envKey),
			slog.Any("error", err),
		)
		return defaultValue, fmt.Errorf(
			"failed to set parameter %s: %w",
			envKey,
			err,
		)
	}

	return value, nil
}

func parseString(s string) (string, error) {
	return s, nil
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func parseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

type AgentConfig struct {
	LogLevel       string `mapstructure:"log_level"`
	ServerURL      string `mapstructure:"address"`
	PollInterval   int    `mapstructure:"poll_interval"`
	ReportInterval int    `mapstructure:"report_interval"`
	SecretKey      string `mapstructure:"secret_key"`
	RateLimit      int    `mapstructure:"rate_limit"`
	CryptoKey      string `mapstructure:"crypto_key"`
}

func NewAgentConfig() (*AgentConfig, error) {
	v := viper.New()

	pflag.String("log-level", "info", "log level")

	pflag.StringP(
		"address",
		"a",
		"http://localhost:8080",
		"address to connect to server",
	)

	pflag.IntP(
		"poll-interval",
		"p",
		2,
		"частота опроса метрик из пакета runtime в секундах",
	)

	pflag.IntP(
		"report-interval",
		"r",
		10,
		"частота отправки метрик на сервер в секундах",
	)

	pflag.StringP(
		"secret-key",
		"k",
		"",
		"секретный ключ для подписи сообщений",
	)

	pflag.IntP(
		"rate-limit",
		"l",
		1,
		"лимит одновременных запросов к серверу",
	)

	pflag.String(
		"crypto-key",
		"",
		"путь к публичному ключу для шифрования сообщений",
	)

	cfgPath := pflag.StringP(
		"config",
		"c",
		"",
		"path to config file (по умолчанию не используется)",
	)

	pflag.Parse()

	if *cfgPath != "" {
		v.SetConfigFile(*cfgPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind cmd flags: %w", err)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.RegisterAlias("log_level", "log-level")
	v.RegisterAlias("poll_interval", "poll-interval")
	v.RegisterAlias("report_interval", "report-interval")
	v.RegisterAlias("secret_key", "secret-key")
	v.RegisterAlias("rate_limit", "rate-limit")
	v.RegisterAlias("crypto_key", "crypto-key")

	if !strings.HasPrefix(v.GetString("address"), "http://") {
		v.Set("address", "http://"+v.GetString("address"))
	}

	cfg := &AgentConfig{}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// Debug logs the current agent configuration.
func (c *AgentConfig) Debug() {
	slog.Info(
		"agent config",
		slog.String("log_level", c.LogLevel),
		slog.String("server_url", c.ServerURL),
		slog.Int("poll_interval", c.PollInterval),
		slog.Int("report_interval", c.ReportInterval),
		slog.Int("rate_limit", c.RateLimit),
		slog.String("crypto_key", c.CryptoKey),
	)
}

type ServerConfig struct {
	LogLevel      string        `mapstructure:"log_level"`
	Address       string        `mapstructure:"address"`
	StoreInterval time.Duration `mapstructure:"store_interval"`
	FileStorePath string        `mapstructure:"file_storage_path"`
	Restore       bool          `mapstructure:"restore"`
	DatabaseDSN   string        `mapstructure:"database_dsn"`
	SecretKey     string        `mapstructure:"secret_key"`
	AuditFile     string        `mapstructure:"audit_file"`
	AuditURL      string        `mapstructure:"audit_url"`
	CryptoKey     string        `mapstructure:"crypto_key"`
}

func NewServerConfig() (*ServerConfig, error) {
	v := viper.New()

	pflag.String("log-level", "info", "log level")
	pflag.StringP("address", "a", "localhost:8080", "address to listen on")

	pflag.DurationP(
		"store-interval",
		"i",
		300*time.Second,
		"частота сохранения метрик в файл в секундах",
	)

	pflag.StringP(
		"file-storage-path",
		"f",
		"/tmp/metrics.json",
		"путь к файлу для сохранения метрик",
	)

	pflag.BoolP(
		"restore",
		"r",
		false,
		"восстанавливать метрики из файла при запуске сервера",
	)

	pflag.StringP(
		"database-dsn",
		"d",
		"",
		"строка подключения к БД, если не указана используется memory storage",
	)

	pflag.StringP(
		"secret-key",
		"k",
		"",
		"секретный ключ для подписи сообщений",
	)

	pflag.String(
		"audit-file",
		"",
		"файл для записи аудита (по умолчанию не используется)",
	)

	pflag.String(
		"audit-url",
		"",
		"URL для отправки аудита (по умолчанию не используется)",
	)

	pflag.String(
		"crypto-key",
		"",
		"путь к публичному ключу для шифрования сообщений",
	)

	cfgPath := pflag.StringP(
		"config",
		"c",
		"",
		"path to config file (по умолчанию не используется)",
	)

	pflag.Parse()

	if *cfgPath != "" {
		v.SetConfigFile(*cfgPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind cmd flags: %w", err)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.RegisterAlias("log_level", "log-level")
	v.RegisterAlias("store_interval", "store-interval")
	v.RegisterAlias("file_storage_path", "file-storage-path")
	v.RegisterAlias("database_dsn", "database-dsn")
	v.RegisterAlias("secret_key", "secret-key")
	v.RegisterAlias("audit_file", "audit-file")
	v.RegisterAlias("audit_url", "audit-url")
	v.RegisterAlias("crypto_key", "crypto-key")

	cfg := &ServerConfig{}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.AuditURL != "" && !validateURL(cfg.AuditURL) {
		return nil, fmt.Errorf("invalid audit URL: %s", cfg.AuditURL)
	}

	return cfg, nil
}

// Debug logs the current server configuration.
func (c *ServerConfig) Debug() {
	slog.Info(
		"server config",
		slog.String("log_level", c.LogLevel),
		slog.String("address", c.Address),
		slog.Duration("store_interval", c.StoreInterval),
		slog.String("file_store_path", c.FileStorePath),
		slog.Bool("restore", c.Restore),
		slog.String("database_dsn", c.DatabaseDSN),
		slog.String("audit_file", c.AuditFile),
		slog.String("audit_url", c.AuditURL),
		slog.String("crypto_key", c.CryptoKey),
	)
}

func validateURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	return true
}
