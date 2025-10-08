package audit

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPObserver_Notify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("expected POST method, got %s", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Fatalf("expected content-type application/json, got %s", ct)
				}
				_, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				w.WriteHeader(http.StatusOK)
			}),
		)
		defer ts.Close()

		o := NewHTTPObserver(ts.URL)
		o.client = ts.Client()

		var ev Event
		if err := o.Notify(context.Background(), ev); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		ts := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
		)
		defer ts.Close()

		o := NewHTTPObserver(ts.URL)
		o.client = ts.Client()

		var ev Event
		err := o.Notify(context.Background(), ev)
		if err == nil {
			t.Fatal("expected error for non-200 status, got nil")
		}
		if !strings.Contains(err.Error(), "invalid status code:") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("client error", func(t *testing.T) {
		o := NewHTTPObserver("http://example.invalid")

		o.client = &http.Client{
			Transport: errTransport{err: errors.New("network down")},
		}

		var ev Event
		err := o.Notify(context.Background(), ev)
		if err == nil {
			t.Fatal("expected error when client.Do fails, got nil")
		}
		if !strings.Contains(err.Error(), "failed to send audit request") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

type errTransport struct{ err error }

func (et errTransport) RoundTrip(
	*http.Request,
) (*http.Response, error) {
	return nil, et.err
}
