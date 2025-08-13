package repository

import (
	"context"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

//go:generate go tool mockgen -package=mocks -destination=../mocks/repository/repository_mock.go . Repository
type Repository interface {
	GetMetrics(ctx context.Context) (map[string]model.Metric, error)
	GetMetric(ctx context.Context, name string) (model.Metric, error)
	SetOrUpdateMetric(ctx context.Context, metric model.Metric) error
	SetOrUpdateMetricBatch(ctx context.Context, metrics []model.Metric) error
	Initialize([]model.Metric) error
	Reset() error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
