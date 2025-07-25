package cacher

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

type Cacher struct {
	ctx     context.Context
	logger  *slog.Logger
	storage repository.Repository

	filename string
	interval time.Duration
}

func NewCacher(
	ctx context.Context,
	logger *slog.Logger,
	storage repository.Repository,

	filename string,
	interval time.Duration,
) *Cacher {
	return &Cacher{
		ctx:     ctx,
		logger:  logger,
		storage: storage,

		filename: filename,
		interval: interval,
	}
}

func (s *Cacher) Run() error {
	s.logger.Info("cacher started")
	defer s.logger.Info("cacher stopped")

	runPeriodically(s.ctx, s.saveMetrics, s.interval)
	return nil
}

func (s *Cacher) Restore() ([]*model.Metrics, error) {
	file, err := os.Open(s.filename)
	if err != nil {
		s.logger.Error(
			"failed to open file for restoring metrics",
			slog.String("filename", s.filename),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	defer file.Close()

	var metricsList []*model.Metrics
	if err := json.NewDecoder(file).Decode(&metricsList); err != nil {
		s.logger.Error(
			"failed to decode metrics from file",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return metricsList, nil
}

func (s *Cacher) saveMetrics(ctx context.Context) {
	s.logger.Info("saving metrics")
	metrics, err := s.storage.GetMetrics(ctx)
	if err != nil {
		s.logger.Error("failed to get metrics", slog.String("error", err.Error()))
		return
	}

	if len(metrics) == 0 {
		s.logger.Info("no metrics to save")
		return
	}

	var metricsList []*model.Metrics
	for _, metric := range metrics {
		metricsList = append(metricsList, metric)
	}

	data, err := json.Marshal(metricsList)
	if err != nil {
		s.logger.Error(
			"failed to marshal metrics",
			slog.String("error", err.Error()),
		)
		return
	}

	file, err := os.OpenFile(s.filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		s.logger.Error(
			"failed to open file for saving metrics",
			slog.String("filename", s.filename),
			slog.String("error", err.Error()),
		)
		return
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		s.logger.Error(
			"failed to write metrics to file",
			slog.String("error", err.Error()),
		)
		return
	}

	if err := file.Sync(); err != nil {
		s.logger.Error(
			"failed to sync file",
			slog.String("error", err.Error()),
		)
		return
	}
	s.logger.Info("metrics saved", slog.Int("count", len(metricsList)))
}

func runPeriodically(
	ctx context.Context,
	f func(ctx context.Context),
	period time.Duration,
) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				continue
			}
			f(ctx)
		case <-ctx.Done():
			return
		}
	}
}
