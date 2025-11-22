package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	privName = "private.pem"
	pubName  = "public.pem"
)

func TestGenerateKeys(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name: "success",
			path: "/tmp",
		},
		{
			name: "success relative path",
			path: "../keygen/",
		},
		{
			name:    "fail non existent path",
			path:    "/tmp/asd123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generateKeys(tt.path)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			priv := filepath.Join(tt.path, privName)
			pub := filepath.Join(tt.path, pubName)

			_, err = os.Stat(priv)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			_, err = os.Stat(pub)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			err = os.Remove(priv)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			err = os.Remove(pub)
			if !tt.wantErr {
				assert.NoError(t, err)
			}
		})
	}
}
