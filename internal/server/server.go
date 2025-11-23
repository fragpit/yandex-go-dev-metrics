package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fragpit/yandex-go-dev-metrics/internal/audit"
	"github.com/fragpit/yandex-go-dev-metrics/internal/cacher"
	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/grpcapi"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/postgresql"
	"golang.org/x/sync/errgroup"
)

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
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

	var repo repository.Repository
	if cfg.DatabaseDSN != "" {
		if repo, err = postgresql.NewStorage(ctx, cfg.DatabaseDSN); err != nil {
			return err
		}
	} else {
		repo = memstorage.NewMemoryStorage()
	}
	defer repo.Close(ctx)

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

	eg, ctx := errgroup.WithContext(ctx)

	if cfg.DatabaseDSN == "" {
		cr := cacher.NewCacher(
			logger,
			repo,
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
			if err = repo.Initialize(metricsList); err != nil {
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

		eg.Go(func() error {
			if err := cr.Run(ctx); err != nil {
				logger.Error("cacher error", slog.String("error", err.Error()))
				return err
			}
			return nil
		})
	}

	if len(cfg.Address) > 0 {
		router, err := router.NewRouter(
			logger.With("service", "router"),
			auditor,
			repo,
			[]byte(cfg.SecretKey),
			cfg.CryptoKey,
			cfg.TrustedSubnet,
		)
		if err != nil {
			return err
		}

		eg.Go(func() error {
			if err := router.Run(ctx, cfg.Address); err != nil {
				logger.Error("router error", slog.String("error", err.Error()))
				return err
			}
			return nil
		})
	}

	if len(cfg.GRPCAddress) > 0 {
		var opts []grpcapi.Option
		if cfg.TrustedSubnet != "" {
			opts = append(opts, grpcapi.WithTrustedSubnet(cfg.TrustedSubnet))
		}
		gapi, err := grpcapi.NewGRPCAPI(cfg.GRPCAddress, repo, opts...)
		if err != nil {
			logger.Error("failed to init grpc api", slog.String("error", err.Error()))
			return err
		}

		eg.Go(func() error {
			if err := gapi.Run(ctx); err != nil {
				logger.Error("grpc api error", slog.String("error", err.Error()))
				return err
			}

			return nil
		})
	}

	err = eg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("server shutdown with error", slog.Any("error", err))
		return err
	}

	logger.Info("server shut down")
	return nil
}
