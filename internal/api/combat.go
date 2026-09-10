package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/fleetops"
)

func (s *Server) combatSnapshotV8(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.fleetops.CombatSnapshot())
}

func (s *Server) combatEntityV8(w http.ResponseWriter, r *http.Request) {
	value, err := s.fleetops.CombatEntity(r.PathValue("id"))
	respondV2(w, value, err, http.StatusOK)
}

func (s *Server) combatEngagementsV8(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"engagements": s.fleetops.CombatSnapshot().Engagements})
}

func (s *Server) createCombatEngagementV8(w http.ResponseWriter, r *http.Request) {
	var req fleetops.CombatEngagementRequest
	if !decode(w, r, &req) {
		return
	}
	value, err := s.fleetops.PlanCombatEngagement(req)
	respondV2(w, value, err, http.StatusCreated)
}

func (s *Server) combatEngagementActionV8(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if id, ok := cutAction(action, ":authorize"); ok {
		var req fleetops.CombatAuthorizeRequest
		if !decode(w, r, &req) {
			return
		}
		value, err := s.fleetops.AuthorizeCombatEngagement(id, req)
		respondV2(w, value, err, http.StatusOK)
		return
	}
	if id, ok := cutAction(action, ":stop"); ok {
		var req fleetops.Mutation
		if !decode(w, r, &req) {
			return
		}
		value, err := s.fleetops.StopCombatEngagement(id, req)
		respondV2(w, value, err, http.StatusOK)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"code": "ENGAGEMENT_NOT_FOUND", "message": "Unknown engagement action."})
}

func cutAction(value, suffix string) (string, bool) {
	if len(value) <= len(suffix) || value[len(value)-len(suffix):] != suffix {
		return "", false
	}
	return value[:len(value)-len(suffix)], true
}

func (s *Server) repairCombatVesselV8(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if id, ok := cutAction(action, ":arm"); ok {
		var req fleetops.CombatArmRequest
		if !decode(w, r, &req) {
			return
		}
		req.Armed = true
		value, err := s.fleetops.ArmCombatVessel(id, req)
		respondV2(w, value, err, http.StatusOK)
		return
	}
	if id, ok := cutAction(action, ":disarm"); ok {
		var req fleetops.CombatArmRequest
		if !decode(w, r, &req) {
			return
		}
		req.Armed = false
		value, err := s.fleetops.ArmCombatVessel(id, req)
		respondV2(w, value, err, http.StatusOK)
		return
	}
	id, ok := cutAction(action, ":repair")
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "COMBAT_ENTITY_NOT_FOUND", "message": "Unknown vessel combat action."})
		return
	}
	var req fleetops.CombatRepairRequest
	if !decode(w, r, &req) {
		return
	}
	value, err := s.fleetops.RepairCombatVessel(id, req)
	respondV2(w, value, err, http.StatusOK)
}

func (s *Server) resetCombatV8(w http.ResponseWriter, r *http.Request) {
	var req fleetops.Mutation
	if !decode(w, r, &req) {
		return
	}
	value, err := s.fleetops.ResetCombat(req)
	respondV2(w, value, err, http.StatusOK)
}

func (s *Server) combatEventsV8(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	last := int64(0)
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		last, _ = strconv.ParseInt(raw, 10, 64)
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		snapshot := s.fleetops.CombatSnapshot()
		if snapshot.StateVersion > last {
			payload, _ := json.Marshal(snapshot)
			_, _ = fmt.Fprintf(w, "id: %d\nevent: combat\ndata: %s\n\n", snapshot.StateVersion, payload)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			last = snapshot.StateVersion
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
