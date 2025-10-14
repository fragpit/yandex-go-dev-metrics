package memstorage

import (
	"context"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	assert.NotNil(t, storage)
	assert.NotNil(t, storage.Metrics)
	assert.Equal(t, 0, len(storage.Metrics))
}

func TestMemoryStorage_SetOrUpdateMetric(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*MemoryStorage)
		metric    func() model.Metric
		wantErr   bool
		errMsg    string
		checkFunc func(*testing.T, *MemoryStorage)
	}{
		{
			name:  "add new gauge metric",
			setup: func(s *MemoryStorage) {},
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_gauge", model.GaugeType)
				_ = m.SetValue("42.5")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				m, err := s.GetMetric(context.Background(), "test_gauge")
				require.NoError(t, err)
				assert.Equal(t, "42.5", m.GetValue())
			},
		},
		{
			name:  "add new counter metric",
			setup: func(s *MemoryStorage) {},
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_counter", model.CounterType)
				_ = m.SetValue("100")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				m, err := s.GetMetric(context.Background(), "test_counter")
				require.NoError(t, err)
				assert.Equal(t, "100", m.GetValue())
			},
		},
		{
			name: "update existing counter metric",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("test_counter", model.CounterType)
				_ = m.SetValue("50")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_counter", model.CounterType)
				_ = m.SetValue("25")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				m, err := s.GetMetric(context.Background(), "test_counter")
				require.NoError(t, err)
				assert.Equal(t, "75", m.GetValue())
			},
		},
		{
			name: "update existing gauge metric",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("test_gauge", model.GaugeType)
				_ = m.SetValue("10.5")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_gauge", model.GaugeType)
				_ = m.SetValue("20.7")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				m, err := s.GetMetric(context.Background(), "test_gauge")
				require.NoError(t, err)
				assert.Equal(t, "20.7", m.GetValue())
			},
		},
		{
			name: "error on type mismatch",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("test_metric", model.GaugeType)
				_ = m.SetValue("10.5")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_metric", model.CounterType)
				_ = m.SetValue("100")
				return m
			},
			wantErr: true,
			errMsg:  "metric already exist with another type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			err := storage.SetOrUpdateMetric(context.Background(), tt.metric())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, storage)
				}
			}
		})
	}
}

func TestMemoryStorage_GetMetric(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MemoryStorage)
		id      string
		wantErr bool
	}{
		{
			name: "get existing metric",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("test_metric", model.GaugeType)
				_ = m.SetValue("42.5")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			id:      "test_metric",
			wantErr: false,
		},
		{
			name:    "get non-existing metric",
			setup:   func(s *MemoryStorage) {},
			id:      "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			metric, err := storage.GetMetric(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metric)
				assert.Equal(t, tt.id, metric.GetID())
			}
		})
	}
}

func TestMemoryStorage_SetOrUpdateMetricBatch(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*MemoryStorage)
		metrics   func() []model.Metric
		wantErr   bool
		checkFunc func(*testing.T, *MemoryStorage)
	}{
		{
			name:  "add multiple new metrics",
			setup: func(s *MemoryStorage) {},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("gauge1", model.GaugeType)
				_ = m1.SetValue("10.5")
				m2, _ := model.NewMetric("counter1", model.CounterType)
				_ = m2.SetValue("100")
				return []model.Metric{m1, m2}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				assert.Equal(t, 2, len(s.Metrics))
			},
		},
		{
			name: "update existing and add new metrics",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("existing", model.CounterType)
				_ = m.SetValue("50")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("existing", model.CounterType)
				_ = m1.SetValue("30")
				m2, _ := model.NewMetric("new_metric", model.GaugeType)
				_ = m2.SetValue("42.5")
				return []model.Metric{m1, m2}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *MemoryStorage) {
				assert.Equal(t, 2, len(s.Metrics))
				m, _ := s.GetMetric(context.Background(), "existing")
				assert.Equal(t, "80", m.GetValue())
			},
		},
		{
			name: "error on type mismatch in batch",
			setup: func(s *MemoryStorage) {
				m, _ := model.NewMetric("test_metric", model.GaugeType)
				_ = m.SetValue("10.5")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("test_metric", model.CounterType)
				_ = m1.SetValue("100")
				return []model.Metric{m1}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			err := storage.SetOrUpdateMetricBatch(context.Background(), tt.metrics())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, storage)
				}
			}
		})
	}
}

func TestMemoryStorage_GetMetrics(t *testing.T) {
	storage := NewMemoryStorage()

	m1, _ := model.NewMetric("gauge1", model.GaugeType)
	_ = m1.SetValue("10.5")
	m2, _ := model.NewMetric("counter1", model.CounterType)
	_ = m2.SetValue("100")

	_ = storage.SetOrUpdateMetric(context.Background(), m1)
	_ = storage.SetOrUpdateMetric(context.Background(), m2)

	metrics, err := storage.GetMetrics(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(metrics))
}

func TestMemoryStorage_Initialize(t *testing.T) {
	storage := NewMemoryStorage()

	m1, _ := model.NewMetric("gauge1", model.GaugeType)
	_ = m1.SetValue("10.5")
	m2, _ := model.NewMetric("counter1", model.CounterType)
	_ = m2.SetValue("100")

	err := storage.Initialize([]model.Metric{m1, m2})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(storage.Metrics))

	metric, err := storage.GetMetric(context.Background(), "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, "10.5", metric.GetValue())
}

func TestMemoryStorage_Reset(t *testing.T) {
	storage := NewMemoryStorage()

	m, _ := model.NewMetric("test", model.GaugeType)
	_ = m.SetValue("10.5")
	_ = storage.SetOrUpdateMetric(context.Background(), m)

	assert.Equal(t, 1, len(storage.Metrics))

	err := storage.Reset()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(storage.Metrics))
}

func TestMemoryStorage_Ping(t *testing.T) {
	storage := NewMemoryStorage()
	err := storage.Ping(context.Background())
	assert.NoError(t, err)
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	err := storage.Close(context.Background())
	assert.NoError(t, err)
}
