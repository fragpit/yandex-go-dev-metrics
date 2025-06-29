package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/stretchr/testify/assert"
)

func TestRouter_MetricsHandler(t *testing.T) {
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
			name:    "no metric name status",
			request: "/update/counter",
			want: want{
				code:        http.StatusNotFound,
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
			r := NewRouter(st)

			r.MetricsHandler(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.value != "" {
				sr := strings.Split(tt.request, "/")
				if value, ok := st.Metrics[sr[3]]; ok {
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
