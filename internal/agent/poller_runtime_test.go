package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewRuntimePoller(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	poller := NewRuntimePoller(logger)
	assert.NotNil(t, poller)
	assert.NotNil(t, poller.logger)
}

func TestRuntimePoller_PollOnce(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	poller := NewRuntimePoller(logger)

	ctx := context.Background()
	out := make(chan model.Metric, 100)

	err := poller.PollOnce(ctx, out)
	assert.NoError(t, err)

	close(out)

	metricsCount := 0
	for metric := range out {
		assert.NotEmpty(t, metric.GetID())
		assert.Equal(t, model.GaugeType, metric.GetType())
		assert.NotEmpty(t, metric.GetValue())
		metricsCount++
	}

	assert.Greater(
		t,
		metricsCount,
		20,
		"should have collected multiple runtime metrics",
	)
}
