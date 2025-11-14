package router

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/audit"
	mocks "github.com/fragpit/yandex-go-dev-metrics/internal/mocks/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkRouter_updatesHandler(b *testing.B) {
	logger := slog.New(slog.DiscardHandler)
	auditor := audit.NewAuditor()

	count := 100
	value := 100.0
	var metrics []*model.Metrics
	for i := 0; i < count; i++ {
		name := "Test" + strconv.Itoa(i)
		metric := &model.Metrics{
			ID:    name,
			MType: string(model.GaugeType),
			Value: &value,
		}
		metrics = append(metrics, metric)
	}

	var testMetrics []model.Metric
	for _, m := range metrics {
		metric, err := model.MetricFromJSON(m)
		if err != nil {
			b.Fail()
		}

		testMetrics = append(testMetrics, metric)
	}

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	storeMock := mocks.NewMockRepository(ctrl)
	storeMock.EXPECT().
		SetOrUpdateMetricBatch(gomock.Any(), testMetrics).
		Return(nil).
		AnyTimes()

	router, err := NewRouter(logger, auditor, storeMock, nil, "")
	require.NoError(b, err)

	body, _ := json.Marshal(metrics)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/updates/",
			bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.updatesHandler(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf(
				"expected status 200, got %d",
				w.Code,
			)
		}
	}
}
