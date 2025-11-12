package config

import (
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
	LogLevel       string
	ServerURL      string
	PollInterval   int
	ReportInterval int
	SecretKey      []byte
	RateLimit      int
	CryptoKey      string
}

func NewAgentConfig() (*AgentConfig, error) {
	logLevel := flag.String(
		"log-level",
		"info",
		"log level (default: info)",
	)

	serverURL := flag.String(
		"a",
		"localhost:8080",
		"address to connect to server (по умолчанию http://localhost:8080)",
	)

	pollInterval := flag.Int(
		"p",
		2,
		"частота опроса метрик из пакета runtime (по умолчанию 2 секунды)",
	)

	reportInterval := flag.Int(
		"r",
		10,
		"частота отправки метрик на сервер (по умолчанию 10 секунд)",
	)

	secretKey := flag.String(
		"k",
		"",
		"секретный ключ для подписи сообщений",
	)

	rateLimit := flag.Int(
		"l",
		1,
		"лимит одновременных запросов к серверу (по умолчанию 1)",
	)

	cryptoKey := flag.String(
		"crypto-key",
		"",
		"публичный ключ для шифрования сообщений (выключено по умолчанию)",
	)

	flag.Parse()

	finalLogLevel, err := getEnvOrDefault("LOG_LEVEL", *logLevel, parseString)
	if err != nil {
		return nil, err
	}

	finalServerURL, err := getEnvOrDefault("ADDRESS", *serverURL, parseString)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(finalServerURL, "http://") {
		finalServerURL = "http://" + finalServerURL
	}

	finalPollInterval, err := getEnvOrDefault(
		"POLL_INTERVAL",
		*pollInterval,
		parseInt,
	)
	if err != nil {
		return nil, err
	}

	finalReportInterval, err := getEnvOrDefault(
		"REPORT_INTERVAL",
		*reportInterval,
		parseInt,
	)
	if err != nil {
		return nil, err
	}

	finalSecretKey, err := getEnvOrDefault("KEY", *secretKey, parseString)
	if err != nil {
		return nil, err
	}

	finalRateLimit, err := getEnvOrDefault("RATE_LIMIT", *rateLimit, parseInt)
	if err != nil {
		return nil, err
	}

	finalCryptoKey, err := getEnvOrDefault("CRYPTO_KEY", *cryptoKey, parseString)
	if err != nil {
		return nil, err
	}

	return &AgentConfig{
		LogLevel:       finalLogLevel,
		ServerURL:      finalServerURL,
		PollInterval:   finalPollInterval,
		ReportInterval: finalReportInterval,
		SecretKey:      []byte(finalSecretKey),
		RateLimit:      finalRateLimit,
		CryptoKey:      finalCryptoKey,
	}, nil
}

// Debug logs the current agent configuration.
func (c *AgentConfig) Debug() {
	slog.Info(
		"agent config",
		slog.String("log_level", c.LogLevel),
		slog.String("server_url", c.ServerURL),
		slog.Int("poll_interval", c.PollInterval),
		slog.Int("report_interval", c.ReportInterval),
	)
}

type ServerConfig struct {
	LogLevel      string
	Address       string
	StoreInterval time.Duration
	FileStorePath string
	Restore       bool
	DatabaseDSN   string
	SecretKey     []byte
	AuditFile     string
	AuditURL      string
	CryptoKey     string
}

func NewServerConfig() (*ServerConfig, error) {
	logLevel := flag.String(
		"log-level",
		"info",
		"log level (default: info)",
	)

	address := flag.String(
		"a",
		"localhost:8080",
		"address to listen on (по умолчанию localhost:8080)",
	)

	storeInterval := flag.Int(
		"i",
		300,
		"частота сохранения метрик в файл в секундах (по умолчанию 300 секунд)",
	)

	fileStorePath := flag.String(
		"f",
		"/tmp/metrics.json",
		"путь к файлу для сохранения метрик (по умолчанию /tmp/metrics.json)",
	)

	restore := flag.Bool(
		"r",
		false,
		"восстанавливать метрики из файла при запуске сервера (по умолчанию false)",
	)

	dbDSN := flag.String(
		"d",
		"",
		"строка подключения к БД, если не указана используется memory storage (по умолчанию пусто)",
	)

	secretKey := flag.String(
		"k",
		"",
		"секретный ключ для подписи сообщений",
	)

	auditFile := flag.String(
		"audit-file",
		"",
		"файл для записи аудита (по умолчанию не используется)",
	)

	auditURL := flag.String(
		"audit-url",
		"",
		"URL для отправки аудита (по умолчанию не используется)",
	)

	cryptoKey := flag.String(
		"crypto-key",
		"",
		"приватный ключ для шифрования сообщений (выключено по умолчанию)",
	)

	flag.Parse()

	finalLogLevel, err := getEnvOrDefault("LOG_LEVEL", *logLevel, parseString)
	if err != nil {
		return nil, err
	}

	finalAddress, err := getEnvOrDefault("ADDRESS", *address, parseString)
	if err != nil {
		return nil, err
	}

	finalStoreInterval, err := getEnvOrDefault(
		"STORE_INTERVAL",
		*storeInterval,
		parseInt,
	)
	if err != nil {
		return nil, err
	}
	storeIntervalDuration := time.Duration(finalStoreInterval) * time.Second

	finalFileStorePath, err := getEnvOrDefault(
		"FILE_STORAGE_PATH",
		*fileStorePath,
		parseString,
	)
	if err != nil {
		return nil, err
	}

	finalRestore, err := getEnvOrDefault("RESTORE", *restore, parseBool)
	if err != nil {
		return nil, err
	}

	finalDBDSN, err := getEnvOrDefault("DATABASE_DSN", *dbDSN, parseString)
	if err != nil {
		return nil, err
	}

	finalSecretKey, err := getEnvOrDefault("KEY", *secretKey, parseString)
	if err != nil {
		return nil, err
	}

	finalAuditFile, err := getEnvOrDefault("AUDIT_FILE", *auditFile, parseString)
	if err != nil {
		return nil, err
	}

	finalAuditURL, err := getEnvOrDefault("AUDIT_URL", *auditURL, parseString)
	if err != nil {
		return nil, err
	}

	if finalAuditURL != "" && !validateURL(finalAuditURL) {
		slog.Error("invalid audit URL", slog.String("url", finalAuditURL))
		return nil, fmt.Errorf("invalid audit URL: %s", finalAuditURL)
	}

	finalCryptoKey, err := getEnvOrDefault("CRYPTO_KEY", *cryptoKey, parseString)
	if err != nil {
		return nil, err
	}

	return &ServerConfig{
		LogLevel:      finalLogLevel,
		Address:       finalAddress,
		StoreInterval: storeIntervalDuration,
		FileStorePath: finalFileStorePath,
		Restore:       finalRestore,
		DatabaseDSN:   finalDBDSN,
		SecretKey:     []byte(finalSecretKey),
		AuditFile:     finalAuditFile,
		AuditURL:      finalAuditURL,
		CryptoKey:     finalCryptoKey,
	}, nil
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
