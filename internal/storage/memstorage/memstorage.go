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
	Metrics map[string]*model.Metrics
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		mu:      sync.RWMutex{},
		Metrics: map[string]*model.Metrics{},
	}
}

func (s *MemoryStorage) GetMetric(
	ctx context.Context,
	name string,
) (*model.Metrics, error) {
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
	metric *model.Metrics,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[metric.ID]; ok {
		if m.MType != metric.MType {
			return errors.New("metric already exist with another type")
		}

		if metric.MType == string(model.GaugeType) {
			s.Metrics[metric.ID] = metric
			return nil
		}

		if metric.MType == string(model.CounterType) {
			if m.Delta == nil {
				m.Delta = metric.Delta
			} else {
				*m.Delta += *metric.Delta
			}
		}
	} else {
		s.Metrics[metric.ID] = metric
	}

	return nil
}

func (s *MemoryStorage) GetMetrics(
	ctx context.Context,
) (map[string]*model.Metrics, error) {
	return s.Metrics, nil
}

func (s *MemoryStorage) Initialize(metrics []*model.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(metrics) == 0 {
		return nil
	}

	for _, metric := range metrics {
		if metric == nil || metric.ID == "" {
			continue
		}
		s.Metrics[metric.ID] = metric
	}

	return nil
}
