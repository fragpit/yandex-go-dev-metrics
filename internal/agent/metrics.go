package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

const (
	clientPostTimeout = 5 * time.Second
)

type metric struct {
	Type  model.MetricType
	Value string
}

type Metrics struct {
	counter int64
	Metrics map[string]metric
}

func NewMetrics() *Metrics {
	return &Metrics{
		Metrics: make(map[string]metric),
	}
}

func (m *Metrics) pollMetrics() error {
	log.Println("Polling metrics at", time.Now().Format("15:04:05"))

	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	m.register(model.GaugeType, "mstat_alloc", fmt.Sprintf("%d", mstat.Alloc))
	m.register(
		model.GaugeType,
		"mstat_buckhashsys",
		fmt.Sprintf("%d", mstat.BuckHashSys),
	)
	m.register(model.GaugeType, "mstat_frees", fmt.Sprintf("%d", mstat.Frees))
	m.register(
		model.GaugeType,
		"mstat_gccpufraction",
		fmt.Sprintf("%f", mstat.GCCPUFraction),
	)
	m.register(model.GaugeType, "mstat_gcsys", fmt.Sprintf("%d", mstat.GCSys))
	m.register(
		model.GaugeType,
		"mstat_heapalloc",
		fmt.Sprintf("%d", mstat.HeapAlloc),
	)
	m.register(
		model.GaugeType,
		"mstat_heapidle",
		fmt.Sprintf("%d", mstat.HeapIdle),
	)
	m.register(
		model.GaugeType,
		"mstat_heapinuse",
		fmt.Sprintf("%d", mstat.HeapInuse),
	)
	m.register(
		model.GaugeType,
		"mstat_heapobjects",
		fmt.Sprintf("%d", mstat.HeapObjects),
	)
	m.register(
		model.GaugeType,
		"mstat_heapreleased",
		fmt.Sprintf("%d", mstat.HeapReleased),
	)
	m.register(model.GaugeType, "mstat_heapsys", fmt.Sprintf("%d", mstat.HeapSys))
	m.register(model.GaugeType, "mstat_lastgc", fmt.Sprintf("%d", mstat.LastGC))
	m.register(model.GaugeType, "mstat_lookups", fmt.Sprintf("%d", mstat.Lookups))
	m.register(
		model.GaugeType,
		"mstat_mcacheinuse",
		fmt.Sprintf("%d", mstat.MCacheInuse),
	)
	m.register(
		model.GaugeType,
		"mstat_mcachesys",
		fmt.Sprintf("%d", mstat.MCacheSys),
	)
	m.register(
		model.GaugeType,
		"mstat_mspaninuse",
		fmt.Sprintf("%d", mstat.MSpanInuse),
	)
	m.register(
		model.GaugeType,
		"mstat_mspansys",
		fmt.Sprintf("%d", mstat.MSpanSys),
	)
	m.register(model.GaugeType, "mstat_mallocs", fmt.Sprintf("%d", mstat.Mallocs))
	m.register(model.GaugeType, "mstat_nextgc", fmt.Sprintf("%d", mstat.NextGC))
	m.register(
		model.GaugeType,
		"mstat_numforcedgc",
		fmt.Sprintf("%d", mstat.NumForcedGC),
	)
	m.register(model.GaugeType, "mstat_numgc", fmt.Sprintf("%d", mstat.NumGC))
	m.register(
		model.GaugeType,
		"mstat_othersys",
		fmt.Sprintf("%d", mstat.OtherSys),
	)
	m.register(
		model.GaugeType,
		"mstat_pausetotalns",
		fmt.Sprintf("%d", mstat.PauseTotalNs),
	)
	m.register(
		model.GaugeType,
		"mstat_stackinuse",
		fmt.Sprintf("%d", mstat.StackInuse),
	)
	m.register(
		model.GaugeType,
		"mstat_stacksys",
		fmt.Sprintf("%d", mstat.StackSys),
	)
	m.register(model.GaugeType, "mstat_sys", fmt.Sprintf("%d", mstat.Sys))
	m.register(
		model.GaugeType,
		"mstat_totalalloc",
		fmt.Sprintf("%d", mstat.TotalAlloc),
	)

	rvalue := rand.IntN(100)
	m.register(model.GaugeType, "random_value", fmt.Sprintf("%d", rvalue))

	m.counter++
	m.register(model.CounterType, "poll_count", fmt.Sprintf("%d", m.counter))

	return nil
}

func (m *Metrics) reportMetrics(serverURL string) {
	log.Println("Reporting metrics at", time.Now().Format("15:04:05"))

	client := &http.Client{
		Timeout: clientPostTimeout,
	}

	for name, metric := range m.Metrics {
		metricURL := fmt.Sprintf(
			"%s/update/%s/%s/%s",
			serverURL,
			metric.Type,
			name,
			metric.Value,
		)

		resp, err := client.Post(metricURL, "text/plain", nil)
		if err != nil {
			log.Printf("error reporting metrics %s: %v", name, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("error reporting metrics %s: %v", name, err)
			return
		}
	}

	m.counter = 0
}

func (m *Metrics) register(tp model.MetricType, name, value string) {
	metric := &metric{
		Type:  tp,
		Value: value,
	}

	m.Metrics[name] = *metric
}
