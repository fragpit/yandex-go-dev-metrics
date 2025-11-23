package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

const (
	clientPostTimeout = 5 * time.Second
)

var ErrUnknownTransport = errors.New("unknown transport type")

type Transport interface {
	SendMetrics(context.Context, map[string]model.Metric) error
	Close() error
}

// Reporter is responsible for reporting metrics to the server.
type Reporter struct {
	logger    *slog.Logger
	repo      repository.Repository
	transport Transport
}

// NewReporter creates a new Reporter instance.
func NewReporter(
	logger *slog.Logger,
	repo repository.Repository,
	serverURL string,
	secretKey []byte,
	rateLimit int,
	cryptoKey string,
	grpcServerAddress string,
) (*Reporter, error) {
	var t Transport
	switch {
	case len(grpcServerAddress) > 0:
		var err error
		if t, err = NewGRPCTransport(grpcServerAddress); err != nil {
			return nil, fmt.Errorf("failed to init grpc transport: %w", err)
		}
	case len(serverURL) > 0:
		t = &RESTTransport{
			serverURL: serverURL,
			secretKey: secretKey,
			rateLimit: rateLimit,
			cryptoKey: cryptoKey,
		}
	default:
		return nil, ErrUnknownTransport
	}

	return &Reporter{
		logger:    logger,
		repo:      repo,
		transport: t,
	}, nil
}

// RunReporter starts the reporting process at the specified interval.
func (r *Reporter) RunReporter(
	ctx context.Context,
	interval time.Duration,
) error {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			m, err := r.repo.GetMetrics(ctx)
			if err != nil {
				r.logger.Error("failed to get metrics",
					slog.Any("error", err))
				return fmt.Errorf("failed to get metrics: %w", err)
			}

			if err := r.repo.Reset(); err != nil {
				r.logger.Error("failed to reset map",
					slog.Any("error", err))
				return fmt.Errorf("failed to reset map: %w", err)
			}

			if err := r.transport.SendMetrics(ctx, m); err != nil {
				r.logger.Error("failed to report metrics",
					slog.Any("error", err))
				return fmt.Errorf("failed to report metrics: %w", err)
			}
		}
	}
}
