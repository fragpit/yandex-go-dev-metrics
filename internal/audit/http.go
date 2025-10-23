package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type HTTPObserver struct {
	URL    string
	client *http.Client
}

// NewHTTPObserver creates a new HTTPObserver with the given URL.
// The URL is the endpoint where audit events will be sent.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		URL:    url,
		client: &http.Client{},
	}
}

func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	slog.Debug(
		"sending http audit event",
		slog.Int("metrics_num", len(event.Metrics)),
		slog.String("client_ip", event.IPAddress),
	)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(event); err != nil {
		return fmt.Errorf("failed to encode audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.URL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create audit request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send audit request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return nil
}
