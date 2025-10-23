package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetric(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		metricType MetricType
		wantErr    bool
	}{
		{
			name:       "valid counter metric",
			id:         "test_counter",
			metricType: CounterType,
			wantErr:    false,
		},
		{
			name:       "valid gauge metric",
			id:         "test_gauge",
			metricType: GaugeType,
			wantErr:    false,
		},
		{
			name:       "invalid metric type",
			id:         "test_metric",
			metricType: MetricType("invalid"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := NewMetric(tt.id, tt.metricType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, metric)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metric)
				assert.Equal(t, tt.id, metric.GetID())
				assert.Equal(t, tt.metricType, metric.GetType())
			}
		})
	}
}

func TestCounterMetric_SetValue(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		setValue string
		expected string
		wantErr  bool
	}{
		{
			name:     "set valid value",
			initial:  "0",
			setValue: "100",
			expected: "100",
			wantErr:  false,
		},
		{
			name:     "add to existing value",
			initial:  "50",
			setValue: "30",
			expected: "80",
			wantErr:  false,
		},
		{
			name:     "set invalid value",
			initial:  "0",
			setValue: "invalid",
			expected: "0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := NewMetric("test_counter", CounterType)
			require.NoError(t, err)

			_ = metric.SetValue(tt.initial)
			err = metric.SetValue(tt.setValue)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, metric.GetValue())
			}
		})
	}
}

func TestGaugeMetric_SetValue(t *testing.T) {
	tests := []struct {
		name     string
		setValue string
		wantErr  bool
	}{
		{
			name:     "set valid float value",
			setValue: "42.5",
			wantErr:  false,
		},
		{
			name:     "set valid integer value",
			setValue: "100",
			wantErr:  false,
		},
		{
			name:     "set invalid value",
			setValue: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := NewMetric("test_gauge", GaugeType)
			require.NoError(t, err)

			err = metric.SetValue(tt.setValue)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.setValue, metric.GetValue())
			}
		})
	}
}

func TestMetric_ToJSON(t *testing.T) {
	t.Run("counter to json", func(t *testing.T) {
		metric, _ := NewMetric("test_counter", CounterType)
		_ = metric.SetValue("100")

		json := metric.ToJSON()
		assert.Equal(t, "test_counter", json.ID)
		assert.Equal(t, "counter", json.MType)
		require.NotNil(t, json.Delta)
		assert.Equal(t, int64(100), *json.Delta)
		assert.Nil(t, json.Value)
	})

	t.Run("gauge to json", func(t *testing.T) {
		metric, _ := NewMetric("test_gauge", GaugeType)
		_ = metric.SetValue("42.5")

		json := metric.ToJSON()
		assert.Equal(t, "test_gauge", json.ID)
		assert.Equal(t, "gauge", json.MType)
		require.NotNil(t, json.Value)
		assert.Equal(t, 42.5, *json.Value)
		assert.Nil(t, json.Delta)
	})
}

func TestMetricFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    *Metrics
		wantErr bool
		check   func(*testing.T, Metric)
	}{
		{
			name: "valid counter from json",
			json: &Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: func() *int64 { v := int64(100); return &v }(),
			},
			wantErr: false,
			check: func(t *testing.T, m Metric) {
				assert.Equal(t, "test_counter", m.GetID())
				assert.Equal(t, CounterType, m.GetType())
				assert.Equal(t, "100", m.GetValue())
			},
		},
		{
			name: "valid gauge from json",
			json: &Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: func() *float64 { v := 42.5; return &v }(),
			},
			wantErr: false,
			check: func(t *testing.T, m Metric) {
				assert.Equal(t, "test_gauge", m.GetID())
				assert.Equal(t, GaugeType, m.GetType())
				assert.Equal(t, "42.5", m.GetValue())
			},
		},
		{
			name: "invalid metric type",
			json: &Metrics{
				ID:    "test",
				MType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "counter without delta",
			json: &Metrics{
				ID:    "test_counter",
				MType: "counter",
			},
			wantErr: false,
			check: func(t *testing.T, m Metric) {
				assert.Equal(t, "0", m.GetValue())
			},
		},
		{
			name: "gauge without value",
			json: &Metrics{
				ID:    "test_gauge",
				MType: "gauge",
			},
			wantErr: false,
			check: func(t *testing.T, m Metric) {
				assert.Equal(t, "0", m.GetValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := MetricFromJSON(tt.json)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metric)
				if tt.check != nil {
					tt.check(t, metric)
				}
			}
		})
	}
}
