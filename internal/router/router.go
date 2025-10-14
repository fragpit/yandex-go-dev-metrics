package router

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/audit"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// apiShutdownTimeout defines the timeout for graceful shutdown of the API server.
const apiShutdownTimeout = 5 * time.Second

// Router handles HTTP requests and routes them to appropriate handlers.
type Router struct {
	repo      repository.Repository
	router    http.Handler
	logger    *slog.Logger
	auditor   *audit.Auditor
	secretKey []byte
}

// NewRouter creates a new Router instance.
func NewRouter(
	l *slog.Logger,
	a *audit.Auditor,
	st repository.Repository,
	key []byte,
) *Router {
	r := &Router{
		logger:    l,
		auditor:   a,
		repo:      st,
		secretKey: key,
	}
	r.router = r.initRoutes()
	return r
}

// initRoutes initializes the HTTP routes and middleware.
func (rt *Router) initRoutes() http.Handler {
	r := chi.NewMux()

	compressForTypes := []string{
		"text/html",
		"application/json",
	}

	compressor := middleware.NewCompressor(5, compressForTypes...)

	r.Use(rt.slogMiddleware)
	r.Use(compressor.Handler)

	r.Get("/", rt.rootHandler)
	r.Get("/ping", rt.pingHandler)

	r.Route("/value", func(r chi.Router) {
		r.Use(rt.decompressMiddleware)
		r.Post("/", rt.getMetricJSON)
		r.Get("/{type}/{name}", rt.getMetric)
	})

	r.Route("/update", func(r chi.Router) {
		r.Use(rt.decompressMiddleware)
		r.Post("/", rt.updateMetricJSON)
		r.Post("/{type}/{name}/{value}", rt.updateMetric)
	})

	r.Route("/updates", func(r chi.Router) {
		if len(rt.secretKey) > 0 {
			r.Use(rt.checksumMiddleware)
		}
		r.Use(rt.decompressMiddleware)
		r.Post("/", rt.updatesHandler)
	})

	return r
}

// Run starts the HTTP server and listens for incoming requests.
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

// rootHandler serves the root HTML page displaying all metrics.
func (rt Router) rootHandler(w http.ResponseWriter, req *http.Request) {
	metrics, err := rt.repo.GetMetrics(req.Context())
	if err != nil {
		rt.logger.Error("error retrieving metrics", slog.Any("error", err))
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	_, filename, _, _ := runtime.Caller(0)
	templatePath := filepath.Join(filepath.Dir(filename), "templates", "root.tpl")

	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		rt.logger.Error("template parse error", slog.Any("error", err))
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := tpl.Execute(w, metrics); err != nil {
		rt.logger.Error("template execute error", slog.Any("error", err))
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

// pingHandler checks the health of the storage by performing a ping operation.
func (rt Router) pingHandler(w http.ResponseWriter, req *http.Request) {
	if err := rt.repo.Ping(req.Context()); err != nil {
		rt.logger.Error(
			"storage ping failed",
			slog.Any("error", err),
		)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// getMetricJSON handles retrieval of a single metric by JSON payload.
func (rt Router) getMetricJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metric *model.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
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
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		rt.logger.Error(
			"error writing response",
			slog.Any("error", err),
		)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

}

func (rt Router) updateMetricJSON(w http.ResponseWriter, req *http.Request) {
	var jsonMetric *model.Metrics
	if err := json.NewDecoder(req.Body).Decode(&jsonMetric); err != nil {
		rt.logger.Error(
			"error decoding request body",
			slog.Any("error", err),
		)
		http.Error(w, "error decoding request body", http.StatusBadRequest)
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
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	metricTypes := []string{metric.GetID()}
	go rt.runAudit(metricTypes, req.RemoteAddr)

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

// updateMetric handles updating or creating a single metric by type, name,
// and value.
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
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	metricTypes := []string{metric.GetID()}
	go rt.runAudit(metricTypes, req.RemoteAddr)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
}

// getMetric handles retrieval of a single metric by type and name.
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

// updatesHandler handles batch updates of metrics via JSON payload.
func (rt Router) updatesHandler(w http.ResponseWriter, req *http.Request) {
	var jsonMetrics []*model.Metrics
	if err := json.NewDecoder(req.Body).Decode(&jsonMetrics); err != nil {
		rt.logger.Error(
			"error decoding request body",
			slog.Any("error", err),
		)
		http.Error(w, "error decoding request body", http.StatusBadRequest)
		return
	}

	var metrics []model.Metric
	metricTypesMap := make(map[string]struct{})
	for _, m := range jsonMetrics {
		if m.ID == "" {
			rt.logger.Error(
				"metric name is empty",
				slog.Any("metric", m),
			)
			http.Error(w, "metric name is empty", http.StatusBadRequest)
			return
		}

		metric, err := model.MetricFromJSON(m)
		if err != nil {
			rt.logger.Error(
				"error converting metrics from json",
				slog.Any("metric", m),
			)
			http.Error(w, "error converting metrics from json", http.StatusBadRequest)
			return
		}

		metrics = append(metrics, metric)
		metricTypesMap[m.ID] = struct{}{}
	}

	if err := rt.repo.SetOrUpdateMetricBatch(req.Context(), metrics); err != nil {
		rt.logger.Error(
			"error batch updating metrics",
			slog.Any("error", err),
		)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	metricTypes := make([]string, 0, len(metricTypesMap))
	for id := range metricTypesMap {
		metricTypes = append(metricTypes, id)
	}

	go rt.runAudit(metricTypes, req.RemoteAddr)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
}

func (rt *Router) runAudit(metricTypes []string, ipPort string) {
	ctx, cancel := context.WithTimeout(context.Background(), audit.DefaultTimeout)
	defer cancel()

	clientIP := getClientIP(ipPort)

	if err := rt.auditor.LogEvent(
		ctx,
		metricTypes,
		clientIP,
	); err != nil {
		slog.Error("failed to log audit event", slog.Any("error", err))
	}
}

func getClientIP(ipPort string) string {
	ip, _, err := net.SplitHostPort(ipPort)
	if err != nil {
		return ipPort
	}
	return ip
}
