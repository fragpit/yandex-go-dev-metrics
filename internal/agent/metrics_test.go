package agent

import (
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
)

func TestMetrics_register(t *testing.T) {
	type fields struct {
		counter int64
		Metrics map[string]metric
	}
	type args struct {
		tp    model.MetricType
		name  string
		value string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "positive test",
			fields: fields{
				counter: 1,
				Metrics: map[string]metric{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				counter: tt.fields.counter,
				Metrics: tt.fields.Metrics,
			}
			m.register(tt.args.tp, tt.args.name, tt.args.value)
		})
	}
}
