package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fourtytwo42/keelmesh/internal/agent"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (s *Server) assistantTurnV4(w http.ResponseWriter, r *http.Request) {
	var request domain.AssistantTurnRequestV2
	if !decode(w, r, &request) {
		return
	}
	if s.agent == nil || s.fleetops == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "AI_UNAVAILABLE", Message: "Command-scene runtime is unavailable."})
		return
	}
	if s.memory != nil {
		s.memory.SyncFleet(r.Context(), s.fleetops.Snapshot())
		request.MemoryContext = ptrContext(s.memory.Assemble(r.Context(), request.RequestID, request.ActorIdentity, request.SessionID, request.ActiveMissionID, request.Text))
	}
	value, err := s.agent.CreateAssistantTurn(r.Context(), request, s.fleetops.Snapshot())
	if err == nil && s.memory != nil {
		s.memory.RecordExchange(r.Context(), value.ID, request.ActorIdentity, request.SessionID, request.ActiveMissionID, request.Text, value.Assistant.Speech, value.Assistant.Provider)
		s.memory.SaveScene(r.Context(), value.Scene)
	}
	respondAgent(w, value, err, http.StatusCreated)
}

func (s *Server) cancelAssistantTurnV4(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "AI_UNAVAILABLE", Message: "Command-scene runtime is unavailable."})
		return
	}
	id, ok := strings.CutSuffix(r.PathValue("action"), ":cancel")
	if !ok || id == "" {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "SURFACE_NOT_FOUND", Message: "Unknown assistant-turn action."})
		return
	}
	value, err := s.agent.Turn(id)
	if err != nil {
		respondAgent(w, nil, err, http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": value.ID, "state": "completed", "detail": "The turn had already reached a terminal state; no committed scene action was reversed."})
}

func (s *Server) assistantTurnEventsV4(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "AI_UNAVAILABLE", Message: "Command-scene runtime is unavailable."})
		return
	}
	after := int64(0)
	if header := r.Header.Get("Last-Event-ID"); header != "" {
		after, _ = strconv.ParseInt(header, 10, 64)
	}
	if query := r.URL.Query().Get("after"); query != "" {
		after, _ = strconv.ParseInt(query, 10, 64)
	}
	events := s.agent.SceneEvents(r.PathValue("id"), after)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	for _, event := range events {
		raw, _ := json.Marshal(event)
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Kind, raw)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *Server) scenesV4(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "AI_UNAVAILABLE", Message: "Command-scene runtime is unavailable."})
		return
	}
	if s.fleetops != nil {
		s.agent.RefreshScenes(s.fleetops.Snapshot())
	}
	if s.memory != nil {
		s.agent.RestoreScenes(s.memory.Scenes(r.URL.Query().Get("actor_identity"), r.URL.Query().Get("session_id")))
	}
	actor, session := r.URL.Query().Get("actor_identity"), r.URL.Query().Get("session_id")
	turns := []domain.ConversationTurnV1{}
	if s.memory != nil {
		turns = s.memory.ConversationTurns(actor, session, 50)
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenes": s.agent.Scenes(actor, session), "turns": turns})
}
func (s *Server) assistantHistoryV4(w http.ResponseWriter, r *http.Request) { s.scenesV4(w, r) }
func (s *Server) sceneV4(w http.ResponseWriter, r *http.Request) {
	value, err := s.agent.SceneForSession(r.PathValue("id"), r.URL.Query().Get("actor_identity"), r.URL.Query().Get("session_id"))
	respondAgent(w, value, err, http.StatusOK)
}
func (s *Server) sceneCatalogV4(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, agent.SceneCatalog())
}

func (s *Server) sceneMutationV4(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("action")
	operations := []string{"pin", "unpin", "dismiss"}
	for _, operation := range operations {
		suffix := ":" + operation
		if id, ok := strings.CutSuffix(raw, suffix); ok && id != "" {
			var request domain.SceneMutationV1
			if !decode(w, r, &request) {
				return
			}
			value, err := s.agent.MutateScene(id, operation, request)
			if err == nil && s.memory != nil {
				s.memory.SaveScene(r.Context(), value)
			}
			respondAgent(w, value, err, http.StatusOK)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "SURFACE_NOT_FOUND", Message: "Unknown command-scene mutation."})
}

func (s *Server) sceneActionV4(w http.ResponseWriter, r *http.Request) {
	var request domain.SceneActionRequestV1
	if !decode(w, r, &request) {
		return
	}
	value, err := s.agent.ApplySceneAction(r.PathValue("id"), request)
	if err == nil && s.memory != nil {
		s.memory.SaveScene(r.Context(), value)
	}
	respondAgent(w, value, err, http.StatusOK)
}
