package main

import (
	"log/slog"
	"os"

	"github.com/fragpit/yandex-go-dev-metrics/internal/server"
	"github.com/fragpit/yandex-go-dev-metrics/pkg/utils/buildinfo"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	buildinfo.PrintBuildInfo(buildVersion, buildDate, buildCommit)

	if err := server.Run(); err != nil {
		slog.Error("server fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}
