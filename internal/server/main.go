package server

import (
	"github.com/fragpit/yandex-go-dev-metrics/internal/router"
	"github.com/fragpit/yandex-go-dev-metrics/internal/storage/memstorage"
)

func Run() {
	st := memstorage.NewMemoryStorage()

	router := router.New(st)

	router.Run()
}
