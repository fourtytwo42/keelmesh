package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/coordination"
)

func main() {
	configPath := flag.String("config", "", "path to a coordination node JSON configuration")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if *configPath == "" {
		logger.Error("coordination node config is required")
		os.Exit(2)
	}
	config, err := coordination.ConfigFromFile(*configPath)
	if err != nil {
		logger.Error("load coordination node config", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	manager, err := coordination.NewManager(config, logger)
	if err != nil {
		logger.Error("start coordination manager", "error", err)
		os.Exit(1)
	}
	if err := manager.StartManagement(ctx); err != nil {
		logger.Error("start coordination management endpoint", "error", err)
		os.Exit(1)
	}
	reloadSignals := make(chan os.Signal, 1)
	signal.Notify(reloadSignals, syscall.SIGHUP)
	defer signal.Stop(reloadSignals)
	go func() {
		for range reloadSignals {
			if err := manager.ReloadCredentials(); err != nil {
				logger.Error("coordination credential reload rejected", "error", err)
			} else {
				logger.Info("coordination credentials reloaded")
			}
		}
	}()
	logger.Info("coordination node ready", "node", config.Identity.NodeID, "cell", config.Identity.CellID, "radio", config.RaftAddress, "management", config.ManagementAddress)
	<-ctx.Done()
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = manager.Close(closeCtx)
}
