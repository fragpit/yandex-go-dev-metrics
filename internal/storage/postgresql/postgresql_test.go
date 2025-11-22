package postgresql

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgStorage *Storage
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	pgContainer, err := testcontainers.Run(
		ctx,
		"postgres:17.6-alpine",
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "metricstest",
		}),
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create pg container: %s", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	pgEndpoint, err := pgContainer.Endpoint(ctx, "tcp")
	if err != nil {
		log.Fatalf("failed to get pg endpoint: %s", err)
	}

	pgEndpoint = strings.TrimPrefix(pgEndpoint, "tcp://")
	pgHost, pgPort, err := net.SplitHostPort(pgEndpoint)
	if err != nil {
		log.Fatalf("failed to get pg port number: %s", err)
	}

	pgDSN := fmt.Sprintf(
		"postgresql://postgres:postgres@%s:%s/metricstest?sslmode=disable",
		pgHost,
		pgPort,
	)
	pgStorage, err = NewStorage(
		ctx,
		pgDSN,
	)
	if err != nil {
		log.Fatalf("fail to create storage: %s", err)
	}

	os.Exit(m.Run())
}

func TestStorage_Ping(t *testing.T) {
	t.Run("test ping", func(t *testing.T) {
		err := pgStorage.Ping(t.Context())
		assert.NoError(t, err)
	})
}

func TestStorage_SetOrUpdateMetric(t *testing.T) {
	tests := []struct {
		name      string
		metric    func() model.Metric
		wantErr   bool
		errMsg    string
		checkFunc func(*testing.T, *Storage)
	}{
		{
			name: "add new gauge metric",
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_gauge_1", model.GaugeType)
				_ = m.SetValue("42.5")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {
				m, err := s.GetMetric(context.Background(), "test_gauge_1")
				require.NoError(t, err)
				assert.Equal(t, "42.5", m.GetValue())
			},
		},
		{
			name: "update existing gauge metric",
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_gauge_1", model.GaugeType)
				_ = m.SetValue("20.7")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {
				m, err := s.GetMetric(context.Background(), "test_gauge_1")
				require.NoError(t, err)
				assert.Equal(t, "20.7", m.GetValue())
			},
		},
		{
			name: "add new counter metric",
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_counter_1", model.CounterType)
				_ = m.SetValue("100")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {
				m, err := s.GetMetric(context.Background(), "test_counter_1")
				require.NoError(t, err)
				assert.Equal(t, "100", m.GetValue())
			},
		},
		{
			name: "update existing counter metric",
			metric: func() model.Metric {
				m, _ := model.NewMetric("test_counter_1", model.CounterType)
				_ = m.SetValue("25")
				return m
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {
				m, err := s.GetMetric(context.Background(), "test_counter_1")
				require.NoError(t, err)
				assert.Equal(t, "125", m.GetValue())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pgStorage.SetOrUpdateMetric(t.Context(), tt.metric())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, pgStorage)
				}
			}
		})
	}
}

func TestStorage_GetMetric(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Storage)
		id      string
		wantErr bool
	}{
		{
			name: "get existing metric",
			setup: func(s *Storage) {
				m, _ := model.NewMetric("test_metric_2", model.GaugeType)
				_ = m.SetValue("42.5")
				_ = s.SetOrUpdateMetric(context.Background(), m)
			},
			id:      "test_metric_2",
			wantErr: false,
		},
		{
			name:    "get non-existing metric",
			setup:   func(s *Storage) {},
			id:      "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(pgStorage)

			metric, err := pgStorage.GetMetric(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metric)
				assert.Equal(t, tt.id, metric.GetID())
			}
		})
	}
}

func TestStorage_SetOrUpdateMetricBatch(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Storage)
		metrics   func() []model.Metric
		wantErr   bool
		checkFunc func(*testing.T, *Storage)
	}{
		{
			name:  "add multiple new metrics",
			setup: func(s *Storage) {},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("test_gauge_3", model.GaugeType)
				_ = m1.SetValue("10.5")
				m2, _ := model.NewMetric("test_counter_3", model.CounterType)
				_ = m2.SetValue("100")
				return []model.Metric{m1, m2}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {
				metrics, _ := s.GetMetrics(t.Context())
				assert.NotNil(t, metrics["test_gauge_3"].GetID())
				assert.Equal(t, "test_gauge_3", metrics["test_gauge_3"].GetID())
				assert.NotNil(t, metrics["test_counter_3"].GetID())
				assert.Equal(t, "test_counter_3", metrics["test_counter_3"].GetID())
			},
		},
		{
			name: "update existing and add new metrics",
			setup: func(s *Storage) {
				m, _ := model.NewMetric("existing_3", model.CounterType)
				_ = m.SetValue("50")
				_ = s.SetOrUpdateMetric(t.Context(), m)
			},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("existing_3", model.CounterType)
				_ = m1.SetValue("30")
				m2, _ := model.NewMetric("new_metric_3", model.GaugeType)
				_ = m2.SetValue("42.5")
				return []model.Metric{m1, m2}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, s *Storage) {

				m1, _ := s.GetMetric(t.Context(), "existing_3")
				m2, _ := s.GetMetric(t.Context(), "new_metric_3")
				assert.Equal(t, "80", m1.GetValue())
				assert.Equal(t, "42.5", m2.GetValue())
			},
		},
		{
			name: "error on type mismatch in batch",
			setup: func(s *Storage) {
				m, _ := model.NewMetric("test_metric_3", model.GaugeType)
				_ = m.SetValue("10.5")
				_ = s.SetOrUpdateMetric(t.Context(), m)
			},
			metrics: func() []model.Metric {
				m1, _ := model.NewMetric("test_metric_3", model.CounterType)
				_ = m1.SetValue("100")
				return []model.Metric{m1}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(pgStorage)

			err := pgStorage.SetOrUpdateMetricBatch(t.Context(), tt.metrics())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, pgStorage)
				}
			}
		})
	}
}

func TestStorage_GetMetrics(t *testing.T) {
	t.Run("test GetMetrics", func(t *testing.T) {
		m1, _ := model.NewMetric("test_gauge_4", model.GaugeType)
		_ = m1.SetValue("10.5")
		m2, _ := model.NewMetric("test_counter_4", model.CounterType)
		_ = m2.SetValue("100")

		var err error
		err = pgStorage.SetOrUpdateMetric(t.Context(), m1)
		require.NoError(t, err)
		err = pgStorage.SetOrUpdateMetric(t.Context(), m2)
		require.NoError(t, err)

		metrics, err := pgStorage.GetMetrics(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, m1, metrics["test_gauge_4"])
		assert.Equal(t, m2, metrics["test_counter_4"])
	})
}
