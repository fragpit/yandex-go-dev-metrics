package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileObserver_Notify_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	obs := NewFileObserver(filePath)

	event := Event{
		Metrics:   nil,
		IPAddress: "127.0.0.1",
	}

	if err := obs.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal file content: %v", err)
	}

	if got.IPAddress != event.IPAddress {
		t.Fatalf("ip mismatch: got %q want %q", got.IPAddress, event.IPAddress)
	}
	if len(got.Metrics) != len(event.Metrics) {
		t.Fatalf(
			"metrics length mismatch: got %d want %d",
			len(got.Metrics),
			len(event.Metrics),
		)
	}
}

func TestFileObserver_Notify_OpenFileError(t *testing.T) {
	// passing a directory path should cause OpenFile to fail
	tmpDir := t.TempDir()
	obs := NewFileObserver(tmpDir)

	err := obs.Notify(context.Background(), Event{IPAddress: "1.2.3.4"})
	if err == nil {
		t.Fatalf("expected error when opening a directory as file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
