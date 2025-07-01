package main

import (
	"log"

	"github.com/fragpit/yandex-go-dev-metrics/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatalf("server fatal error: %v", err)
	}
}
