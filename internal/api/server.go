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
)

type Server struct {
	engine    *core.Engine
	logger    *slog.Logger
	web       fs.FS
	startedAt time.Time
}

func New(engine *core.Engine, logger *slog.Logger, web fs.FS) *Server {
	return &Server{engine: engine, logger: logger, web: web, startedAt: time.Now().UTC()}
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
	mux.HandleFunc("GET /api/v1/stream", s.stream)
	mux.Handle("GET /", spaHandler(s.web))
	return requestLog(s.logger, mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"name": "keelmesh-core", "status": "healthy", "version": "m1", "started_at": s.startedAt.Format(time.RFC3339)})
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
		}
	}
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
