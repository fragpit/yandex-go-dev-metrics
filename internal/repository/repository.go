package repository

import "github.com/fragpit/yandex-go-dev-metrics/internal/model"

type Repository interface {
	GetMetrics() (map[string]model.Metrics, error)
	GetMetric(name string) (*model.Metrics, error)
	SetMetric(metric *model.Metrics, value string) error
}
