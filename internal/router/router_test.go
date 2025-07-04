package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_setMetric(t *testing.T) {
	st := memstorage.NewMemoryStorage()

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
			name:    "valid request counter",
			request: "/update/counter/test_metric_1/1",
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
			r := NewRouter(nil, st)

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

			r.setMetric(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.value != "" {
				if value, ok := st.Metrics[mValue]; ok {
					var strVal string
					if value.MType == "gauge" {
						strVal = fmt.Sprintf("%v", *value.Value)
					}
					if value.MType == "counter" {
						strVal = fmt.Sprintf("%v", *value.Delta)
					}
					assert.Equal(t, strVal, tt.want.value)
				}
			}
		})
	}
}

func TestRouter_EmptyMetricName404(t *testing.T) {
	st := memstorage.NewMemoryStorage()
	r := NewRouter(nil, st)

	ts := httptest.NewServer(r.router)
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/update/counter",
		"text/plain",
		nil,
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRouter_getMetric(t *testing.T) {
	st := memstorage.NewMemoryStorage()

	metric := model.Metrics{
		ID:    "test_metric_1",
		MType: "counter",
	}
	err := st.SetMetric(&metric, "42")
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
			r := NewRouter(nil, st)

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
