package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"golang.org/x/sync/errgroup"
)

var counter atomic.Int64

func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stop()

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
	slog.SetDefault(logger)
	logger.Info("starting agent")

	repo := memstorage.NewMemoryStorage()

	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	reportDuration := time.Duration(cfg.ReportInterval) * time.Second

	var pollers []Poller
	runtimePoller := NewRuntimePoller(logger.With("service", "runtime poller"))
	customPoller := NewCustomPoller(logger.With("service", "custom poller"))
	psPoller := NewPSPoller(logger.With("service", "ps poller"))
	pollers = append(pollers, runtimePoller, customPoller, psPoller)

	pollCh := make(chan model.Metric, 1024)

	eg, ctx := errgroup.WithContext(ctx)

	for i := range pollers {
		p := pollers[i]
		eg.Go(func() error {
			if err := Poll(ctx, pollCh, pollDuration, p); err != nil {
				logger.Error("poller error", slog.Any("error", err))
				return err
			}
			return nil
		})
	}

	aggregator := NewAggregator(logger.With("service", "aggregator"), repo)
	eg.Go(func() error {
		if err := aggregator.RunAggregator(ctx, pollCh); err != nil {
			logger.Error("aggregator error", slog.Any("error", err))
			return err
		}
		return nil
	})

	reporter, err := NewReporter(
		logger.With("service", "reporter"),
		repo,
		cfg.ServerURL,
		[]byte(cfg.SecretKey),
		cfg.RateLimit,
		cfg.CryptoKey,
		cfg.GRPCServerAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to init reporter: %w", err)
	}
	defer reporter.transport.Close()

	eg.Go(func() error {
		if err := reporter.RunReporter(ctx, reportDuration); err != nil {
			logger.Error("reporter error", slog.Any("error", err))
			return err
		}
		return nil
	})

	err = eg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Info("agent shutdown with error", slog.Any("error", err))
		return err
	}

	logger.Info("agent shutdown")
	return nil
}
