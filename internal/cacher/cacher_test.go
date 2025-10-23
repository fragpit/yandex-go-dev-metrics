package cacher

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacher(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	storage := memstorage.NewMemoryStorage()
	cacher := NewCacher(logger, storage, "test.json", time.Second)

	assert.NotNil(t, cacher)
	assert.Equal(t, "test.json", cacher.filename)
	assert.Equal(t, time.Second, cacher.interval)
}

func TestCacher_Run(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "metrics-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage := memstorage.NewMemoryStorage()
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	cacher := NewCacher(logger, storage, tmpFile.Name(), 50*time.Millisecond)

	m1, _ := model.NewMetric("test_gauge", model.GaugeType)
	_ = m1.SetValue("42.5")
	_ = storage.SetOrUpdateMetric(context.Background(), m1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = cacher.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	require.NoError(t, err)
	assert.NotEmpty(t, metrics)
}

func TestCacher_Restore(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() string
		expectedCount int
		expectError   bool
	}{
		{
			name: "restore valid metrics",
			setup: func() string {
				tmpFile, _ := os.CreateTemp("", "metrics-*.json")
				defer tmpFile.Close()

				m1, _ := model.NewMetric("gauge1", model.GaugeType)
				_ = m1.SetValue("10.5")
				m2, _ := model.NewMetric("counter1", model.CounterType)
				_ = m2.SetValue("100")

				data, _ := json.Marshal([]model.Metrics{*m1.ToJSON(), *m2.ToJSON()})
				_, _ = tmpFile.Write(data)

				return tmpFile.Name()
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "restore from non-existent file",
			setup: func() string {
				return "/tmp/non_existent_file.json"
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "restore from invalid json",
			setup: func() string {
				tmpFile, _ := os.CreateTemp("", "metrics-*.json")
				defer tmpFile.Close()
				_, _ = tmpFile.WriteString("{invalid json")
				return tmpFile.Name()
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setup()
			if tt.name != "restore from non-existent file" {
				defer os.Remove(filename)
			}

			logger := slog.New(
				slog.NewTextHandler(
					os.Stdout,
					&slog.HandlerOptions{Level: slog.LevelError},
				),
			)
			storage := memstorage.NewMemoryStorage()
			cacher := NewCacher(logger, storage, filename, time.Second)

			metrics, err := cacher.Restore()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(metrics))
			}
		})
	}
}

func TestCacher_saveMetrics(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*memstorage.MemoryStorage)
		expectError bool
	}{
		{
			name: "save metrics successfully",
			setup: func(s *memstorage.MemoryStorage) {
				m1, _ := model.NewMetric("gauge1", model.GaugeType)
				_ = m1.SetValue("10.5")
				_ = s.SetOrUpdateMetric(context.Background(), m1)
			},
			expectError: false,
		},
		{
			name: "save empty metrics",
			setup: func(s *memstorage.MemoryStorage) {
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "metrics-*.json")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			storage := memstorage.NewMemoryStorage()
			tt.setup(storage)

			logger := slog.New(
				slog.NewTextHandler(
					os.Stdout,
					&slog.HandlerOptions{Level: slog.LevelError},
				),
			)
			cacher := NewCacher(logger, storage, tmpFile.Name(), time.Second)

			err = cacher.saveMetrics(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
