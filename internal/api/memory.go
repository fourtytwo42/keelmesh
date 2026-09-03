package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/memory"
)

func ptrContext(value domain.ContextAssemblyV1) *domain.ContextAssemblyV1 { return &value }

func (s *Server) memorySnapshotV5(w http.ResponseWriter, _ *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	if s.fleetops != nil {
		s.memory.SyncFleet(context.Background(), s.fleetops.Snapshot())
	}
	writeJSON(w, http.StatusOK, s.memory.Snapshot())
}
func (s *Server) memorySearchV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	var request memory.SearchRequest
	if !decode(w, r, &request) {
		return
	}
	hits, receipt, err := s.memory.Search(r.Context(), request)
	respondMemory(w, map[string]any{"hits": hits, "receipt": receipt}, err, http.StatusOK)
}
func (s *Server) memoryItemV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	value, err := s.memory.Item(r.PathValue("id"), r.URL.Query().Get("actor_identity"))
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memoryItemActionV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	id, ok := strings.CutSuffix(r.PathValue("action"), ":forget")
	if !ok || id == "" {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "MEMORY_SOURCE_INVALID", Message: "Unknown memory item action."})
		return
	}
	var request memory.Mutation
	if !decode(w, r, &request) {
		return
	}
	value, err := s.memory.Forget(r.Context(), id, request)
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memoryCandidatesV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": s.memory.Candidates(r.URL.Query().Get("actor_identity"))})
}
func (s *Server) memoryCandidateActionV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	raw := r.PathValue("action")
	decision := ""
	id := ""
	for _, v := range []string{"approve", "reject"} {
		if candidate, ok := strings.CutSuffix(raw, ":"+v); ok {
			id = candidate
			decision = v
		}
	}
	if id == "" {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "MEMORY_SOURCE_INVALID", Message: "Unknown memory candidate action."})
		return
	}
	var request memory.CandidateMutation
	if !decode(w, r, &request) {
		return
	}
	value, err := s.memory.DecideCandidate(r.Context(), id, decision, request)
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memoryContextV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	value, err := s.memory.Context(r.PathValue("turn_id"), r.URL.Query().Get("actor_identity"))
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memoryEntityV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	value, err := s.memory.Entity(r.PathValue("id"), r.URL.Query().Get("actor_identity"))
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memorySyncV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": s.memory.Sync(r.URL.Query().Get("actor_identity"))})
}
func (s *Server) memoryReplayV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	var request memory.Mutation
	if !decode(w, r, &request) {
		return
	}
	value, err := s.memory.StartReplay(r.Context(), request)
	respondMemory(w, value, err, http.StatusCreated)
}
func (s *Server) memoryReplayResultV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	value, err := s.memory.Replay(r.PathValue("id"))
	respondMemory(w, value, err, http.StatusOK)
}
func (s *Server) memoryResetV5(w http.ResponseWriter, r *http.Request) {
	if s.memory == nil {
		memoryUnavailable(w)
		return
	}
	var request memory.Mutation
	if !decode(w, r, &request) {
		return
	}
	value, err := s.memory.Reset(r.Context(), request)
	respondMemory(w, value, err, http.StatusOK)
}

func memoryUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "MEMORY_UNAVAILABLE", Message: "Central memory is unavailable; mission authority remains operational."})
}
func respondMemory(w http.ResponseWriter, value any, err error, status int) {
	if err == nil {
		writeJSON(w, status, value)
		return
	}
	var problem *memory.Error
	if errors.As(err, &problem) {
		code := http.StatusUnprocessableEntity
		if strings.Contains(problem.Code, "STALE") || strings.Contains(problem.Code, "HASH") {
			code = http.StatusConflict
		}
		if strings.Contains(problem.Code, "DENIED") {
			code = http.StatusForbidden
		}
		writeJSON(w, code, domain.APIError{Code: problem.Code, Message: problem.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "MEMORY_UNAVAILABLE", Message: "Memory operation failed."})
}
