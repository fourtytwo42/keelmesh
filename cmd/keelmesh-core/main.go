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
	"syscall"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/api"
	"github.com/fourtytwo42/keelmesh/internal/core"
)

//go:embed web/*
var webContent embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	webRoot, err := fs.Sub(webContent, "web")
	if err != nil {
		logger.Error("prepare embedded web root", "error", err)
		os.Exit(1)
	}

	engine := core.New()
	serverAPI := api.New(engine, logger, webRoot)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           serverAPI.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go engine.Run(shutdownContext)

	go func() {
		<-shutdownContext.Done()
		contextWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(contextWithTimeout); shutdownErr != nil {
			logger.Error("graceful shutdown", "error", shutdownErr)
		}
	}()

	logger.Info("keelmesh core listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
