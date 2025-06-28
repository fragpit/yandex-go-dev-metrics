package memstorage

import (
	"errors"
	"strconv"
	"sync"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

type MemoryStorage struct {
	mu      sync.RWMutex
	Metrics map[string]model.Metrics
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		mu:      sync.RWMutex{},
		Metrics: map[string]model.Metrics{},
	}
}

func (s *MemoryStorage) SetMetric(metric *model.Metrics, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m, ok := s.Metrics[metric.ID]; ok {
		if m.MType != metric.MType {
			return errors.New("metric already exist with another type")
		}
	}

	if metric.MType == "gauge" {
		if err := s.setGauge(metric, value); err != nil {
			return err
		}
	}

	if metric.MType == "counter" {
		if err := s.setCounter(metric, value); err != nil {
			return err
		}
	}

	return nil
}

func (s *MemoryStorage) setGauge(metric *model.Metrics, value string) error {
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	if m, ok := s.Metrics[metric.ID]; !ok {
		metric.Value = &parsedValue
		s.Metrics[metric.ID] = *metric
	} else {
		*m.Value = parsedValue
	}

	return nil
}

func (s *MemoryStorage) setCounter(metric *model.Metrics, value string) error {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	if m, ok := s.Metrics[metric.ID]; !ok {
		metric.Delta = &parsedValue
		s.Metrics[metric.ID] = *metric
	} else {
		*m.Delta += parsedValue
	}

	return nil
}
