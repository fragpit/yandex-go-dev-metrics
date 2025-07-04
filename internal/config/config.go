package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

type AgentConfig struct {
	ServerURL      string `yaml:"address"`
	PollInterval   int    `yaml:"poll"`
	ReportInterval int    `yaml:"report"`
	Restore        bool   `yaml:"restore"`
}

func NewAgentConfig() *AgentConfig {
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

	finalServerURL := *serverURL
	if envServerURL := os.Getenv("ADDRESS"); envServerURL != "" {
		finalServerURL = envServerURL
	}

	finalPollInterval := *pollInterval
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		var err error
		finalPollInterval, err = strconv.Atoi(envPollInterval)
		if err != nil {
			log.Fatalf("invalid POLL_INTERVAL value: %v", err)
		}
	}

	finalReportInterval := *reportInterval
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		var err error
		finalPollInterval, err = strconv.Atoi(envReportInterval)
		if err != nil {
			log.Fatalf("invalid REPORT_INTERVAL value: %v", err)
		}
	}

	finalRestore := *restore
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		var err error
		finalRestore, err = strconv.ParseBool(envRestore)
		if err != nil {
			log.Fatalf("invalid REPORT_INTERVAL value: %v", err)
		}
	}

	if !strings.HasPrefix(finalServerURL, "http://") {
		finalServerURL = "http://" + finalServerURL
	}

	return &AgentConfig{
		ServerURL:      finalServerURL,
		PollInterval:   finalPollInterval,
		ReportInterval: finalReportInterval,
		Restore:        finalRestore,
	}
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

func NewServerConfig() *ServerConfig {
	address := flag.String(
		"a",
		"localhost:8080",
		"address to listen on (по умолчанию localhost:8080)",
	)

	flag.Parse()

	finalAddress := *address
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		finalAddress = envAddr
	}

	return &ServerConfig{
		Address: finalAddress,
	}
}
