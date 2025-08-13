package config

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type AgentConfig struct {
	LogLevel       string
	ServerURL      string
	PollInterval   int
	ReportInterval int
	SecretKey      []byte
	RateLimit      int
}

func NewAgentConfig() *AgentConfig {
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

	flag.Parse()

	finalLogLevel := *logLevel
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		finalLogLevel = env
	}

	finalServerURL := *serverURL
	if env := os.Getenv("ADDRESS"); env != "" {
		finalServerURL = env
	}
	if !strings.HasPrefix(finalServerURL, "http://") {
		finalServerURL = "http://" + finalServerURL
	}

	finalPollInterval := *pollInterval
	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		var err error
		finalPollInterval, err = strconv.Atoi(env)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "POLL_INTERVAL"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	finalReportInterval := *reportInterval
	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		var err error
		finalReportInterval, err = strconv.Atoi(env)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "REPORT_INTERVAL"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	finalSecretKey := *secretKey
	if env := os.Getenv("KEY"); env != "" {
		finalSecretKey = env
	}

	finalRateLimit := *rateLimit
	if env := os.Getenv("RATE_LIMIT"); env != "" {
		var err error
		finalRateLimit, err = strconv.Atoi(env)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "RATE_LIMIT"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	return &AgentConfig{
		LogLevel:       finalLogLevel,
		ServerURL:      finalServerURL,
		PollInterval:   finalPollInterval,
		ReportInterval: finalReportInterval,
		SecretKey:      []byte(finalSecretKey),
		RateLimit:      finalRateLimit,
	}
}

type ServerConfig struct {
	LogLevel      string
	Address       string
	StoreInterval time.Duration
	FileStorePath string
	Restore       bool
	DatabaseDSN   string
	SecretKey     []byte
}

func NewServerConfig() *ServerConfig {
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

	flag.Parse()

	finalLogLevel := *logLevel
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		finalLogLevel = env
	}

	finalAddress := *address
	if env := os.Getenv("ADDRESS"); env != "" {
		finalAddress = env
	}

	finalStoreInterval := *storeInterval
	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		var err error
		finalStoreInterval, err = strconv.Atoi(env)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "STORE_INTERVAL"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}
	storeIntervalDuration := time.Duration(finalStoreInterval) * time.Second

	finalFileStorePath := *fileStorePath
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		finalFileStorePath = env
	}

	finalRestore := *restore
	if env := os.Getenv("RESTORE"); env != "" {
		var err error
		finalRestore, err = strconv.ParseBool(env)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "RESTORE"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	finalDBDSN := *dbDSN
	if env := os.Getenv("DATABASE_DSN"); env != "" {
		finalDBDSN = env
	}

	finalSecretKey := *secretKey
	if env := os.Getenv("KEY"); env != "" {
		finalSecretKey = env
	}

	return &ServerConfig{
		LogLevel:      finalLogLevel,
		Address:       finalAddress,
		StoreInterval: storeIntervalDuration,
		FileStorePath: finalFileStorePath,
		Restore:       finalRestore,
		DatabaseDSN:   finalDBDSN,
		SecretKey:     []byte(finalSecretKey),
	}
}

func (c *AgentConfig) Debug() {
	slog.Info(
		"agent config",
		slog.String("log_level", c.LogLevel),
		slog.String("server_url", c.ServerURL),
		slog.Int("poll_interval", c.PollInterval),
		slog.Int("report_interval", c.ReportInterval),
	)
}

func (c *ServerConfig) Debug() {
	slog.Info(
		"server config",
		slog.String("log_level", c.LogLevel),
		slog.String("address", c.Address),
		slog.Duration("store_interval", c.StoreInterval),
		slog.String("file_store_path", c.FileStorePath),
		slog.Bool("restore", c.Restore),
		slog.String("database_dsn", c.DatabaseDSN),
	)
}
