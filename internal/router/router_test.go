package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/audit"
	mocks "github.com/fragpit/yandex-go-dev-metrics/internal/mocks/repository"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRouter_updateMetricJSON(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()

	type want struct {
		code        int
		contentType string
		value       string
	}

	tests := []struct {
		name string
		body *model.Metrics
		want want
	}{
		{
			name: "valid request counter",
			body: &model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.CounterType),
				Delta: int64Ptr(1),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "valid request gauge",
			body: &model.Metrics{
				ID:    "test_metric_2",
				MType: string(model.GaugeType),
				Value: float64Ptr(1.0),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "valid request gauge negative",
			body: &model.Metrics{
				ID:    "test_metric_3",
				MType: string(model.GaugeType),
				Value: float64Ptr(-1.0),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "gauge must rewrite #1",
			body: &model.Metrics{
				ID:    "test_metric_4",
				MType: string(model.GaugeType),
				Value: float64Ptr(100.0),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "gauge must rewrite #2",
			body: &model.Metrics{
				ID:    "test_metric_4",
				MType: string(model.GaugeType),
				Value: float64Ptr(200.0),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
				value:       "200",
			},
		},
		{
			name: "counter must increment #1",
			body: &model.Metrics{
				ID:    "test_metric_5",
				MType: string(model.CounterType),
				Delta: int64Ptr(100),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "counter must increment #2",
			body: &model.Metrics{
				ID:    "test_metric_5",
				MType: string(model.CounterType),
				Delta: int64Ptr(200),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
				value:       "300",
			},
		},
		{
			name: "empty metric name",
			body: &model.Metrics{
				ID:    "",
				MType: string(model.CounterType),
				Delta: int64Ptr(200),
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.body)
			assert.Nil(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			r, err := NewRouter(l, a, st, nil, "")
			require.NoError(t, err)

			r.updateMetricJSON(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.value != "" {
				if metric, ok := st.Metrics[tt.body.ID]; ok {
					assert.Equal(t, tt.want.value, metric.GetValue())
				}
			}
		})
	}
}

func TestRouter_getMetricJSON(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()

	var err error
	m1, err := model.NewMetric("test_metric_1", model.CounterType)
	require.NoError(t, err)
	err = m1.SetValue("42")
	require.NoError(t, err)

	m2, err := model.NewMetric("test_metric_2", model.GaugeType)
	require.NoError(t, err)
	err = m2.SetValue("3.14")
	require.NoError(t, err)

	var metricsStore []model.Metric
	metricsStore = append(metricsStore, m1, m2)

	for _, metric := range metricsStore {
		err := st.SetOrUpdateMetric(context.Background(), metric)
		require.NoError(t, err)
	}

	type want struct {
		code        int
		contentType string
		value       string
	}

	tests := []struct {
		name string
		body *model.Metrics
		want want
	}{
		{
			name: "valid request counter",
			body: &model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.CounterType),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "valid request gauge",
			body: &model.Metrics{
				ID:    "test_metric_2",
				MType: string(model.GaugeType),
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
		},
		{
			name: "valid request no type",
			body: &model.Metrics{
				ID: "test_metric_1",
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name: "invalid request empty id",
			body: &model.Metrics{
				ID:    "",
				MType: string(model.GaugeType),
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name: "invalid request non existent id",
			body: &model.Metrics{
				ID:    "xxxxxxxxxxxxxxxxxxxxxxx",
				MType: string(model.GaugeType),
			},
			want: want{
				code:        http.StatusNotFound,
				contentType: "application/json",
			},
		},
		{
			name: "invalid request wrong type",
			body: &model.Metrics{
				ID:    "test_metric_1",
				MType: "wrong_type",
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.body)
			assert.Nil(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))
			rr := httptest.NewRecorder()

			r, err := NewRouter(l, a, st, nil, "")
			require.NoError(t, err)

			r.getMetricJSON(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.value != "" {
				if metric, ok := st.Metrics[tt.body.ID]; ok {
					assert.Equal(t, tt.want.value, metric.GetValue())
				}
			}
		})
	}
}

func TestRouter_TestRoutes(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()
	r, err := NewRouter(l, a, st, nil, "")
	require.NoError(t, err)

	ts := httptest.NewServer(r.router)
	defer ts.Close()

	tests := []struct {
		name         string
		endpoint     string
		contentType  string
		code         int
		data         model.Metrics
		isCompressed bool
	}{
		{
			name:        "valid endpoint /update",
			endpoint:    "/update",
			contentType: "application/json",
			code:        http.StatusOK,
			data: model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.GaugeType),
				Value: float64Ptr(42.0),
			},
		},
		{
			name:        "valid endpoint /update/",
			endpoint:    "/update/",
			contentType: "application/json",
			code:        http.StatusOK,
			data: model.Metrics{
				ID:    "test_metric_2",
				MType: string(model.CounterType),
				Delta: int64Ptr(200),
			},
		},
		{
			name:        "valid endpoint /value",
			endpoint:    "/value",
			contentType: "application/json",
			code:        http.StatusOK,
			data: model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.GaugeType),
			},
		},
		{
			name:        "valid endpoint /value/",
			endpoint:    "/value/",
			contentType: "application/json",
			code:        http.StatusOK,
			data: model.Metrics{
				ID:    "test_metric_2",
				MType: string(model.CounterType),
			},
		},
		{
			name:        "non existent route",
			endpoint:    "/nonexistent",
			contentType: "application/json",
			code:        http.StatusNotFound,
			data: model.Metrics{
				ID:    "test_metric",
				MType: string(model.CounterType),
			},
		},
		{
			name:        "empty metric name",
			endpoint:    "/update/counter",
			contentType: "application/json",
			code:        http.StatusNotFound,
		},
		{
			name:         "valid endpoint /update compressed",
			endpoint:     "/update",
			contentType:  "application/json",
			code:         http.StatusOK,
			isCompressed: true,
			data: model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.GaugeType),
				Value: float64Ptr(42.0),
			},
		},
		{
			name:         "valid endpoint /value compressed",
			endpoint:     "/value",
			contentType:  "application/json",
			code:         http.StatusOK,
			isCompressed: true,
			data: model.Metrics{
				ID:    "test_metric_1",
				MType: string(model.GaugeType),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}

			data, err := json.Marshal(tt.data)
			require.NoError(t, err)

			var body io.Reader
			if tt.isCompressed {
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				_, err = zw.Write(data)
				require.NoError(t, err)
				err = zw.Close()
				require.NoError(t, err)
				body = bytes.NewReader(buf.Bytes())
			} else {
				body = bytes.NewReader(data)
			}

			req, err := http.NewRequest(
				http.MethodPost,
				ts.URL+tt.endpoint,
				body,
			)
			require.NoError(t, err)

			req.Header.Set("Content-Type", tt.contentType)
			if tt.isCompressed {
				req.Header.Set("Content-Encoding", "gzip")
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.code, resp.StatusCode)
		})
	}
}

