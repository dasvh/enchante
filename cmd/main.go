package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/dasvh/enchante/internal/config"
	"github.com/dasvh/enchante/internal/probe"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func main() {
	configFile := flag.String("config", "testdata/BasicAuthNoDelay.yaml", "Path to the probe configuration file")
	flag.Parse()

	logger.Info("Loading configuration", "file", *configFile)

	cfg, err := config.LoadConfig(*configFile, logger)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("Loaded config", "config", cfg)

	probe.RunProbe(cfg, logger)
}
