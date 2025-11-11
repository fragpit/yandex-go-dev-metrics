package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/stretchr/testify/assert"
)

func TestAgentComponents(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	storage := memstorage.NewMemoryStorage()

	t.Run("create runtime poller", func(t *testing.T) {
		poller := NewRuntimePoller(logger)
		assert.NotNil(t, poller)
	})

	t.Run("create custom poller", func(t *testing.T) {
		poller := NewCustomPoller(logger)
		assert.NotNil(t, poller)
	})

	t.Run("create aggregator", func(t *testing.T) {
		aggregator := NewAggregator(logger, storage)
		assert.NotNil(t, aggregator)
	})

	t.Run("create reporter", func(t *testing.T) {
		reporter := NewReporter(
			logger,
			storage,
			"http://localhost:8080",
			nil,
			1,
			"",
		)
		assert.NotNil(t, reporter)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assert.NotNil(t, ctx)
		cancel()
		assert.Error(t, ctx.Err())
	})
}
