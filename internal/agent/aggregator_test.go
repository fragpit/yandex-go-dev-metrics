package agent

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocks "github.com/fragpit/yandex-go-dev-metrics/internal/mocks/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

func TestNewAggregator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.DiscardHandler)
	repo := mocks.NewMockRepository(ctrl)

	aggregator := NewAggregator(logger, repo)

	require.NotNil(t, aggregator)
	assert.Equal(t, logger, aggregator.l)
	assert.Equal(t, repo, aggregator.repo)
}

func TestAggregator_RunAggregator_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.DiscardHandler)
	repo := mocks.NewMockRepository(ctrl)

	aggregator := NewAggregator(logger, repo)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		100*time.Millisecond,
	)
	defer cancel()

	in := make(chan model.Metric, 10)

	metric1 := &model.GaugeMetric{
		ID:    "metric1",
		Value: 10.5,
	}
	metric2 := &model.CounterMetric{
		ID:    "metric2",
		Value: 42,
	}

	repo.EXPECT().
		SetOrUpdateMetric(gomock.Any(), metric1).
		Return(nil).
		Times(1)
	repo.EXPECT().
		SetOrUpdateMetric(gomock.Any(), metric2).
		Return(nil).
		Times(1)

	errCh := make(chan error, 1)
	go func() {
		errCh <- aggregator.RunAggregator(ctx, in)
	}()

	in <- metric1
	in <- metric2

	<-ctx.Done()

	err := <-errCh
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestAggregator_RunAggregator_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.DiscardHandler)
	repo := mocks.NewMockRepository(ctrl)

	aggregator := NewAggregator(logger, repo)

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan model.Metric)

	errCh := make(chan error, 1)
	go func() {
		errCh <- aggregator.RunAggregator(ctx, in)
	}()

	cancel()

	err := <-errCh
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAggregator_RunAggregator_MultipleMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.DiscardHandler)
	repo := mocks.NewMockRepository(ctrl)

	aggregator := NewAggregator(logger, repo)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		200*time.Millisecond,
	)
	defer cancel()

	in := make(chan model.Metric, 100)

	repo.EXPECT().
		SetOrUpdateMetric(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(10)

	errCh := make(chan error, 1)
	go func() {
		errCh <- aggregator.RunAggregator(ctx, in)
	}()

	for i := 0; i < 10; i++ {
		value := float64(i)
		metric := &model.GaugeMetric{
			ID:    "test_metric",
			Value: value,
		}
		in <- metric
	}

	<-ctx.Done()

	err := <-errCh
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestAggregator_RunAggregator_EmptyChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.DiscardHandler)
	repo := mocks.NewMockRepository(ctrl)

	aggregator := NewAggregator(logger, repo)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		50*time.Millisecond,
	)
	defer cancel()

	in := make(chan model.Metric)

	errCh := make(chan error, 1)
	go func() {
		errCh <- aggregator.RunAggregator(ctx, in)
	}()

	<-ctx.Done()

	err := <-errCh
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
