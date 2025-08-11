package model

import (
	"errors"
	"fmt"
	"strconv"
)

type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

var (
	ErrInvalidMetricType = errors.New("invalid metric type")
	ErrMetricTypeNotSet  = errors.New("metric type is not set")
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type Metric interface {
	GetID() string
	GetType() MetricType
	GetValue() string
	SetValue(string) error
	ToJSON() *Metrics
}

type CounterMetric struct {
	ID    string
	Value int64
}

func (c *CounterMetric) GetID() string {
	return c.ID
}

func (c *CounterMetric) GetType() MetricType {
	return CounterType
}

func (c *CounterMetric) GetValue() string {
	return strconv.FormatInt(c.Value, 10)
}

func (c *CounterMetric) SetValue(value string) error {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("error setting value: %w", err)
	}

	c.Value += parsedValue
	return nil
}

func (c *CounterMetric) ToJSON() *Metrics {
	return &Metrics{
		ID:    c.ID,
		MType: string(CounterType),
		Delta: &c.Value,
	}
}

type GaugeMetric struct {
	ID    string
	Value float64
}

func (g *GaugeMetric) GetID() string {
	return g.ID
}

func (g *GaugeMetric) GetType() MetricType {
	return GaugeType
}

func (g *GaugeMetric) GetValue() string {
	return strconv.FormatFloat(g.Value, 'f', -1, 64)
}

func (g *GaugeMetric) SetValue(value string) error {
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("error setting value: %w", err)
	}

	g.Value = parsedValue
	return nil
}

func (g *GaugeMetric) ToJSON() *Metrics {
	return &Metrics{
		ID:    g.ID,
		MType: string(GaugeType),
		Value: &g.Value,
	}
}

func NewMetric(id string, metricType MetricType) (Metric, error) {
	switch MetricType(metricType) {
	case CounterType:
		return &CounterMetric{ID: id}, nil
	case GaugeType:
		return &GaugeMetric{ID: id}, nil
	default:
		return nil, ErrInvalidMetricType
	}
}

func MetricFromJSON(m *Metrics) (Metric, error) {
	switch MetricType(m.MType) {
	case CounterType:
		counter := &CounterMetric{ID: m.ID}
		if m.Delta != nil {
			counter.Value = *m.Delta
		}
		return counter, nil
	case GaugeType:
		gauge := &GaugeMetric{ID: m.ID}
		if m.Value != nil {
			gauge.Value = *m.Value
		}
		return gauge, nil
	default:
		return nil, ErrInvalidMetricType
	}
}

func ValidateType(tp string) bool {
	convType := MetricType(tp)
	return convType == CounterType || convType == GaugeType
}
