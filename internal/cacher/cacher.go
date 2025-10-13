package cacher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

type Cacher struct {
	logger  *slog.Logger
	storage repository.Repository

	filename string
	interval time.Duration
}

func NewCacher(
	logger *slog.Logger,
	storage repository.Repository,

	filename string,
	interval time.Duration,
) *Cacher {
	return &Cacher{
		logger:  logger,
		storage: storage,

		filename: filename,
		interval: interval,
	}
}

func (s *Cacher) Run(ctx context.Context) error {
	s.logger.Info("cacher started")
	defer s.logger.Info("cacher stopped")

	if err := runPeriodically(ctx, s.saveMetrics, s.interval); err != nil {
		return fmt.Errorf("error running cacher: %w", err)
	}

	return nil
}

func (s *Cacher) Restore() ([]model.Metric, error) {
	file, err := os.Open(s.filename)
	if os.IsNotExist(err) {
		s.logger.Info("no backup file found, skipping restore")
		return nil, nil
	}
	if err != nil {
		s.logger.Error(
			"failed to open file for restoring metrics",
			slog.String("filename", s.filename),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to open file for restoring metrics: %w", err)
	}
	defer file.Close()

	var metricsList []model.Metric
	decoder := json.NewDecoder(file)

	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	for decoder.More() {
		var metric model.Metrics
		if err := decoder.Decode(&metric); err != nil {
			if err.Error() == "EOF" {
				break
			}

			s.logger.Error(
				"failed to decode metric from file",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("error decoding json: %w", err)
		}

		m, err := model.MetricFromJSON(&metric)
		if err != nil {
			s.logger.Error(
				"failed to convert metric from json",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to convert metric from json: %w", err)
		}

		metricsList = append(metricsList, m)
	}

	return metricsList, nil
}

func (s *Cacher) saveMetrics(ctx context.Context) error {
	s.logger.Info("saving metrics")
	metrics, err := s.storage.GetMetrics(ctx)
	if err != nil {
		s.logger.Error("failed to get metrics", slog.String("error", err.Error()))
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(metrics) == 0 {
		s.logger.Info("no metrics to save")
		return nil
	}

	metricsList := make([]model.Metrics, 0, len(metrics))
	for _, metric := range metrics {
		metricsList = append(metricsList, *metric.ToJSON())
	}

	data, err := json.Marshal(metricsList)
	if err != nil {
		s.logger.Error(
			"failed to marshal metrics",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	file, err := os.OpenFile(s.filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		s.logger.Error(
			"failed to open file for saving metrics",
			slog.String("filename", s.filename),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to open file for saving metrics: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		s.logger.Error(
			"failed to write metrics to file",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to write metrics to file: %w", err)
	}

	if err := file.Sync(); err != nil {
		s.logger.Error(
			"failed to sync file",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to sync file: %w", err)
	}
	s.logger.Info("metrics saved", slog.Int("count", len(metricsList)))

	return nil
}

func runPeriodically(
	ctx context.Context,
	f func(ctx context.Context) error,
	period time.Duration,
) error {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				continue
			}
			if err := f(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
