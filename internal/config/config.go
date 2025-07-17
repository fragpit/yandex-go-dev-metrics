package config

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type AgentConfig struct {
	LogLevel       string `yaml:"log_level"`
	ServerURL      string `yaml:"address"`
	PollInterval   int    `yaml:"poll"`
	ReportInterval int    `yaml:"report"`
	Restore        bool   `yaml:"restore"`
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
	restore := flag.Bool(
		"rs",
		false,
		"",
	)

	flag.Parse()

	finalLogLevel := *logLevel
	if env := os.Getenv("DEBUG"); env != "" {
		finalLogLevel = env
	}

	finalServerURL := *serverURL
	if envServerURL := os.Getenv("ADDRESS"); envServerURL != "" {
		finalServerURL = envServerURL
	}

	finalPollInterval := *pollInterval
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		var err error
		finalPollInterval, err = strconv.Atoi(envPollInterval)
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
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		var err error
		finalReportInterval, err = strconv.Atoi(envReportInterval)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "REPORT_INTERVAL"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	finalRestore := *restore
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		var err error
		finalRestore, err = strconv.ParseBool(envRestore)
		if err != nil {
			slog.Error(
				"error converting parameter",
				slog.String("parameter", "RESTORE"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	if !strings.HasPrefix(finalServerURL, "http://") {
		finalServerURL = "http://" + finalServerURL
	}

	return &AgentConfig{
		LogLevel:       finalLogLevel,
		ServerURL:      finalServerURL,
		PollInterval:   finalPollInterval,
		ReportInterval: finalReportInterval,
		Restore:        finalRestore,
	}
}

type ServerConfig struct {
	LogLevel string `yaml:"log_level"`
	Address  string `yaml:"address"`
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

	flag.Parse()

	finalLogLevel := *logLevel
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		finalLogLevel = env
	}

	finalAddress := *address
	if env := os.Getenv("ADDRESS"); env != "" {
		finalAddress = env
	}

	return &ServerConfig{
		LogLevel: finalLogLevel,
		Address:  finalAddress,
	}
}
