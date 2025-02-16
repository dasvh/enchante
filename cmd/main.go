package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/dasvh/enchante/internal/config"
	"github.com/dasvh/enchante/internal/logger"
	"github.com/dasvh/enchante/internal/probe"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	configFile := flag.String("config", "probe_config.yaml", "Path to the probe configuration file")
	flag.Parse()

	newLogger := logger.NewLogger(*debug)
	newLogger.Info("Starting probe service", "debug_enabled", *debug)

	cfg, err := config.LoadConfig(*configFile, newLogger)
	if err != nil {
		newLogger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		newLogger.Warn("Shutdown signal received, exiting gracefully...")
		cancel()
	}()

	probe.RunProbe(ctx, cfg, newLogger)

	newLogger.Info("Probe execution completed")
}
