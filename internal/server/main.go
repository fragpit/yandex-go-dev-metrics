package server

import (
	"log"

	"github.com/fragpit/yandex-go-dev-metrics/internal/config"
	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

func Run() error {
	cfg := config.NewServerConfig()
	st := memstorage.NewMemoryStorage()
	router := router.NewRouter(st)

	log.Printf("starting server (%v)", cfg.Address)
	if err := router.Run(cfg.Address); err != nil {
		return err
	}

	return nil
}
