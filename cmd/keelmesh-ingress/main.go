package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type coordinatorResponse struct {
	ManagementURL string `json:"management_url"`
	Epoch         int64  `json:"epoch"`
}

func main() {
	core := env("KEELMESH_CORE_URL", "http://core:8080")
	faction := strings.ToUpper(env("KEELMESH_INGRESS_FACTION", "B"))
	client := &http.Client{Timeout: 2 * time.Second}
	resolve := func() (string, int64, error) {
		response, err := client.Get(core + "/api/v3/ingress/" + faction + "/coordinator")
		if err != nil {
			return "", 0, err
		}
		defer response.Body.Close()
		var value coordinatorResponse
		err = json.NewDecoder(response.Body).Decode(&value)
		return value.ManagementURL, value.Epoch, err
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingress-healthz" {
			target, epoch, err := resolve()
			if err != nil {
				http.Error(w, "coordinator unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy", "faction": faction, "coordinator_url": target, "epoch": epoch})
			return
		}
		target, _, err := resolve()
		if err != nil {
			http.Error(w, "coordinator discovery unavailable", http.StatusServiceUnavailable)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v3/") {
			target = core
			r.Header.Set("X-KeelMesh-Faction", faction)
		}
		u, err := url.Parse(target)
		if err != nil {
			http.Error(w, "invalid coordinator target", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "current coordinator unavailable", http.StatusBadGateway)
		}
		proxy.ServeHTTP(w, r)
	})
	server := &http.Server{Addr: ":8082", Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("player %s ingress listening on %s", faction, server.Addr)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
