package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/fourtytwo42/keelmesh/internal/core"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/platform"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	engine         *core.Engine
	logger         *slog.Logger
	web            fs.FS
	startedAt      time.Time
	platform       *platform.Manager
	metricsHandler http.Handler
}

func New(engine *core.Engine, logger *slog.Logger, web fs.FS, managers ...*platform.Manager) *Server {
	var manager *platform.Manager
	if len(managers) > 0 {
		manager = managers[0]
	}
	server := &Server{engine: engine, logger: logger, web: web, startedAt: time.Now().UTC(), platform: manager}
	if manager != nil {
		registry := prometheus.NewRegistry()
		gauges := []struct {
			name, help string
			value      func(domain.PipelineMetricsV1) float64
		}{{"keelmesh_events_attempted_total", "Synthetic event delivery attempts.", func(m domain.PipelineMetricsV1) float64 { return float64(m.Attempted) }}, {"keelmesh_events_persisted_total", "Unique events persisted.", func(m domain.PipelineMetricsV1) float64 { return float64(m.UniqueInserted) }}, {"keelmesh_consumer_lag", "Current Kafka consumer-group lag.", func(m domain.PipelineMetricsV1) float64 { return float64(m.CurrentLag) }}, {"keelmesh_ingest_latency_p95_ms", "Rolling ingest p95 in milliseconds.", func(m domain.PipelineMetricsV1) float64 { return m.LatencyP95MS }}, {"keelmesh_worker_rebalances_total", "Observed worker rebalance events.", func(m domain.PipelineMetricsV1) float64 { return float64(m.RebalanceCount) }}, {"keelmesh_events_dropped_total", "Synthetic events intentionally dropped after bounded buffers fill.", func(m domain.PipelineMetricsV1) float64 { return float64(m.Dropped) }}}
		for _, g := range gauges {
			g := g
			registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: g.name, Help: g.help}, func() float64 { return g.value(manager.Snapshot().Metrics) }))
		}
		server.metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	}
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /api/v1/bootstrap", s.bootstrap)
	mux.HandleFunc("POST /api/v1/intents:compile", s.compile)
	mux.HandleFunc("POST /api/v1/plans", s.plans)
	mux.HandleFunc("POST /api/v1/plans/{action}", s.planAction)
	mux.HandleFunc("POST /api/v1/missions/{action}", s.missionAction)
	mux.HandleFunc("GET /api/v1/audit/{trace_id}", s.audit)
	mux.HandleFunc("GET /api/v1/resilience", s.resilience)
	mux.HandleFunc("POST /api/v1/faults", s.fault)
	mux.HandleFunc("POST /api/v1/scenarios/resilient-edge:reset", s.resetResilience)
	mux.HandleFunc("POST /api/v1/scenarios/resilient-edge:advance", s.advanceResilience)
	mux.HandleFunc("GET /api/v1/stream", s.stream)
	mux.HandleFunc("GET /api/v1/platform", s.platformSnapshot)
	mux.HandleFunc("GET /api/v1/metrics/snapshot", s.platformSnapshot)
	mux.HandleFunc("GET /metrics", s.metrics)
	mux.HandleFunc("POST /api/v1/load/runs", s.startLoadRun)
	mux.HandleFunc("POST /api/v1/load/runs/{action}", s.loadRunAction)
	mux.HandleFunc("POST /api/v1/platform/faults", s.platformFault)
	mux.HandleFunc("POST /api/v1/quarantine/{action}", s.quarantineAction)
	mux.HandleFunc("POST /api/v1/platform/replays", s.startReplay)
	mux.HandleFunc("GET /api/v1/platform/replays/{id}", s.getReplay)
	mux.HandleFunc("GET /api/v1/retrieval/similar", s.retrieval)
	mux.HandleFunc("GET /api/v1/evidence/{run_id}", s.evidence)
	mux.HandleFunc("POST /api/v1/scenarios/scale-lab:reset", s.resetPlatform)
	mux.Handle("GET /", spaHandler(s.web))
	return requestLog(s.logger, mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"name": "keelmesh-core", "status": "healthy", "version": "m3", "started_at": s.startedAt.Format(time.RFC3339)})
}
func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (s *Server) bootstrap(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.Bootstrap())
}
func (s *Server) compile(w http.ResponseWriter, r *http.Request) {
	var req core.CompileRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.Compile(req)
	respond(w, v, err, http.StatusCreated)
}
func (s *Server) plans(w http.ResponseWriter, r *http.Request) {
	var req core.PlansRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.GeneratePlans(req)
	respond(w, map[string]any{"plans": v}, err, http.StatusCreated)
}
func (s *Server) planAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if id, ok := strings.CutSuffix(action, ":preview"); ok && id != "" {
		s.preview(w, r, id)
		return
	}
	if id, ok := strings.CutSuffix(action, ":authorize"); ok && id != "" {
		s.authorize(w, r, id)
		return
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "NOT_FOUND", Message: "Unknown plan action."})
}
func (s *Server) preview(w http.ResponseWriter, r *http.Request, planID string) {
	var req core.PreviewRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.Preview(planID, req)
	respond(w, v, err, http.StatusOK)
}
func (s *Server) authorize(w http.ResponseWriter, r *http.Request, planID string) {
	var req core.AuthorizeRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.Authorize(planID, req)
	respond(w, v, err, http.StatusCreated)
}
func (s *Server) missionAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	id, ok := strings.CutSuffix(action, ":start")
	if !ok || id == "" {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "NOT_FOUND", Message: "Unknown mission action."})
		return
	}
	var req core.StartRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.Start(id, req)
	respond(w, v, err, http.StatusOK)
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"events": s.engine.Audit(r.PathValue("trace_id"))})
}