func TestRouter_updateMetric(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()

	type want struct {
		code        int
		contentType string
		value       string
	}

	tests := []struct {
		name    string
		request string
		want    want
	}{
		{
			name:    "valid request counter #1",
			request: "/update/counter/test_metric_1/1",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "valid request counter #2",
			request: "/update/counter/testSetGet111/606",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "valid request gauge",
			request: "/update/gauge/test_metric_2/1",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "overwrite with different type",
			request: "/update/counter/test_metric_2/1",
			want: want{
				code:        http.StatusInternalServerError,
				contentType: "text/plain",
			},
		},
		{
			name:    "valid request gauge negative",
			request: "/update/gauge/test_metric_3/-1",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "incorrect metric type",
			request: "/update/incorrect/test_metric/1",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
			},
		},
		{
			name:    "incorrect value",
			request: "/update/counter/test_metric/aaa",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
			},
		},
		{
			name:    "gauge must rewrite #1",
			request: "/update/gauge/test_metric_4/100",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "gauge must rewrite #2",
			request: "/update/gauge/test_metric_4/200",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
				value:       "200",
			},
		},
		{
			name:    "counter must increment #1",
			request: "/update/counter/test_metric_5/100",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:    "counter must increment #2",
			request: "/update/counter/test_metric_5/200",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
				value:       "300",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				tt.request,
				nil,
			)
			rr := httptest.NewRecorder()
			r, err := NewRouter(l, a, st, nil, "")
			require.NoError(t, err)

			chiCtx := chi.NewRouteContext()
			req = req.WithContext(
				context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx),
			)

			sr := strings.Split(tt.request, "/")

			var mType string
			if len(sr) > 2 {
				mType = sr[2]
			}

			var mName string
			if len(sr) > 3 {
				mName = sr[3]
			}

			var mValue string
			if len(sr) > 4 {
				mValue = sr[4]
			}

			chiCtx.URLParams.Add("type", fmt.Sprintf("%v", mType))
			chiCtx.URLParams.Add("name", fmt.Sprintf("%v", mName))
			chiCtx.URLParams.Add("value", fmt.Sprintf("%v", mValue))

			r.updateMetric(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.value != "" {
				if metric, ok := st.Metrics[mValue]; ok {
					assert.Equal(t, tt.want.value, metric.GetValue())
				}
			}
		})
	}
}

