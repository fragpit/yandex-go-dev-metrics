package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

type mockPoller struct {
	callCount int
	err       error
}

func (m *mockPoller) PollOnce(
	ctx context.Context,
	out chan<- model.Metric,
) error {
	m.callCount++
	if m.err != nil {
		return m.err
	}

	metric, _ := model.NewMetric("test_metric", model.GaugeType)
	_ = metric.SetValue("42")
	out <- metric
	return nil
}

func TestPoll(t *testing.T) {
	tests := []struct {
		name          string
		pollerErr     error
		expectedError bool
		duration      time.Duration
	}{
		{
			name:          "successful poll with timeout",
			pollerErr:     nil,
			expectedError: true,
			duration:      10 * time.Millisecond,
		},
		{
			name:          "poll with error",
			pollerErr:     errors.New("poll error"),
			expectedError: true,
			duration:      10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				50*time.Millisecond,
			)
			defer cancel()

			out := make(chan model.Metric, 10)
			poller := &mockPoller{err: tt.pollerErr}

			err := Poll(ctx, out, tt.duration, poller)

			if tt.expectedError {
				assert.Error(t, err)
			}

			if tt.pollerErr == nil {
				assert.True(
					t,
					poller.callCount >= 1,
					"poller should be called at least once",
				)
			}
		})
	}
}

func TestPoll_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	out := make(chan model.Metric, 10)
	poller := &mockPoller{}

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := Poll(ctx, out, 10*time.Millisecond, poller)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, poller.callCount >= 1)
}

func TestRegisterMetric(t *testing.T) {
	tests := []struct {
		name       string
		tp         model.MetricType
		metricName string
		value      string
		expectErr  bool
	}{
		{
			name:       "valid counter metric",
			tp:         model.CounterType,
			metricName: "test_counter",
			value:      "100",
			expectErr:  false,
		},
		{
			name:       "valid gauge metric",
			tp:         model.GaugeType,
			metricName: "test_gauge",
			value:      "42.5",
			expectErr:  false,
		},
		{
			name:       "invalid metric type",
			tp:         model.MetricType("invalid"),
			metricName: "test_metric",
			value:      "100",
			expectErr:  true,
		},
		{
			name:       "invalid counter value",
			tp:         model.CounterType,
			metricName: "test_counter",
			value:      "invalid",
			expectErr:  true,
		},
		{
			name:       "invalid gauge value",
			tp:         model.GaugeType,
			metricName: "test_gauge",
			value:      "invalid",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan model.Metric, 1)

			err := registerMetric(ch, tt.tp, tt.metricName, tt.value)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				select {
				case metric := <-ch:
					assert.Equal(t, tt.metricName, metric.GetID())
					assert.Equal(t, tt.tp, metric.GetType())
					assert.Equal(t, tt.value, metric.GetValue())
				case <-time.After(100 * time.Millisecond):
					t.Fatal("timeout waiting for metric")
				}
			}
		})
	}
}
