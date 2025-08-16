package agent

import (
	"context"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

type Poller interface {
	PollOnce(ctx context.Context, out chan<- model.Metric) error
}

func Poll(
	ctx context.Context,
	out chan<- model.Metric,
	interval time.Duration,
	p Poller,
) error {
	t := time.NewTicker(interval)
	defer t.Stop()

	counter.Add(1)
	if err := p.PollOnce(ctx, out); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			counter.Add(1)
			if err := p.PollOnce(ctx, out); err != nil {
				return err
			}
		}
	}
}

func registerMetric(
	ch chan<- model.Metric,
	tp model.MetricType,
	name, value string,
) error {
	metric, err := model.NewMetric(name, tp)
	if err != nil {
		return err
	}

	if err := metric.SetValue(value); err != nil {
		return err
	}

	ch <- metric

	return nil
}