func TestRouter_getMetric(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()

	var err error
	metric, err := model.NewMetric("test_metric_1", model.CounterType)
	require.NoError(t, err)
	err = metric.SetValue("42")
	require.NoError(t, err)

	err = st.SetOrUpdateMetric(context.Background(), metric)
	require.NoError(t, err)

	type want struct {
		code  int
		value string
	}

	tests := []struct {
		name    string
		request string
		want    want
	}{
		{
			name:    "valid get request",
			request: "/value/counter/test_metric_1",
			want: want{
				code:  http.StatusOK,
				value: "42",
			},
		},
		{
			name:    "valid get request",
			request: "/value/counter/test_metric_2",
			want: want{
				code:  http.StatusNotFound,
				value: "metric not found\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				tt.request,
				nil,
			)

			rr := httptest.NewRecorder()

			r, err := NewRouter(l, a, st, nil, "")
			require.NoError(t, err)

			chiCtx := chi.NewRouteContext()
			req = req.WithContext(
				context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx),
			)

			sr := strings.Split(tt.request, "/")

			var mName string
			if len(sr) > 3 {
				mName = sr[3]
			}
			chiCtx.URLParams.Add("name", fmt.Sprintf("%v", mName))

			r.getMetric(rr, req)
			res := rr.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.value, string(body))
		})
	}
}

func TestRouter_pingHandler(t *testing.T) {
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()
	ctx := context.Background()

	type want struct {
		code int
	}

	tests := []struct {
		name string
		err  error
		want want
	}{
		{
			name: "valid request ping",
			err:  nil,
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name: "invalid request ping",
			err:  errors.New("some error"),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			storeMock := mocks.NewMockRepository(ctrl)
			storeMock.EXPECT().
				Ping(ctx).
				Times(1).
				Return(tt.err)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			rr := httptest.NewRecorder()

			router, err := NewRouter(l, a, storeMock, nil, "")
			require.NoError(t, err)

			router.pingHandler(rr, req)

			assert.Equal(t, tt.want.code, rr.Code)
		})
	}

}

func TestRouter_updatesHandler(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	l := slog.New(slog.DiscardHandler)
	a := audit.NewAuditor()
	r, err := NewRouter(l, a, st, nil, "")
	require.NoError(t, err)

	type want struct {
		code int
	}

	tests := []struct {
		name      string
		body      []*model.Metrics
		want      want
		expectErr bool
	}{
		{
			name: "valid batch update gauge and counter",
			body: []*model.Metrics{
				{
					ID:    "batch_gauge_1",
					MType: string(model.GaugeType),
					Value: float64Ptr(10.5),
				},
				{
					ID:    "batch_counter_1",
					MType: string(model.CounterType),
					Delta: int64Ptr(5),
				},
			},
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name: "empty metric name in batch",
			body: []*model.Metrics{
				{
					ID:    "",
					MType: string(model.GaugeType),
					Value: float64Ptr(1.0),
				},
			},
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "invalid metric type in batch",
			body: []*model.Metrics{
				{
					ID:    "invalid_type_metric",
					MType: "invalid_type",
					Value: float64Ptr(1.0),
				},
			},
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "invalid json body",
			body: nil,
			want: want{
				code: http.StatusBadRequest,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.expectErr {
				req = httptest.NewRequest(
					http.MethodPost,
					"/updates/",
					bytes.NewBuffer([]byte("{invalid json")),
				)
			} else {
				data, err := json.Marshal(tt.body)
				require.NoError(t, err)
				req = httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(data))
			}
			rr := httptest.NewRecorder()
			r.updatesHandler(rr, req)
			res := rr.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.code == http.StatusOK && tt.body != nil {
				for _, m := range tt.body {
					if m.ID != "" && model.ValidateType(m.MType) {
						metric, ok := st.Metrics[m.ID]
						assert.True(t, ok)
						if m.MType == string(model.GaugeType) && m.Value != nil {
							assert.Equal(t, fmt.Sprintf("%v", *m.Value), metric.GetValue())
						}
						if m.MType == string(model.CounterType) && m.Delta != nil {
							assert.Equal(t, fmt.Sprintf("%v", *m.Delta), metric.GetValue())
						}
					}
				}
			}
		})
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestRouter_rootHandler(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)
	repo := memstorage.NewMemoryStorage()
	auditor := audit.NewAuditor()
	router, err := NewRouter(logger, auditor, repo, nil, "")
	require.NoError(t, err)

	m1, _ := model.NewMetric("test_gauge", model.GaugeType)
	_ = m1.SetValue("42.5")
	_ = repo.SetOrUpdateMetric(context.Background(), m1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "test_gauge")
}
