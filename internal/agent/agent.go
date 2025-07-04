package agent

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
)

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	cfg := config.NewAgentConfig()
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	pollTick := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	reportTick := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)

	m := NewMetrics(logger)
	if err := m.pollMetrics(); err != nil {
		return err
	}

	for {
		select {
		case <-pollTick.C:
			logger.Info("polling metrics")
			if err := m.pollMetrics(); err != nil {
				return err
			}
		case <-reportTick.C:
			logger.Info("reporting metrics")
			m.reportMetrics(cfg.ServerURL)
		case <-ctx.Done():
			log.Println("agent shut down")
			return nil
		}
	}
}
