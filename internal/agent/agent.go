package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

var counter atomic.Int64

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	cfg, err := config.NewAgentConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if cfg.LogLevel == "debug" {
		cfg.Debug()
	}

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
	logger.Info("starting agent")

	st := memstorage.NewMemoryStorage()

	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	reportDuration := time.Duration(cfg.ReportInterval) * time.Second

	var wg sync.WaitGroup

	var pollers []Poller
	runtimePoller := NewRuntimePoller(logger.With("service", "runtime poller"))
	customPoller := NewCustomPoller(logger.With("service", "custom poller"))
	psPoller := NewPSPoller(logger.With("service", "ps poller"))
	pollers = append(pollers, runtimePoller, customPoller, psPoller)

	pollCh := make(chan model.Metric, 1024)

	wg.Add(len(pollers))
	for _, p := range pollers {
		go func() {
			defer wg.Done()
			if err := Poll(ctx, pollCh, pollDuration, p); err != nil {
				logger.Error("poller error", slog.Any("error", err))
				cancel()
			}
		}()
	}

	aggregator := NewAggregator(logger.With("service", "aggregator"), st)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := aggregator.RunAggregator(ctx, pollCh); err != nil {
			logger.Error("aggregator error", slog.Any("error", err))
			cancel()
		}
	}()

	reporter := NewReporter(
		logger.With("service", "reporter"),
		st,
		cfg.ServerURL,
		[]byte(cfg.SecretKey),
		cfg.RateLimit,
		cfg.CryptoKey,
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := reporter.RunReporter(ctx, reportDuration); err != nil {
			logger.Error("reporter error", slog.Any("error", err))
			cancel()
		}
	}()

	wg.Wait()

	logger.Info("agent shutdown")
	return nil
}
