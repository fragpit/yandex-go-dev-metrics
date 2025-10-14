package model

import (
	"errors"
	"fmt"
	"strconv"
)

// MetricType represents the type of a metric, either "counter" or "gauge".
type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

var (
	ErrInvalidMetricType = errors.New("invalid metric type")
	ErrMetricTypeNotSet  = errors.New("metric type is not set")
)

// Metrics is a struct used for JSON serialization/deserialization of metrics.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// Metric is an interface that defines methods for working with metrics.
type Metric interface {
	GetID() string
	GetType() MetricType
	GetValue() string
	SetValue(string) error
	ToJSON() *Metrics
}

// CounterMetric represents a counter metric.
type CounterMetric struct {
	ID    string
	Value int64
}

// GetID returns the ID of the counter metric.
func (c *CounterMetric) GetID() string {
	return c.ID
}

// GetType returns the type of the counter metric.
func (c *CounterMetric) GetType() MetricType {
	return CounterType
}

// GetValue returns the value of the counter metric as a string.
func (c *CounterMetric) GetValue() string {
	return strconv.FormatInt(c.Value, 10)
}

// SetValue sets the value of the counter metric from a string.
func (c *CounterMetric) SetValue(value string) error {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("error setting value: %w", err)
	}

	c.Value += parsedValue
	return nil
}

// ToJSON converts the counter metric to its JSON representation.
func (c *CounterMetric) ToJSON() *Metrics {
	return &Metrics{
		ID:    c.ID,
		MType: string(CounterType),
		Delta: &c.Value,
	}
}

// GaugeMetric represents a gauge metric.
type GaugeMetric struct {
	ID    string
	Value float64
}

// GetID returns the ID of the gauge metric.
func (g *GaugeMetric) GetID() string {
	return g.ID
}

// GetType returns the type of the gauge metric.
func (g *GaugeMetric) GetType() MetricType {
	return GaugeType
}

// GetValue returns the value of the gauge metric as a string.
func (g *GaugeMetric) GetValue() string {
	return strconv.FormatFloat(g.Value, 'f', -1, 64)
}

// SetValue sets the value of the gauge metric from a string.
func (g *GaugeMetric) SetValue(value string) error {
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("error setting value: %w", err)
	}

	g.Value = parsedValue
	return nil
}

// ToJSON converts the gauge metric to its JSON representation.
func (g *GaugeMetric) ToJSON() *Metrics {
	return &Metrics{
		ID:    g.ID,
		MType: string(GaugeType),
		Value: &g.Value,
	}
}

// NewMetric creates a new Metric instance based on the provided type.
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

// MetricFromJSON converts a Metrics struct to a Metric interface.
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

// ValidateType checks if the provided metric type is valid.
func ValidateType(tp string) bool {
	convType := MetricType(tp)
	return convType == CounterType || convType == GaugeType
}
