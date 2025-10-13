package cacher

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	mocks "github.com/fragpit/yandex-go-dev-metrics/internal/mocks/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"go.uber.org/mock/gomock"
)

func BenchmarkCacher_saveMetrics(b *testing.B) {
	count := 100
	metrics := make(map[string]model.Metric, count)
	for i := 0; i < count; i++ {
		name := "Test" + strconv.Itoa(i)
		metric, _ := model.NewMetric(name, model.GaugeType)
		_ = metric.SetValue("100.0")
		metrics[name] = metric
	}

	logger := slog.New(slog.DiscardHandler)
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	storeMock := mocks.NewMockRepository(ctrl)
	storeMock.EXPECT().GetMetrics(gomock.Any()).Return(metrics, nil).AnyTimes()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		file, err := os.CreateTemp("", "benchmark-*")
		if err != nil {
			b.Fatal(err)
		}
		filename := file.Name()
		file.Close()

		cr := &Cacher{
			logger:   logger,
			storage:  storeMock,
			filename: filename,
			interval: 1 * time.Second,
		}
		b.StartTimer()

		_ = cr.saveMetrics(context.Background())

		b.StopTimer()
		_ = os.Remove(filename)
		b.StartTimer()
	}
}
