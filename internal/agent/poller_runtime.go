package agent

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

var _ Poller = (*RuntimePoller)(nil)

type RuntimePoller struct {
	logger *slog.Logger
}

func NewRuntimePoller(logger *slog.Logger) *RuntimePoller {
	return &RuntimePoller{
		logger: logger,
	}
}

func (p *RuntimePoller) PollOnce(
	ctx context.Context,
	out chan<- model.Metric,
) error {
	p.logger.Info("starting poller")

	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	metrics := []struct {
		tp    model.MetricType
		name  string
		value string
	}{
		{model.GaugeType, "Alloc", fmt.Sprintf("%d", mstat.Alloc)},
		{model.GaugeType, "BuckHashSys", fmt.Sprintf("%d", mstat.BuckHashSys)},
		{model.GaugeType, "Frees", fmt.Sprintf("%d", mstat.Frees)},
		{model.GaugeType, "GCCPUFraction", fmt.Sprintf("%f", mstat.GCCPUFraction)},
		{model.GaugeType, "GCSys", fmt.Sprintf("%d", mstat.GCSys)},
		{model.GaugeType, "HeapAlloc", fmt.Sprintf("%d", mstat.HeapAlloc)},
		{model.GaugeType, "HeapIdle", fmt.Sprintf("%d", mstat.HeapIdle)},
		{model.GaugeType, "HeapInuse", fmt.Sprintf("%d", mstat.HeapInuse)},
		{model.GaugeType, "HeapObjects", fmt.Sprintf("%d", mstat.HeapObjects)},
		{model.GaugeType, "HeapReleased", fmt.Sprintf("%d", mstat.HeapReleased)},
		{model.GaugeType, "HeapSys", fmt.Sprintf("%d", mstat.HeapSys)},
		{model.GaugeType, "LastGC", fmt.Sprintf("%d", mstat.LastGC)},
		{model.GaugeType, "Lookups", fmt.Sprintf("%d", mstat.Lookups)},
		{model.GaugeType, "MCacheInuse", fmt.Sprintf("%d", mstat.MCacheInuse)},
		{model.GaugeType, "MCacheSys", fmt.Sprintf("%d", mstat.MCacheSys)},
		{model.GaugeType, "MSpanInuse", fmt.Sprintf("%d", mstat.MSpanInuse)},
		{model.GaugeType, "MSpanSys", fmt.Sprintf("%d", mstat.MSpanSys)},
		{model.GaugeType, "Mallocs", fmt.Sprintf("%d", mstat.Mallocs)},
		{model.GaugeType, "NextGC", fmt.Sprintf("%d", mstat.NextGC)},
		{model.GaugeType, "NumForcedGC", fmt.Sprintf("%d", mstat.NumForcedGC)},
		{model.GaugeType, "NumGC", fmt.Sprintf("%d", mstat.NumGC)},
		{model.GaugeType, "OtherSys", fmt.Sprintf("%d", mstat.OtherSys)},
		{model.GaugeType, "PauseTotalNs", fmt.Sprintf("%d", mstat.PauseTotalNs)},
		{model.GaugeType, "StackInuse", fmt.Sprintf("%d", mstat.StackInuse)},
		{model.GaugeType, "StackSys", fmt.Sprintf("%d", mstat.StackSys)},
		{model.GaugeType, "Sys", fmt.Sprintf("%d", mstat.Sys)},
		{model.GaugeType, "TotalAlloc", fmt.Sprintf("%d", mstat.TotalAlloc)},
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
