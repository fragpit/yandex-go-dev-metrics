package router

import (
	"context"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const apiShutdownTimeout = 5 * time.Second

type Router struct {
	repo   repository.Repository
	router http.Handler
	logger *slog.Logger
}

func NewRouter(l *slog.Logger, st repository.Repository) *Router {
	r := &Router{
		logger: l,
		repo:   st,
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

func (rt *Router) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      rt.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rt.logger.Error("failed to start server")
			errChan <- err
			return
		}
	}()

	rt.logger.Debug("server started", slog.String("address", srv.Addr))

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(ctx, apiShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			rt.logger.Error("failed to shutdown service gracefully")
			return err
		}

		rt.logger.Info("service shut down gracefully")
	}

	return nil
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
