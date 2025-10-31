package agent

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPSPoller(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	poller := NewPSPoller(logger)

	assert.NotNil(t, poller)
	assert.NotNil(t, poller.l)
}

func TestPSPoller_PollOnce(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	poller := NewPSPoller(logger)

	ctx := context.Background()
	out := make(chan model.Metric, 10)

	err := poller.PollOnce(ctx, out)
	require.NoError(t, err)

	close(out)

	// Проверяем, что были собраны метрики
	metricsCount := 0
	for range out {
		metricsCount++
	}

	assert.Greater(t, metricsCount, 0, "должны быть собраны метрики")
}
