package agent

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
)

func Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	cfg := config.NewAgentConfig()

	pollTick := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	reportTick := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)

	m := NewMetrics()
	if err := m.pollMetrics(); err != nil {
		return err
	}

	for {
		select {
		case <-pollTick.C:
			if err := m.pollMetrics(); err != nil {
				return err
			}
		case <-reportTick.C:
			m.reportMetrics(cfg.ServerURL)
		case <-ctx.Done():
			log.Println("agent shut down")
			return nil
		}
	}
}
