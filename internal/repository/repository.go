package repository

import (
	"context"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

type Repository interface {
	GetMetrics(ctx context.Context) (map[string]*model.Metrics, error)
	GetMetric(ctx context.Context, name string) (*model.Metrics, error)
	SetOrUpdateMetric(ctx context.Context, metric *model.Metrics) error
}
