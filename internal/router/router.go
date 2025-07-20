package router

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
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

	compressForTypes := []string{
		"text/html",
		"application/json",
	}

	compressor := middleware.NewCompressor(5, compressForTypes...)

	r.Use(rt.slogMiddleware)
	r.Use(compressor.Handler)
	r.Use(rt.decompressMiddleware)

	r.Get("/", rt.rootHandler)
	r.Get("/ping", rt.pingHandler)

	r.Route("/value", func(r chi.Router) {
		r.Post("/", rt.getMetricJSON)
		r.Get("/{type}/{name}", rt.getMetric)
	})

	r.Route("/update", func(r chi.Router) {
		r.Post("/", rt.updateMetricJSON)
		r.Post("/{type}/{name}/{value}", rt.updateMetric)
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
			rt.logger.Error(
				"failed to start server",
				slog.Any("error", err),
			)
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
			rt.logger.Error(
				"failed to shutdown service gracefully",
				slog.Any("error", err),
			)
			return err
		}

		rt.logger.Info("service shut down gracefully")
	}

	return nil
}

func (rt Router) rootHandler(w http.ResponseWriter, req *http.Request) {
	metrics, err := rt.repo.GetMetrics(req.Context())
	if err != nil {
		rt.logger.Error("error retrieving metrics", slog.Any("error", err))
		http.Error(w, "error retrieving metrics", http.StatusInternalServerError)
		return
	}

	_, filename, _, _ := runtime.Caller(0)
	templatePath := filepath.Join(filepath.Dir(filename), "templates", "root.tpl")

	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		rt.logger.Error("template parse error", slog.Any("error", err))
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := tpl.Execute(w, metrics); err != nil {
		rt.logger.Error("template execute error", slog.Any("error", err))
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (rt Router) pingHandler(w http.ResponseWriter, req *http.Request) {
	if err := rt.repo.Ping(req.Context()); err != nil {
		rt.logger.Error(
			"storage ping failed",
			slog.Any("error", err),
		)
		http.Error(w, "storage ping failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt Router) getMetricJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		rt.logger.Error(
			"error reading request body",
			slog.Any("error", err),
		)
		http.Error(
			w,
			"error reading request body",
			http.StatusBadRequest,
		)
		return
	}

	var metric *model.Metrics
	if err := json.Unmarshal(body, &metric); err != nil {
		rt.logger.Error(
			"error parsing request body",
			slog.Any("error", err),
		)
		http.Error(w, "error parsing request body", http.StatusBadRequest)
		return
	}

	if metric.ID == "" {
		rt.logger.Error(
			"metric name is empty",
			slog.Any("metric", metric),
		)
		http.Error(w, "metric name is empty", http.StatusBadRequest)
		return
	}

	if !model.ValidateType(metric.MType) {
		rt.logger.Error(
			"wrong metric type",
			slog.String("type", metric.MType),
		)
		http.Error(w, "wrong metric type", http.StatusBadRequest)
		return
	}

	m, err := rt.repo.GetMetric(req.Context(), metric.ID)
	if err != nil {
		rt.logger.Error(
			"error retrieving metric",
			slog.Any("error", err),
			slog.String("metric_id", metric.ID),
		)
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	data, err := json.Marshal(m.ToJSON())
	if err != nil {
		rt.logger.Error(
			"error marshalling metric",
			slog.Any("error", err),
		)
		http.Error(w, "error marshalling metric", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		rt.logger.Error(
			"error writing response",
			slog.Any("error", err),
		)
		http.Error(w, "error writing response", http.StatusInternalServerError)
		return
	}

}

func (rt Router) updateMetricJSON(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "error setting metric", http.StatusInternalServerError)
		return
	}

	var jsonMetric *model.Metrics
	if err := json.Unmarshal(body, &jsonMetric); err != nil {
		rt.logger.Error(
			"error parsing request body",
			slog.Any("error", err),
		)
		http.Error(w, "error setting metric", http.StatusBadRequest)
		return
	}

	if jsonMetric.ID == "" {
		rt.logger.Error(
			"metric name is empty",
			slog.Any("metric", jsonMetric),
		)
		http.Error(w, "metric name is empty", http.StatusBadRequest)
		return
	}

	metric, err := model.MetricFromJSON(jsonMetric)
	if err != nil {
		rt.logger.Error(
			"error converting json to metric object",
			slog.Any("metric", jsonMetric),
			slog.Any("error", err),
		)
		http.Error(
			w,
			"error converting json to metric object",
			http.StatusBadRequest,
		)
		return
	}

	if err := rt.repo.SetOrUpdateMetric(req.Context(), metric); err != nil {
		rt.logger.Error(
			"error updating metric",
			slog.Any("error", err),
			slog.Any("metric", metric),
		)
		http.Error(w, "error setting metric", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size

	return size, err
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rt Router) updateMetric(w http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")
	metricValue := chi.URLParam(req, "value")

	metric, err := model.NewMetric(metricName, model.MetricType(metricType))
	if err != nil {
		rt.logger.Error(
			"error creating new metric",
			slog.Any("error", err),
		)
		http.Error(w, "error setting metric", http.StatusBadRequest)
		return
	}

	if err := metric.SetValue(metricValue); err != nil {
		rt.logger.Error(
			"error setting metric value",
			slog.Any("error", err),
		)
		http.Error(w, "error setting metric value", http.StatusBadRequest)
		return
	}

	if err := rt.repo.SetOrUpdateMetric(req.Context(), metric); err != nil {
		rt.logger.Error(
			"error saving metric in storage",
			slog.Any("error", err),
		)
		http.Error(
			w,
			"error saving metric in storage",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
}

func (rt Router) getMetric(w http.ResponseWriter, req *http.Request) {
	metricName := chi.URLParam(req, "name")

	metric, err := rt.repo.GetMetric(req.Context(), metricName)
	if err != nil {
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	metricValue := metric.GetValue()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(metricValue))
}
