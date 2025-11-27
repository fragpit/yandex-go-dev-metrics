package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

var _ Poller = (*CustomPoller)(nil)

type CustomPoller struct {
	logger *slog.Logger
}

func NewCustomPoller(logger *slog.Logger) *CustomPoller {
	return &CustomPoller{
		logger: logger,
	}
}

func (p *CustomPoller) PollOnce(
	ctx context.Context,
	out chan<- model.Metric,
) error {
	p.logger.Info("starting poller")

	randValue := rand.IntN(100)
	metrics := []struct {
		tp    model.MetricType
		name  string
		value string
	}{
		{model.GaugeType, "RandomValue", fmt.Sprintf("%d", randValue)},
		{model.CounterType, "PollCount", fmt.Sprintf("%d", counter.Swap(0))},
	}

	for _, m := range metrics {
		if err := registerMetric(out, m.tp, m.name, m.value); err != nil {
			p.logger.Error("failed to register metric",
				slog.String("name", m.name),
				slog.Any("error", err))
			return fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return nil
}
