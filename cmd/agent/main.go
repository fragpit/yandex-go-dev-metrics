package main

import (
	"log"

	"github.com/fragpit/yandex-go-dev-metrics/internal/agent"
)

func main() {
	if err := agent.Run(); err != nil {
		log.Fatalf("agent fatal error: %v", err)
	}
}