func (s *Server) resilience(w http.ResponseWriter, _ *http.Request) {
	v, err := s.engine.Resilience()
	respond(w, v, err, http.StatusOK)
}
func (s *Server) fault(w http.ResponseWriter, r *http.Request) {
	var req domain.FaultCommandV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.ApplyFault(req)
	respond(w, v, err, http.StatusOK)
}
func (s *Server) resetResilience(w http.ResponseWriter, r *http.Request) {
	var req domain.ResilienceMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.ResetResilience(req)
	respond(w, v, err, http.StatusOK)
}
func (s *Server) advanceResilience(w http.ResponseWriter, r *http.Request) {
	var req domain.FaultCommandV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.engine.AdvanceResilience(req)
	respond(w, v, err, http.StatusOK)
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{r.Host}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(1024)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	ch, unsubscribe := s.engine.Subscribe()
	defer unsubscribe()
	var platformCh <-chan domain.PlatformSnapshotV1
	var unsubscribePlatform func()
	if s.platform != nil {
		platformCh, unsubscribePlatform = s.platform.Subscribe()
		defer unsubscribePlatform()
	}
	initial := domain.StreamMessageV1{SchemaVersion: domain.SchemaVersion, Kind: "fleet.snapshot", Snapshot: ptr(s.engine.Snapshot())}
	if err := wsjson.Write(ctx, c, initial); err != nil {
		return
	}
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			writeCtx, stop := context.WithTimeout(ctx, 2*time.Second)
			err := wsjson.Write(writeCtx, c, msg)
			stop()
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		case snapshot, ok := <-platformCh:
			if !ok {
				platformCh = nil
				continue
			}
			writeCtx, stop := context.WithTimeout(ctx, 2*time.Second)
			err := wsjson.Write(writeCtx, c, domain.StreamMessageV1{SchemaVersion: 1, Kind: "platform.sample", Platform: &snapshot})
			stop()
			if err != nil {
				return
			}
		}
	}
}

