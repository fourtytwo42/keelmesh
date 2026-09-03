package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/agent"
	"github.com/fourtytwo42/keelmesh/internal/api"
	"github.com/fourtytwo42/keelmesh/internal/arena"
	"github.com/fourtytwo42/keelmesh/internal/core"
	"github.com/fourtytwo42/keelmesh/internal/fleetops"
	"github.com/fourtytwo42/keelmesh/internal/platform"
)

//go:embed web/*
var webContent embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := platform.ConfigFromEnv()
	role := "core"
	for i, arg := range os.Args {
		if arg == "--role" && i+1 < len(os.Args) {
			role = strings.TrimSpace(os.Args[i+1])
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	switch role {
	case "migrate":
		if err := platform.Migrate(ctx, cfg); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}
		return
	case "topic-init":
		if err := platform.InitTopics(ctx, cfg); err != nil {
			logger.Error("topic initialization failed", "error", err)
			os.Exit(1)
		}
		return
	case "loadgen":
		if err := platform.RunLoadgen(ctx, cfg, logger); err != nil {
			logger.Error("load generator failed", "error", err)
			os.Exit(1)
		}
		return
	case "worker-supervisor":
		if err := platform.RunWorkerSupervisor(ctx, cfg, logger); err != nil {
			logger.Error("worker supervisor failed", "error", err)
			os.Exit(1)
		}
		return
	case "worker-child":
		if err := platform.RunWorkerChild(ctx, cfg, logger); err != nil {
			logger.Error("worker child failed", "error", err)
			os.Exit(1)
		}
		return
	}

	webRoot, err := fs.Sub(webContent, "web")
	if err != nil {
		logger.Error("prepare embedded web root", "error", err)
		os.Exit(1)
	}

	engine := core.New()
	platformManager := platform.NewManager(cfg, logger)
	agentManager := agent.NewManager(agent.ConfigFromEnv(), logger)
	fleetManager := fleetops.New(cfg.DatabaseURL, logger)
	arenaManager := arena.NewFromEnv()
	serverAPI := api.New(engine, logger, webRoot, platformManager, agentManager, fleetManager, arenaManager)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           serverAPI.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go engine.Run(ctx)
	go platformManager.Run(ctx)
	go agentManager.Run(ctx)
	go fleetManager.Run(ctx)
	privateServer := &http.Server{Addr: ":8081", Handler: privateHandler(agentManager, fleetManager, arenaManager), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		logger.Info("keelmesh private AI boundary listening", "address", privateServer.Addr)
		if err := privateServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("private AI boundary", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		contextWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(contextWithTimeout); shutdownErr != nil {
			logger.Error("graceful shutdown", "error", shutdownErr)
		}
		_ = privateServer.Shutdown(contextWithTimeout)
	}()

	logger.Info("keelmesh core listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}

func privateHandler(manager *agent.Manager, fleet *fleetops.Manager, arenaManager *arena.Manager) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mcp", manager.MCPHandler())
	mux.Handle("/mcp/control", manager.MCPControlHandler(fleet, arenaManager))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","boundary":"mcp"}`))
	})
	return mux
}
