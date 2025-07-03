package memstorage

import (
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

func (s *MemoryStorage) GetMetric(name string) (*model.Metrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[name]; ok {
		return m, nil
	} else {
		return nil, errors.New("metric id not found")
	}

}

func (s *MemoryStorage) SetMetric(metric *model.Metrics, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[metric.ID]; ok {
		if m.MType != metric.MType {
			return errors.New("metric already exist with another type")
		}
		if err := m.SetMetricValue(value); err != nil {
			return err
		}
	} else {
		if err := metric.SetMetricValue(value); err != nil {
			return err
		}
		s.Metrics[metric.ID] = metric
	}

	return nil
}

func (s *MemoryStorage) GetMetrics() (map[string]*model.Metrics, error) {
	return s.Metrics, nil
}
