package agent

import (
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMetrics_register(t *testing.T) {
	type fields struct {
		counter int64
		Metrics map[string]model.Metric
	}

	type args struct {
		tp    model.MetricType
		name  string
		value string
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
	}{
		{
			name: "positive test #1",
			fields: fields{
				counter: 0,
				Metrics: map[string]model.Metric{},
			},
			args: args{
				tp:    model.CounterType,
				name:  "test_name",
				value: "1",
			},
			expectedErr: nil,
		},
		{
			name: "metric type not set",
			fields: fields{
				counter: 0,
				Metrics: map[string]model.Metric{},
			},
			args: args{
				tp:    "",
				name:  "test_name",
				value: "1",
			},
			expectedErr: model.ErrInvalidMetricType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				counter: tt.fields.counter,
				Metrics: tt.fields.Metrics,
			}
			err := m.register(tt.args.tp, tt.args.name, tt.args.value)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
