package model

import (
	"errors"
	"strconv"
)

type MetricValue struct {
	IntValue   *int64
	FloatValue *float64
}

type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

var ErrInvalidMetricType = errors.New("invalid metric type")

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metrics) SetMetricValue(value string) error {
	if m.MType == string(CounterType) {
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		if m.Delta == nil {
			m.Delta = &parsedValue
		} else {
			*m.Delta += parsedValue
		}
	} else if m.MType == string(GaugeType) {
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		m.Value = &parsedValue
	} else {
		return ErrInvalidMetricType
	}

	return nil
}

func (m *Metrics) GetMetricValue() string {
	if m.MType == string(CounterType) {
		if m.Delta != nil {
			return strconv.FormatInt(*m.Delta, 10)
		}
		return "0"
	} else if m.MType == string(GaugeType) {
		if m.Value != nil {
			return strconv.FormatFloat(*m.Value, 'f', -1, 64)
		}
		return "0"
	}
	return ""
}

func ValidateType(tp string) bool {
	convType := MetricType(tp)
	return convType == CounterType || convType == GaugeType
}

func ValidateValue(tp, val string) bool {
	if MetricType(tp) == CounterType {
		return validateCounter(val)
	}

	if MetricType(tp) == GaugeType {
		return validateGauge(val)
	}

	return false
}

func validateCounter(val string) bool {
	_, err := strconv.ParseInt(val, 10, 64)
	return err == nil
}

func validateGauge(val string) bool {
	_, err := strconv.ParseFloat(val, 64)
	return err == nil
}
