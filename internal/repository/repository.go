package repository

import (
	"context"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

type Repository interface {
	GetMetrics(ctx context.Context) (map[string]model.Metric, error)
	GetMetric(ctx context.Context, name string) (model.Metric, error)
	SetOrUpdateMetric(ctx context.Context, metric model.Metric) error
	Initialize([]model.Metric) error
	Ping(ctx context.Context) error
	Close() error
}
