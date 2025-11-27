package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewCustomPoller(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	poller := NewCustomPoller(logger)
	assert.NotNil(t, poller)
	assert.NotNil(t, poller.logger)
}

func TestCustomPoller_PollOnce(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	poller := NewCustomPoller(logger)

	ctx := context.Background()
	out := make(chan model.Metric, 10)

	err := poller.PollOnce(ctx, out)
	assert.NoError(t, err)

	close(out)

	metrics := make(map[string]model.Metric)
	for metric := range out {
		metrics[metric.GetID()] = metric
	}

	assert.Equal(t, 2, len(metrics), "should have 2 metrics")

	randomValue, ok := metrics["RandomValue"]
	assert.True(t, ok, "should have RandomValue metric")
	assert.Equal(t, model.GaugeType, randomValue.GetType())

	pollCount, ok := metrics["PollCount"]
	assert.True(t, ok, "should have PollCount metric")
	assert.Equal(t, model.CounterType, pollCount.GetType())
}
