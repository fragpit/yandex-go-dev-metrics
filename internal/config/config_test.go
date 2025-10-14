package config

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"empty", "", false},
		{"no scheme", "example.com", false},
		{"scheme only", "http://", false},
		{"scheme_without_slashes", "http:example.com", false},
		{"invalid_scheme", "ftp://example.com", false},
		{"malformed", "://example.com", false},
		{"http_simple", "http://example.com", true},
		{"https_with_path", "https://example.com/path?x=1", true},
		{"http_with_port", "http://localhost:8080", true},
		{"ipv4", "http://127.0.0.1", true},
		{"ipv6", "http://[::1]", true},
		{"userinfo", "http://user:pass@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateURL(tt.raw); got != tt.want {
				t.Fatalf("validateURL(%q) = %v; want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestAgentConfig_Debug(t *testing.T) {
	cfg := &AgentConfig{
		LogLevel:       "debug",
		ServerURL:      "http://localhost:8080",
		PollInterval:   2,
		ReportInterval: 10,
		SecretKey:      []byte("test"),
		RateLimit:      1,
	}

	cfg.Debug()
}

func TestServerConfig_Debug(t *testing.T) {
	cfg := &ServerConfig{
		LogLevel:      "debug",
		Address:       "localhost:8080",
		FileStorePath: "/tmp/metrics.json",
		Restore:       true,
		DatabaseDSN:   "",
		SecretKey:     []byte("test"),
	}

	cfg.Debug()
}
