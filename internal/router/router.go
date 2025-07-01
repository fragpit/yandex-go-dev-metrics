package router

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Router struct {
	repo   repository.Repository
	router http.Handler
}

func NewRouter(st repository.Repository) *Router {
	r := &Router{
		repo: st,
	}
	r.router = r.initRoutes()
	return r
}

func (rt *Router) initRoutes() http.Handler {
	r := chi.NewMux()
	r.Use(middleware.Logger)

	r.Get("/", rt.rootHandler)

	r.Route("/value", func(r chi.Router) {
		r.Route("/{type}", func(r chi.Router) {
			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", rt.getMetric)
			})
		})
	})

	r.Route("/update", func(r chi.Router) {
		r.Route("/{type}", func(r chi.Router) {
			r.Route("/{name}", func(r chi.Router) {
				r.Post("/{value}", rt.setMetric)
			})
		})
	})
	return r
}

func (rt *Router) Run(addr string) error {
	return http.ListenAndServe(addr, rt.router)
}

func (rt Router) rootHandler(resp http.ResponseWriter, req *http.Request) {
	metrics, err := rt.repo.GetMetrics()
	if err != nil {
		http.Error(resp, "error retrieving metrics", http.StatusInternalServerError)
		return
	}

	_, filename, _, _ := runtime.Caller(0)
	templatePath := filepath.Join(filepath.Dir(filename), "templates", "root.tpl")

	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(resp, "template error", http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/html")
	resp.WriteHeader(http.StatusOK)
	if err := tpl.Execute(resp, metrics); err != nil {
		log.Printf("template execute error: %v", err)
		http.Error(resp, "template error", http.StatusInternalServerError)
	}
}

func (rt Router) getMetric(resp http.ResponseWriter, req *http.Request) {
	metricName := chi.URLParam(req, "name")

	metric, err := rt.repo.GetMetric(metricName)
	if err != nil {
		http.Error(resp, "metric not found", http.StatusNotFound)
		return
	}

	metricValue := metric.GetMetricValue()

	resp.WriteHeader(http.StatusOK)
	resp.Header().Set("Content-Type", "text/plain")
	resp.Write([]byte(metricValue))
}

func (rt Router) setMetric(resp http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")
	metricValue := chi.URLParam(req, "value")

	if !model.ValidateType(metricType) {
		http.Error(resp, "incorrect metric type", http.StatusBadRequest)
		return
	}

	if !model.ValidateValue(metricType, metricValue) {
		http.Error(resp, "incorrect metric value", http.StatusBadRequest)
		return
	}

	metric := &model.Metrics{
		ID:    metricName,
		MType: metricType,
	}

	if err := rt.repo.SetMetric(metric, metricValue); err != nil {
		http.Error(resp, "error setting metric", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Header().Set("Content-Type", "text/plain")
}
