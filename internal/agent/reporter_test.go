package agent

import (
	"log/slog"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestReporter_reportMetrics(t *testing.T) {
	type fields struct {
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
				transport: &RESTTransport{
					serverURL: tt.fields.serverURL,
					secretKey: tt.fields.secretKey,
					rateLimit: tt.fields.rateLimit,
				},
			}
			err := r.transport.SendMetrics(t.Context(), tt.args.m)
			assert.NoError(t, err)
		})
	}
}

func TestNewReporter(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	reporter, err := NewReporter(
		logger,
		nil,
		"http://localhost:8080",
		[]byte("secret"),
		1,
		"",
		"",
	)

	assert.NoError(t, err)
	assert.NotNil(t, reporter)
}

func TestReadKey(t *testing.T) {
	tests := []struct {
		name    string
		keyPath string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid RSA key",
			keyPath: "testdata/public.pem",
			wantErr: false,
		},
		{
			name:    "file not found",
			keyPath: "testdata/nonexistent_key.pem",
			wantErr: true,
			errMsg:  "failed to read file",
		},
		{
			name:    "invalid PEM format",
			keyPath: "testdata/invalid_key.pem",
			wantErr: true,
			errMsg:  "invalid key format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := readKey(tt.keyPath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
			}
		})
	}
}
