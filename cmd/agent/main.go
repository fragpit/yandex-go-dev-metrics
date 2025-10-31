package main

import (
	"log/slog"
	"os"

	"github.com/fragpit/yandex-go-dev-metrics/internal/agent"
	"github.com/fragpit/yandex-go-dev-metrics/pkg/utils/buildinfo"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	buildinfo.PrintBuildInfo(buildVersion, buildDate, buildCommit)

	if err := agent.Run(); err != nil {
		slog.Error("agent fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}
