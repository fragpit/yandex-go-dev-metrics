package main

import (
	"log/slog"
	"os"

	"github.com/fragpit/yandex-go-dev-metrics/internal/agent"
)

func main() {
	if err := agent.Run(); err != nil {
		slog.Error("agent fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}
