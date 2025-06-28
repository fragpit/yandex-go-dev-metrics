package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

type Router struct {
	repo *memstorage.MemoryStorage
}

func New(st *memstorage.MemoryStorage) *Router {
	return &Router{
		repo: st,
	}
}

func (r Router) Run() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /", r.MetricsHandler)

	http.ListenAndServe("localhost:8080", mux)
}

func (r Router) MetricsHandler(resp http.ResponseWriter, req *http.Request) {
	urlPath := req.URL.Path
	pathSegments := []string{}
	for _, segment := range strings.Split(urlPath, "/") {
		if segment != "" {
			pathSegments = append(pathSegments, segment)
		}
	}

	if len(pathSegments) == 0 || pathSegments[0] != "update" {
		http.Error(resp, "unknown action", http.StatusBadRequest)
		return
	}

	if len(pathSegments) < 2 {
		http.Error(resp, "metric type missing", http.StatusBadRequest)
		return
	}

	metricType := pathSegments[1]

	if !model.ValidateType(metricType) {
		http.Error(resp, "incorrect metric type", http.StatusBadRequest)
		return
	}

	if len(pathSegments) < 3 {
		http.Error(resp, "metric name not found", http.StatusNotFound)
		return
	}

	metricName := pathSegments[2]

	if len(pathSegments) < 4 {
		http.Error(resp, "metric value not set", http.StatusBadRequest)
		return
	}

	metricValue := pathSegments[3]

	if !model.ValidateValue(metricType, metricValue) {
		http.Error(resp, "incorrect metric value", http.StatusBadRequest)
		return
	}

	metric := &model.Metrics{
		ID:    metricName,
		MType: metricType,
	}

	if err := r.repo.SetMetric(metric, metricValue); err != nil {
		http.Error(resp, "error setting metric", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Header().Set("Content-Type", "text/plain")

	metrics := r.repo.Metrics
	if len(metrics) == 0 {
		resp.Write([]byte("No metrics available"))
		return
	}

	for name, metric := range metrics {
		var deltaStr, valueStr string

		if metric.Delta != nil {
			deltaStr = fmt.Sprintf("%d", *metric.Delta)
		} else {
			deltaStr = "nil"
		}

		if metric.Value != nil {
			valueStr = fmt.Sprintf("%.6f", *metric.Value)
		} else {
			valueStr = "nil"
		}

		resp.Write(
			[]byte(fmt.Sprintf("%s: %s or %s\n", name, deltaStr, valueStr)),
		)
	}
}
