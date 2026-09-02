package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fourtytwo42/keelmesh/internal/arena"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (s *Server) requireArena(w http.ResponseWriter) bool {
	if s.arena == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "ARENA_UNAVAILABLE", Message: "Fleet Arena is disabled."})
		return false
	}
	return true
}
func faction(r *http.Request) string {
	f := strings.ToUpper(r.Header.Get("X-KeelMesh-Faction"))
	if f == "" {
		f = strings.ToUpper(r.URL.Query().Get("faction"))
	}
	if f != "B" {
		f = "A"
	}
	return f
}
func (s *Server) arenaSnapshotV3(w http.ResponseWriter, r *http.Request) {
	if !s.requireArena(w) {
		return
	}
	writeJSON(w, http.StatusOK, s.arena.Snapshot(faction(r)))
}
func (s *Server) arenaInfrastructureV3(w http.ResponseWriter, r *http.Request) {
	if !s.requireArena(w) {
		return
	}
	// Filter player ingress before serialization. Only the referee-facing core,
	// which sends no faction header, may inspect both factions' node topology.
	if r.Header.Get("X-KeelMesh-Faction") != "" {
		writeJSON(w, http.StatusOK, s.arena.Snapshot(faction(r)))
		return
	}
	writeJSON(w, http.StatusOK, s.arena.InfrastructureSnapshot())
}
func (s *Server) createMatchV3(w http.ResponseWriter, r *http.Request) {
	var req domain.ArenaMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.Start(req)
	respondArena(w, v, e, http.StatusCreated)
}
func (s *Server) matchActionV3(w http.ResponseWriter, r *http.Request) {
	if _, ok := strings.CutSuffix(r.PathValue("action"), ":start"); !ok {
		writeJSON(w, 404, domain.APIError{Code: "NOT_FOUND", Message: "Unknown match action."})
		return
	}
	var req domain.ArenaMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.Start(req)
	respondArena(w, v, e, http.StatusOK)
}
func (s *Server) playerStateV3(w http.ResponseWriter, r *http.Request) {
	if !s.requireArena(w) {
		return
	}
	writeJSON(w, http.StatusOK, s.arena.Snapshot(faction(r)))
}
func (s *Server) arenaFaultV3(w http.ResponseWriter, r *http.Request) {
	var req arena.FaultRequest
	if !decode(w, r, &req) {
		return
	}
	if f := r.Header.Get("X-KeelMesh-Faction"); f != "" {
		req.Faction, req.ActorID = f, f
	}
	v, e := s.arena.Fault(req)
	respondArena(w, v, e, http.StatusOK)
}
func (s *Server) arenaAdvanceV3(w http.ResponseWriter, r *http.Request) {
	var req struct {
		domain.ArenaMutationV1
		Seconds int `json:"seconds"`
	}
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.Advance(req.ArenaMutationV1, req.Seconds)
	respondArena(w, v, e, http.StatusOK)
}
func (s *Server) planEngagementV3(w http.ResponseWriter, r *http.Request) {
	var req arena.PlanRequest
	if !decode(w, r, &req) {
		return
	}
	if f := r.Header.Get("X-KeelMesh-Faction"); f != "" {
		req.Faction, req.ActorID = f, f
	}
	v, e := s.arena.PlanEngagement(req)
	respondArena(w, v, e, http.StatusCreated)
}
func (s *Server) engagementActionV3(w http.ResponseWriter, r *http.Request) {
	id, ok := strings.CutSuffix(r.PathValue("action"), ":authorize")
	if !ok {
		writeJSON(w, 404, domain.APIError{Code: "NOT_FOUND", Message: "Unknown engagement action."})
		return
	}
	var req arena.AuthorizeRequest
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.Authorize(id, req)
	respondArena(w, v, e, http.StatusCreated)
}
func (s *Server) effectV3(w http.ResponseWriter, r *http.Request) {
	var req arena.EffectRequest
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.ApplyEffect(req)
	respondArena(w, v, e, http.StatusAccepted)
}
func (s *Server) coordinationV3(w http.ResponseWriter, r *http.Request) {
	if !s.requireArena(w) {
		return
	}
	v := s.arena.Snapshot(strings.ToUpper(r.PathValue("id")))
	writeJSON(w, http.StatusOK, v.Coordinators[0])
}
func (s *Server) ingressCoordinatorV3(w http.ResponseWriter, r *http.Request) {
	if !s.requireArena(w) {
		return
	}
	v := s.arena.Snapshot(strings.ToUpper(r.PathValue("faction_id")))
	c := v.Coordinators[0]
	writeJSON(w, http.StatusOK, map[string]any{"faction": c.Faction, "coordinator": c, "management_url": "http://" + nodeIP(v.Nodes, c.NodeID) + ":8080", "epoch": c.Epoch})
}
func nodeIP(nodes []domain.ArenaNodeV1, id string) string {
	for _, n := range nodes {
		if n.ID == id {
			return n.PlannedManagementIP
		}
	}
	return "unavailable"
}
func (s *Server) createAgentSessionV3(w http.ResponseWriter, r *http.Request) {
	var req arena.AgentMessageRequest
	if !decode(w, r, &req) {
		return
	}
	if f := r.Header.Get("X-KeelMesh-Faction"); f != "" {
		req.Faction, req.ActorID = f, f
	}
	v, e := s.arena.CreateSession(req)
	respondArena(w, v, e, http.StatusCreated)
}
func (s *Server) agentMessageV3(w http.ResponseWriter, r *http.Request) {
	var req arena.AgentMessageRequest
	if !decode(w, r, &req) {
		return
	}
	if f := r.Header.Get("X-KeelMesh-Faction"); f != "" {
		req.Faction, req.ActorID = f, f
	}
	v, e := s.arena.AgentMessage(r.PathValue("id"), req)
	respondArena(w, v, e, http.StatusOK)
}
func (s *Server) workspaceActionV3(w http.ResponseWriter, r *http.Request) {
	var req arena.ActionRequest
	if !decode(w, r, &req) {
		return
	}
	req.SessionID = r.PathValue("session_id")
	v, e := s.arena.WorkspaceAction(req)
	respondArena(w, v, e, http.StatusOK)
}
func (s *Server) resetArenaV3(w http.ResponseWriter, r *http.Request) {
	var req domain.ArenaMutationV1
	if !decode(w, r, &req) {
		return
	}
	v, e := s.arena.Reset(req)
	respondArena(w, v, e, http.StatusOK)
}
func respondArena(w http.ResponseWriter, v any, err error, success int) {
	if err == nil {
		writeJSON(w, success, v)
		return
	}
	var ae *arena.Error
	if errors.As(err, &ae) {
		status := http.StatusUnprocessableEntity
		if strings.Contains(ae.Code, "STALE") || strings.Contains(ae.Code, "CONFLICT") {
			status = http.StatusConflict
		}
		if ae.Code == "PROTECTED_PLANE" || ae.Code == "HUMAN_APPROVAL_REQUIRED" {
			status = http.StatusForbidden
		}
		writeJSON(w, status, domain.APIError{Code: ae.Code, Message: ae.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "INTERNAL", Message: "Arena request failed."})
}
