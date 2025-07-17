package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

const (
	clientPostTimeout = 5 * time.Second
)

type Metrics struct {
	logger  *slog.Logger
	counter int64
	Metrics map[string]*model.Metrics
}

func NewMetrics(l *slog.Logger) *Metrics {
	return &Metrics{
		logger:  l,
		Metrics: make(map[string]*model.Metrics),
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

func (m *Metrics) reportMetrics(serverURL string) {
	updateURL := serverURL + "/update"

	client := &http.Client{
		Timeout: clientPostTimeout,
	}

	for _, metric := range m.Metrics {
		data, err := json.Marshal(metric)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		defer zw.Close()

		_, err = zw.Write(data)
		if err != nil {
			m.logger.Error(
				"error writing compressed data",
				slog.Any("error", err),
			)
			return
		}

		err = zw.Close()
		if err != nil {
			m.logger.Error(
				"error closing gzip writer",
				slog.Any("error", err),
			)
			return
		}

		req, err := http.NewRequest(
			http.MethodPost,
			updateURL,
			&buf,
		)
		if err != nil {
			m.logger.Error(
				"error creating request",
				slog.Any("error", err),
			)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			m.logger.Error(
				"error reporting metrics",
				slog.Any("error", err),
			)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			m.logger.Error(
				"non-OK status code",
				slog.Int("status_code", resp.StatusCode),
			)
			return
		}
	}

	m.reset()
}

func (m *Metrics) register(tp model.MetricType, name, value string) error {
	metric := &model.Metrics{
		ID:    name,
		MType: string(tp),
	}

	if err := metric.SetValue(value); err != nil {
		return err
	}

	m.Metrics[metric.ID] = metric
	return nil
}

func (m *Metrics) reset() {
	clear(m.Metrics)
	m.counter = 0
}
