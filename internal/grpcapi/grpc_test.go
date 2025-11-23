package grpcapi

import (
	"net"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewGRPCAPI(t *testing.T) {
	type fields struct {
		address string
		repo    repository.Repository
		opts    []Option
	}

	tests := []struct {
		name              string
		fields            fields
		wantErr           bool
		wantTrustedSubnet bool
		wantSubnetString  string
	}{
		{
			name: "no options",
			fields: fields{
				address: "localhost:8080",
				repo:    nil,
				opts:    nil,
			},
			wantErr:           false,
			wantTrustedSubnet: false,
		},
		{
			name: "with valid trusted subnet option",
			fields: fields{
				address: "localhost:9090",
				repo:    nil,
				opts: []Option{
					WithTrustedSubnet("192.168.1.0/24"),
				},
			},
			wantErr:           false,
			wantTrustedSubnet: true,
			wantSubnetString:  "192.168.1.0/24",
		},
		{
			name: "with invalid trusted subnet option",
			fields: fields{
				address: "localhost:9091",
				repo:    nil,
				opts: []Option{
					WithTrustedSubnet("invalid-subnet"),
				},
			},
			wantErr:           true,
			wantTrustedSubnet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGRPCAPI(
				tt.fields.address,
				tt.fields.repo,
				tt.fields.opts...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.fields.address, got.address)
			assert.Equal(t, tt.fields.repo, got.repo)

			if tt.wantTrustedSubnet {
				if assert.NotNil(t, got.trustedSubnet) {
					assert.Equal(t, tt.wantSubnetString, got.trustedSubnet.String())
					ip := net.ParseIP("192.168.1.10")
					assert.True(t, got.trustedSubnet.Contains(ip))
				}
			} else {
				assert.Nil(t, got.trustedSubnet)
			}
		})
	}
}
