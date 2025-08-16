package agent

import (
	"log/slog"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestReporter_reportMetrics(t *testing.T) {
	type fields struct {
		l         *slog.Logger
		repo      repository.Repository
		serverURL string
		secretKey []byte
		rateLimit int
	}

	type args struct {
		m map[string]model.Metric
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "empty metrics map",
			fields: fields{
				l:         slog.New(slog.DiscardHandler),
				repo:      nil,
				serverURL: "http://localhost:8080",
				secretKey: nil,
				rateLimit: 1,
			},
			args: args{m: map[string]model.Metric{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reporter{
				l:         tt.fields.l,
				repo:      tt.fields.repo,
				serverURL: tt.fields.serverURL,
				secretKey: tt.fields.secretKey,
				rateLimit: tt.fields.rateLimit,
			}
			err := r.reportMetrics(t.Context(), tt.args.m)
			assert.NoError(t, err)
		})
	}
}
