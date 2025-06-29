package router

import (
	"net/http"
	"strings"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

type Router struct {
	repo repository.Repository
}

func NewRouter(st repository.Repository) *Router {
	return &Router{
		repo: st,
	}
}

func (r Router) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /update/", r.MetricsHandler)

	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		return err
	}

	return nil
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
}