func (s *Server) platformSnapshot(w http.ResponseWriter, _ *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Scale platform is disabled."})
		return
	}
	value := s.platform.Snapshot()
	value.Quarantine = s.platform.Quarantine(context.Background())
	writeJSON(w, http.StatusOK, value)
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if s.metricsHandler == nil {
		http.Error(w, "platform unavailable", http.StatusServiceUnavailable)
		return
	}
	s.metricsHandler.ServeHTTP(w, r)
}
func (s *Server) startLoadRun(w http.ResponseWriter, r *http.Request) {
	var req domain.LoadRunRequestV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.platform.StartRun(r.Context(), req)
	respondPlatform(w, v, err, http.StatusCreated)
}
func (s *Server) loadRunAction(w http.ResponseWriter, r *http.Request) {
	id, ok := strings.CutSuffix(r.PathValue("action"), ":stop")
	if !ok {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "NOT_FOUND", Message: "Unknown load action."})
		return
	}
	var req domain.PlatformMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.platform.StopRun(r.Context(), id, req)
	respondPlatform(w, v, err, http.StatusOK)
}
func (s *Server) platformFault(w http.ResponseWriter, r *http.Request) {
	var req domain.PlatformFaultCommandV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.platform.Fault(r.Context(), req)
	respondPlatform(w, v, err, http.StatusAccepted)
}
func (s *Server) quarantineAction(w http.ResponseWriter, r *http.Request) {
	id, ok := strings.CutSuffix(r.PathValue("action"), ":redrive")
	if !ok {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "NOT_FOUND", Message: "Unknown quarantine action."})
		return
	}
	var req domain.PlatformMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, err := s.platform.Redrive(r.Context(), id, req)
	respondPlatform(w, v, err, http.StatusAccepted)
}
func (s *Server) startReplay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		domain.PlatformMutationV1
		SourceRunID string `json:"source_run_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	v, err := s.platform.Replay(r.Context(), req.SourceRunID, req.PlatformMutationV1)
	respondPlatform(w, v, err, http.StatusCreated)
}
func (s *Server) getReplay(w http.ResponseWriter, r *http.Request) {
	v, err := s.platform.ReplayByID(r.Context(), r.PathValue("id"))
	respondPlatform(w, v, err, http.StatusOK)
}
func (s *Server) retrieval(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"hits": s.platform.Retrieval(r.Context(), r.URL.Query().Get("q"))})
}
func (s *Server) evidence(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.platform.Evidence(r.Context(), r.PathValue("run_id")))
}
func (s *Server) resetPlatform(w http.ResponseWriter, r *http.Request) {
	var req domain.PlatformMutationV1
	if !decode(w, r, &req) {
		return
	}
	err := s.platform.Reset(r.Context(), req)
	respondPlatform(w, map[string]string{"status": "reset"}, err, http.StatusOK)
}
func respondPlatform(w http.ResponseWriter, v any, err error, success int) {
	if err == nil {
		writeJSON(w, success, v)
		return
	}
	var pe *platform.Error
	if errors.As(err, &pe) {
		status := http.StatusUnprocessableEntity
		if pe.Code == "PLATFORM_STALE_STATE" || pe.Code == "FAULT_CONFLICT" {
			status = http.StatusConflict
		}
		if pe.Code == "PLATFORM_UNAVAILABLE" {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, domain.APIError{Code: pe.Code, Message: pe.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "INTERNAL", Message: err.Error()})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.APIError{Code: "INVALID_REQUEST", Message: "Invalid request body: " + err.Error()})
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, domain.APIError{Code: "INVALID_REQUEST", Message: "Only one JSON object is allowed."})
		return false
	}
	return true
}
func respond(w http.ResponseWriter, v any, err error, success int) {
	if err == nil {
		writeJSON(w, success, v)
		return
	}
	var ce *core.Error
	if errors.As(err, &ce) {
		status := http.StatusUnprocessableEntity
		if ce.Code == "STALE_STATE" {
			status = http.StatusConflict
		}
		writeJSON(w, status, domain.APIError{Code: ce.Code, Message: ce.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "INTERNAL", Message: "The request could not be completed."})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func ptr[T any](v T) *T { return &v }

func requestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func spaHandler(web fs.FS) http.Handler {
	files := http.FileServer(http.FS(web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(web, path); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			files.ServeHTTP(w, r2)
			return
		}
		files.ServeHTTP(w, r)
	})
}
