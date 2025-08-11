package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

const (
	clientPostTimeout = 5 * time.Second
)

type Metrics struct {
	logger  *slog.Logger
	counter int64
	cfg     *config.AgentConfig
	Metrics map[string]model.Metric
}

func NewMetrics(l *slog.Logger, cfg *config.AgentConfig) *Metrics {
	return &Metrics{
		logger:  l,
		cfg:     cfg,
		Metrics: make(map[string]model.Metric),
	}
}

func (m *Metrics) pollMetrics() error {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	reg := func(tp model.MetricType, name, value string) {
		if err := m.register(tp, name, value); err != nil {
			m.logger.Error("failed to register metric",
				slog.String("name", name),
				slog.Any("error", err))
		}
	}

	reg(model.GaugeType, "Alloc", fmt.Sprintf("%d", mstat.Alloc))
	reg(
		model.GaugeType,
		"BuckHashSys",
		fmt.Sprintf("%d", mstat.BuckHashSys),
	)
	reg(model.GaugeType, "Frees", fmt.Sprintf("%d", mstat.Frees))
	reg(
		model.GaugeType,
		"GCCPUFraction",
		fmt.Sprintf("%f", mstat.GCCPUFraction),
	)
	reg(model.GaugeType, "GCSys", fmt.Sprintf("%d", mstat.GCSys))
	reg(
		model.GaugeType,
		"HeapAlloc",
		fmt.Sprintf("%d", mstat.HeapAlloc),
	)
	reg(
		model.GaugeType,
		"HeapIdle",
		fmt.Sprintf("%d", mstat.HeapIdle),
	)
	reg(
		model.GaugeType,
		"HeapInuse",
		fmt.Sprintf("%d", mstat.HeapInuse),
	)
	reg(
		model.GaugeType,
		"HeapObjects",
		fmt.Sprintf("%d", mstat.HeapObjects),
	)
	reg(
		model.GaugeType,
		"HeapReleased",
		fmt.Sprintf("%d", mstat.HeapReleased),
	)
	reg(model.GaugeType, "HeapSys", fmt.Sprintf("%d", mstat.HeapSys))
	reg(model.GaugeType, "LastGC", fmt.Sprintf("%d", mstat.LastGC))
	reg(model.GaugeType, "Lookups", fmt.Sprintf("%d", mstat.Lookups))
	reg(
		model.GaugeType,
		"MCacheInuse",
		fmt.Sprintf("%d", mstat.MCacheInuse),
	)
	reg(
		model.GaugeType,
		"MCacheSys",
		fmt.Sprintf("%d", mstat.MCacheSys),
	)
	reg(
		model.GaugeType,
		"MSpanInuse",
		fmt.Sprintf("%d", mstat.MSpanInuse),
	)
	reg(
		model.GaugeType,
		"MSpanSys",
		fmt.Sprintf("%d", mstat.MSpanSys),
	)
	reg(model.GaugeType, "Mallocs", fmt.Sprintf("%d", mstat.Mallocs))
	reg(model.GaugeType, "NextGC", fmt.Sprintf("%d", mstat.NextGC))
	reg(
		model.GaugeType,
		"NumForcedGC",
		fmt.Sprintf("%d", mstat.NumForcedGC),
	)
	reg(model.GaugeType, "NumGC", fmt.Sprintf("%d", mstat.NumGC))
	reg(
		model.GaugeType,
		"OtherSys",
		fmt.Sprintf("%d", mstat.OtherSys),
	)
	reg(
		model.GaugeType,
		"PauseTotalNs",
		fmt.Sprintf("%d", mstat.PauseTotalNs),
	)
	reg(
		model.GaugeType,
		"StackInuse",
		fmt.Sprintf("%d", mstat.StackInuse),
	)
	reg(
		model.GaugeType,
		"StackSys",
		fmt.Sprintf("%d", mstat.StackSys),
	)
	reg(model.GaugeType, "Sys", fmt.Sprintf("%d", mstat.Sys))
	reg(
		model.GaugeType,
		"TotalAlloc",
		fmt.Sprintf("%d", mstat.TotalAlloc),
	)

	rvalue := rand.IntN(100)
	reg(model.GaugeType, "RandomValue", fmt.Sprintf("%d", rvalue))

	m.counter++
	reg(model.CounterType, "PollCount", fmt.Sprintf("%d", m.counter))

	return nil
}

func (m *Metrics) reportMetrics() {
	updateURL := m.cfg.ServerURL + "/updates/"

	client := resty.New()
	client.
		SetTimeout(clientPostTimeout).
		SetRetryCount(3).
		SetRetryWaitTime(1*time.Second).
		SetRetryMaxWaitTime(5*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		OnBeforeRequest(gzipRequestMiddleware())

	if len(m.cfg.SecretKey) > 0 {
		client.OnBeforeRequest(checksumRequestMiddleware(m.cfg.SecretKey))
	}

	var (
		data    []byte
		err     error
		metrics []*model.Metrics
	)

	for _, metric := range m.Metrics {
		m := metric.ToJSON()
		metrics = append(metrics, m)
	}

	data, err = json.Marshal(metrics)
	if err != nil {
		m.logger.Error(
			"error marshaling metrics",
			slog.Any("error", err),
		)
		return
	}

	resp, err := client.R().
		SetBody(data).
		Post(updateURL)
	if err != nil {
		m.logger.Error(
			"error reporting metrics",
			slog.Any("error", err),
		)
		return
	}

	if resp.StatusCode() != http.StatusOK {
		m.logger.Error(
			"non-OK status code",
			slog.Int("status_code", resp.StatusCode()),
		)
		return
	}

	m.reset()
}

func (m *Metrics) register(tp model.MetricType, name, value string) error {
	metric, err := model.NewMetric(name, tp)
	if err != nil {
		return err
	}

	if err := metric.SetValue(value); err != nil {
		return err
	}

	m.Metrics[metric.GetID()] = metric
	return nil
}

func (m *Metrics) reset() {
	clear(m.Metrics)
	m.counter = 0
}

func gzipRequestMiddleware() resty.RequestMiddleware {
	return func(c *resty.Client, req *resty.Request) error {
		if req.Body == nil {
			return nil
		}

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		defer zw.Close()

		bodyBytes, ok := req.Body.([]byte)
		if !ok {
			data, err := json.Marshal(req.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal body %w", err)
			}
			bodyBytes = data
		}

		if _, err := zw.Write(bodyBytes); err != nil {
			return err
		}

		if err := zw.Close(); err != nil {
			return err
		}

		req.SetBody(buf.Bytes())
		req.SetHeader("Content-Encoding", "gzip")

		return nil
	}
}

func checksumRequestMiddleware(key []byte) resty.RequestMiddleware {
	return func(c *resty.Client, req *resty.Request) error {
		mac := hmac.New(sha256.New, key)

		bodyBytes, ok := req.Body.([]byte)
		if !ok {
			data, err := json.Marshal(req.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal body %w", err)
			}
			bodyBytes = data
		}

		mac.Write(bodyBytes)
		sum := mac.Sum(nil)
		sumEncoded := base64.RawStdEncoding.EncodeToString(sum)

		req.SetHeader("HashSHA256", sumEncoded)

		return nil
	}
}
