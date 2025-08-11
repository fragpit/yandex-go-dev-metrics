package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/stretchr/testify/assert"
)

func TestRouter_checksumMiddleware(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	repo := memstorage.NewMemoryStorage()
	secretKey := []byte("test-secret-key")

	router := &Router{
		logger:    logger,
		repo:      repo,
		secretKey: secretKey,
	}

	// Test handler that responds with OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body to ensure it's available
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK: " + string(body)))
	})

	middleware := router.checksumMiddleware(testHandler)

	tests := []struct {
		name             string
		body             string
		headerValue      string
		expectedStatus   int
		expectedResponse string
		setupHeader      bool
	}{
		{
			name:             "missing checksum header",
			body:             `{"test": "data"}`,
			setupHeader:      false,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "checksum header is nil or unset\n",
		},
		{
			name:             "empty checksum header",
			body:             `{"test": "data"}`,
			headerValue:      "",
			setupHeader:      true,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "checksum header is nil or unset\n",
		},
		{
			name: "valid checksum",
			body: `{"test": "data"}`,
			headerValue: generateValidChecksum(
				`{"test": "data"}`,
				secretKey,
			),
			setupHeader:      true,
			expectedStatus:   http.StatusOK,
			expectedResponse: "OK: {\"test\": \"data\"}",
		},
		{
			name:             "invalid checksum",
			body:             `{"test": "data"}`,
			headerValue:      "invalid-checksum",
			setupHeader:      true,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "invalid request checksum\n",
		},
		{
			name: "wrong checksum for different body",
			body: `{"test": "different-data"}`,
			headerValue: generateValidChecksum(
				`{"test": "data"}`,
				secretKey,
			),
			setupHeader:      true,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "invalid request checksum\n",
		},
		{
			name:             "empty body with valid checksum",
			body:             "",
			headerValue:      generateValidChecksum("", secretKey),
			setupHeader:      true,
			expectedStatus:   http.StatusOK,
			expectedResponse: "OK: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.setupHeader {
				req.Header.Set("HashSHA256", tt.headerValue)
			}

			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code,
				"unexpected status code")
			assert.Equal(t, tt.expectedResponse, rr.Body.String(),
				"unexpected response body")
		})
	}
}

func generateValidChecksum(body string, secretKey []byte) string {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(body))
	return base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
}
