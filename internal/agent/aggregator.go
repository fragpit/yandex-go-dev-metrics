package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

type Aggregator struct {
	l    *slog.Logger
	repo repository.Repository
}

func NewAggregator(logger *slog.Logger, st repository.Repository) *Aggregator {
	return &Aggregator{
		l:    logger,
		repo: st,
	}
}

func (a *Aggregator) RunAggregator(
	ctx context.Context,
	in <-chan model.Metric,
) error {
	a.l.Info("starting aggregator")

	merge := func(metric model.Metric) error {
		_ = a.repo.SetOrUpdateMetric(ctx, metric)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case metric := <-in:
			if err := merge(metric); err != nil {
				return fmt.Errorf("error merging metrics: %w", err)
			}
		}
	}
}
