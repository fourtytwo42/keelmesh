package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed web/*
var webContent embed.FS

type statusResponse struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	StartedAt string `json:"started_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	startedAt := time.Now().UTC()

	webRoot, err := fs.Sub(webContent, "web")
	if err != nil {
		logger.Error("prepare embedded web root", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(webRoot)))
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, statusResponse{
			Name:      "keelmesh-core",
			Status:    "healthy",
			Version:   "bootstrap",
			StartedAt: startedAt.Format(time.RFC3339),
		})
	})
	mux.HandleFunc("GET /readyz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           requestLog(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func requestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(writer, request)
		logger.Info("http request",
			"method", request.Method,
			"path", request.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}
