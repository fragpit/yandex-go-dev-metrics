package server

import (
	"log"

	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

func Run() {
	log.Println("server started")

	st := memstorage.NewMemoryStorage()

	router := router.NewRouter(st)
	if err := router.Run(); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}
