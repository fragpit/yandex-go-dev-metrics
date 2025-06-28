package model

import "strconv"

type MetricValue struct {
	IntValue   *int64
	FloatValue *float64
}

type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

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
