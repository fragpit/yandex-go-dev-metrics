package memstorage

import (
	"context"
	"errors"
	"sync"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

var _ repository.Repository = (*MemoryStorage)(nil)

type MemoryStorage struct {
	mu      sync.RWMutex
	Metrics map[string]model.Metric
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		mu:      sync.RWMutex{},
		Metrics: map[string]model.Metric{},
	}
}

func (s *MemoryStorage) GetMetric(
	ctx context.Context,
	name string,
) (model.Metric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[name]; ok {
		return m, nil
	} else {
		return nil, errors.New("metric id not found")
	}
}

func (s *MemoryStorage) SetOrUpdateMetric(
	ctx context.Context,
	metric model.Metric,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[metric.GetID()]; ok {
		if m.GetType() != metric.GetType() {
			return errors.New("metric already exist with another type")
		}

		if err := m.SetValue(metric.GetValue()); err != nil {
			return err
		}

	} else {
		s.Metrics[metric.GetID()] = metric
	}

	return nil
}

func (s *MemoryStorage) GetMetrics(
	ctx context.Context,
) (map[string]model.Metric, error) {
	return s.Metrics, nil
}

func (s *MemoryStorage) Initialize(metrics []model.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, metric := range metrics {
		s.Metrics[metric.GetID()] = metric
	}

	return nil
}

func (s *MemoryStorage) Ping(_ context.Context) error {
	return nil
}

func (s *MemoryStorage) Close() error {
	return nil
}
