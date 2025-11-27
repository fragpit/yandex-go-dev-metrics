package agent

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalIPFor(t *testing.T) {
	tests := []struct {
		name       string
		serverHost string
		wantErr    bool
	}{
		{
			name:       "valid host",
			serverHost: "127.0.0.1",
			wantErr:    false,
		},
		{
			name:       "invalid host",
			serverHost: "invalid-hostname-!@#$",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := localIPFor(tt.serverHost)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ip)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ip)

			parsedIP := net.ParseIP(ip.String())
			assert.NotNil(t, parsedIP)
			assert.NotNil(t, parsedIP.To4())
		})
	}
}
