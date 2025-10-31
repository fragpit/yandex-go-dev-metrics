package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fragpit/yandex-go-dev-metrics/internal/audit"
	"github.com/fragpit/yandex-go-dev-metrics/internal/cacher"
	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/postgresql"
)

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	cfg, err := config.NewServerConfig()
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

	var st repository.Repository
	if cfg.DatabaseDSN != "" {
		if st, err = postgresql.NewStorage(ctx, cfg.DatabaseDSN); err != nil {
			return err
		}
	} else {
		st = memstorage.NewMemoryStorage()
	}

	auditor := audit.NewAuditor()

	if cfg.AuditFile != "" {
		fileAuditor := audit.NewFileObserver(cfg.AuditFile)
		auditor.Add(fileAuditor)
	}

	if cfg.AuditURL != "" {
		httpAuditor := audit.NewHTTPObserver(cfg.AuditURL)
		auditor.Add(httpAuditor)
	}

	logger.Info("starting server", slog.String("address", cfg.Address))

	if cfg.DatabaseDSN == "" {
		cr := cacher.NewCacher(
			logger,
			st,
			cfg.FileStorePath,
			cfg.StoreInterval,
		)

		if cfg.Restore {
			logger.Info(
				"restoring metrics from file",
				slog.String("file", cfg.FileStorePath),
			)
			var err error
			var metricsList []model.Metric
			if metricsList, err = cr.Restore(); err != nil {
				logger.Error(
					"failed to restore metrics",
					slog.String("error", err.Error()),
				)

				if os.IsNotExist(err) {
					logger.Info("no metrics file found, starting with empty storage")
				} else {
					return err
				}
			}
			if err = st.Initialize(metricsList); err != nil {
				logger.Error(
					"failed to restore metrics",
					slog.String("error", err.Error()),
				)
				return err
			}

			logger.Info(
				"metrics restored from file",
				slog.Int("total", len(metricsList)),
			)
		}

		go func() {
			if err := cr.Run(ctx); err != nil {
				logger.Error("cacher error", slog.String("error", err.Error()))
				cancel()
			}
		}()
	}

	router := router.NewRouter(
		logger.With("service", "router"),
		auditor,
		st,
		cfg.SecretKey,
	)

	if err := router.Run(ctx, cfg.Address); err != nil {
		return err
	}

	logger.Info("server shut down")
	return nil
}
