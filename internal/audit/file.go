package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type FileObserver struct {
	mu       sync.Mutex
	filePath string
}

func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{
		filePath: filePath,
		mu:       sync.Mutex{},
	}
}

func (o *FileObserver) Notify(ctx context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	slog.Debug(
		"sending file audit event",
		slog.Int("metrics_num", len(event.Metrics)),
		slog.String("client_ip", event.IPAddress),
	)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	file, err := os.OpenFile(
		o.filePath,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := json.NewEncoder(file).Encode(event); err != nil {
		return fmt.Errorf("failed to encode audit event: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	return nil
}
