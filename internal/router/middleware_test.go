package router

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRouter_decompressMiddleware(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	repo := memstorage.NewMemoryStorage()

	router := &Router{
		logger: logger,
		repo:   repo,
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	middleware := router.decompressMiddleware(testHandler)

	tests := []struct {
		name           string
		body           string
		compress       bool
		contentLength  int64
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "pass through non-gzip body",
			body:           "test data",
			compress:       false,
			expectedStatus: http.StatusOK,
			expectedBody:   "test data",
		},
		{
			name:           "decompress gzip body",
			body:           "test data",
			compress:       true,
			expectedStatus: http.StatusOK,
			expectedBody:   "test data",
		},
		{
			name:           "empty gzip body",
			body:           "",
			compress:       true,
			contentLength:  0,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.compress {
				var buf strings.Builder
				gw := gzip.NewWriter(&buf)
				_, err := gw.Write([]byte(tt.body))
				assert.NoError(t, err)
				err = gw.Close()
				assert.NoError(t, err)

				req = httptest.NewRequest(http.MethodPost, "/test",
					strings.NewReader(buf.String()))
				req.Header.Set("Content-Encoding", "gzip")
				if tt.contentLength > 0 {
					req.ContentLength = tt.contentLength
				}
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test",
					strings.NewReader(tt.body))
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestRouter_decompressMiddleware_InvalidGzip(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	repo := memstorage.NewMemoryStorage()

	router := &Router{
		logger: logger,
		repo:   repo,
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := router.decompressMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/test",
		strings.NewReader("not gzip data"))
	req.Header.Set("Content-Encoding", "gzip")
	req.ContentLength = 10

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to create reader")
}

func TestRouter_slogMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := memstorage.NewMemoryStorage()

	router := &Router{
		logger: logger,
		repo:   repo,
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	middleware := router.slogMiddleware(testHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "simple GET request",
			method:         http.MethodGet,
			path:           "/test",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with body",
			method:         http.MethodPost,
			path:           "/api/update",
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path,
				strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRouter_slogMiddleware_WithDebugLogging(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	repo := memstorage.NewMemoryStorage()

	router := &Router{
		logger: logger,
		repo:   repo,
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := router.slogMiddleware(testHandler)

	tests := []struct {
		name           string
		body           string
		compress       bool
		expectedStatus int
	}{
		{
			name:           "debug logging with plain body",
			body:           `{"test": "data"}`,
			compress:       false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "debug logging with gzip body",
			body:           `{"test": "compressed"}`,
			compress:       true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.compress {
				var buf strings.Builder
				gw := gzip.NewWriter(&buf)
				_, err := gw.Write([]byte(tt.body))
				require.NoError(t, err)
				err = gw.Close()
				require.NoError(t, err)

				req = httptest.NewRequest(http.MethodPost, "/test",
					strings.NewReader(buf.String()))
				req.Header.Set("Content-Encoding", "gzip")
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test",
					strings.NewReader(tt.body))
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
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
			keyPath: "testdata/private.pem",
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

func TestRouter_verifySubnetMiddleware(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	router := &Router{
		logger: logger,
		trustedSubnet: &net.IPNet{
			IP:   net.IPv4(192, 168, 1, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		headerValue    string
		setupHeader    bool
		expectedStatus int
		expectedBody   string
		expectCalled   bool
	}{
		{
			name:           "trusted ip",
			headerValue:    "192.168.1.1",
			setupHeader:    true,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			expectCalled:   true,
		},
		{
			name:           "missing header",
			setupHeader:    false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "x-real-ip header is nil or unset\n",
			expectCalled:   false,
		},
		{
			name:           "invalid cidr",
			headerValue:    "invalid-cidr",
			setupHeader:    true,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to parse x-real-ip\n",
			expectCalled:   false,
		},
		{
			name:           "forbidden ip",
			headerValue:    "10.0.0.1",
			setupHeader:    true,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Forbidden\n",
			expectCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				testHandler.ServeHTTP(w, r)
			})

			middleware := router.verifySubnetMiddleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.setupHeader {
				req.Header.Set("X-Real-IP", tt.headerValue)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			}
			assert.Equal(t, tt.expectCalled, called)
		})
	}
}
