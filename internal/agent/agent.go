package agent

import (
	"log"
	"time"
)

const (
	serverURL      = "http://localhost:8080"
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func Run() {
	pollTick := time.NewTicker(pollInterval)
	reportTick := time.NewTicker(reportInterval)

	m := NewMetrics()

	for {
		select {
		case <-pollTick.C:
			if err := m.pollMetrics(); err != nil {
				log.Printf("fatal error: %v", err)
			}
		case <-reportTick.C:
			if err := m.reportMetrics(); err != nil {
				log.Printf("fatal error: %v", err)
			}
		}
	}
}
