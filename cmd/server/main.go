package main

import (
	"log/slog"
	"os"

	"github.com/fragpit/yandex-go-dev-metrics/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		slog.Error("server fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}
