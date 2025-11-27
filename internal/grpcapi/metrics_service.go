package grpcapi

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	pb "github.com/fragpit/yandex-go-dev-metrics/internal/proto"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

type MetricsService struct {
	pb.UnimplementedMetricsServer

	repo repository.Repository
}

func (m *MetricsService) UpdateMetrics(
	ctx context.Context,
	in *pb.UpdateMetricsRequest,
) (*pb.UpdateMetricsResponse, error) {
	pbMetrics := in.GetMetrics()

	metrics := make([]model.Metric, 0, len(pbMetrics))
	for _, m := range pbMetrics {
		var metric model.Metric

		switch m.GetType() {
		case pb.Metric_MTYPE_COUNTER:
			metric = &model.CounterMetric{
				ID:    m.GetId(),
				Value: int64(m.GetDelta()),
			}
		case pb.Metric_MTYPE_GAUGE:
			metric = &model.GaugeMetric{
				ID:    m.GetId(),
				Value: float64(m.GetValue()),
			}
		default:
			return nil, fmt.Errorf(
				"unknown metric type %s for %s",
				m.GetType(),
				m.GetId(),
			)
		}

		metrics = append(metrics, metric)
	}

	if err := m.repo.SetOrUpdateMetricBatch(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to update metrics: %w", err)
	}

	slog.Info("metrics updated", "count", len(metrics))

	return &pb.UpdateMetricsResponse{}, nil
}
