package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var _ Poller = (*CustomPoller)(nil)

type PSPoller struct {
	l *slog.Logger
}

func NewPSPoller(l *slog.Logger) *PSPoller {
	return &PSPoller{
		l: l,
	}
}

func (p *PSPoller) PollOnce(
	ctx context.Context,
	out chan<- model.Metric,
) error {
	p.l.Info("starting poller")

	memStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error fetching memstat: %w", err)
	}

	cpuUtil, err := cpu.Percent(0, true)
	if err != nil {
		return fmt.Errorf("error fetching cpu utilization: %w", err)
	}

	metrics := []struct {
		tp    model.MetricType
		name  string
		value string
	}{
		{model.GaugeType, "TotalMemory", fmt.Sprintf("%d", memStat.Total)},
		{model.GaugeType, "FreeMemory", fmt.Sprintf("%d", memStat.Free)},
		{model.GaugeType, "CPUutilization1", fmt.Sprintf("%f", cpuUtil[0])},
	}

	for _, m := range metrics {
		if err := registerMetric(out, m.tp, m.name, m.value); err != nil {
			p.l.Error("failed to register metric",
				slog.String("name", m.name),
				slog.Any("error", err))
			return fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return nil
}
