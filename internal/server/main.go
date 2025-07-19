package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fragpit/yandex-go-dev-metrics/internal/cacher"
	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	cfg := config.NewServerConfig()
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

	st := memstorage.NewMemoryStorage()
	router := router.NewRouter(
		logger.With("service", "router"),
		st,
	)

	logger.Info("starting server", slog.String("address", cfg.Address))

	cs := cacher.NewCacher(
		ctx,
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
		var metricsList []*model.Metrics
		if metricsList, err = cs.Restore(); err != nil {
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
	}

	go func() {
		if err := cs.Run(); err != nil {
			logger.Error("cacher error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	if err := router.Run(ctx, cfg.Address); err != nil {
		return err
	}

	logger.Info("server shut down")
	return nil
}
