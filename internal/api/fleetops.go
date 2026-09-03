package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/fleetops"
)

func (s *Server) fleetV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.fleetops.Snapshot())
}
func (s *Server) vesselV2(w http.ResponseWriter, r *http.Request) {
	v, err := s.fleetops.Vessel(r.PathValue("id"))
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) surfaceContactV2(w http.ResponseWriter, r *http.Request) {
	v, err := s.fleetops.SurfaceContact(r.PathValue("id"))
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) patchVesselV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PatchVesselRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.PatchVessel(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) reachabilityV2(w http.ResponseWriter, r *http.Request) {
	v, err := s.fleetops.Reachability(r.PathValue("id"))
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) groupsV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"groups": s.fleetops.Snapshot().Groups})
}
func (s *Server) collectionsV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"collections": s.fleetops.Snapshot().Collections})
}
func (s *Server) missionsV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"missions": s.fleetops.Snapshot().Missions})
}
func (s *Server) missionV2(w http.ResponseWriter, r *http.Request) {
	for _, v := range s.fleetops.Snapshot().Missions {
		if v.ID == r.PathValue("id") {
			writeJSON(w, http.StatusOK, v)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "MISSION_NOT_FOUND", Message: "Mission not found."})
}

func (s *Server) trajectoryV2(w http.ResponseWriter, r *http.Request) {
	value, err := s.fleetops.TrajectoryProgram(r.PathValue("id"))
	respondV2(w, value, err, http.StatusOK)
}

func (s *Server) createGroupV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.CreateGroupRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.CreateGroup(req)
	respondV2(w, v, err, http.StatusCreated)
}
func (s *Server) patchGroupV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PatchGroupRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.PatchGroup(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) groupRouteCommandV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.GroupRouteCommandRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.CommandGroupRoute(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) moveGroupMemberV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.MoveGroupMemberRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.MoveGroupMember(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) deleteGroupV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.Mutation
	if !decode(w, r, &req) {
		return
	}
	err := s.fleetops.DeleteGroup(r.PathValue("id"), req)
	respondV2(w, map[string]bool{"deleted": err == nil}, err, http.StatusOK)
}
func (s *Server) createCollectionV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.CreateCollectionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.CreateCollection(req)
	respondV2(w, v, err, http.StatusCreated)
}
func (s *Server) patchCollectionV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PatchCollectionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.PatchCollection(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) createMissionV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.CreateMissionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.CreateMission(req)
	respondV2(w, v, err, http.StatusCreated)
}
func (s *Server) resetFleetOperationsV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.Mutation
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.ResetOperations(req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) patchMissionV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PatchMissionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.PatchMission(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) deleteMissionV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.Mutation
	if !decode(w, r, &req) {
		return
	}
	err := s.fleetops.DeleteMission(r.PathValue("id"), req)
	respondV2(w, map[string]bool{"deleted": err == nil}, err, http.StatusOK)
}
func (s *Server) geometryV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.GeometryRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.SetGeometry(r.PathValue("id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) compileV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.CompileRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.Compile(r.PathValue("id"), req)
	if err == nil {
		advisor := fleetops.DeterministicAdvisor(len(v.TargetIDs), v.GuidanceKind, "AI service unavailable")
		if context, contextErr := s.fleetops.PlanningContext(v.ID); contextErr == nil && s.agent != nil {
			if proposed, advisorErr := s.agent.MissionOptions(r.Context(), context); advisorErr == nil {
				advisor = proposed
			} else {
				s.logger.Warn("mission advisor degraded to deterministic fallback", "error", advisorErr)
			}
		}
		v, err = s.fleetops.ApplyAdvisor(v.ID, advisor)
	}
	respondV2(w, v, err, http.StatusCreated)
}
func (s *Server) plansV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PlansRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.GeneratePlans(r.PathValue("id"), req)
	respondV2(w, map[string]any{"plans": v}, err, http.StatusCreated)
}
func (s *Server) planActionV2(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if id, ok := strings.CutSuffix(action, ":preview"); ok {
		r.SetPathValue("plan_id", id)
		s.previewV2(w, r)
		return
	}
	if id, ok := strings.CutSuffix(action, ":authorize"); ok {
		r.SetPathValue("plan_id", id)
		s.authorizeV2(w, r)
		return
	}
	if id, ok := strings.CutSuffix(action, ":start"); ok {
		r.SetPathValue("plan_id", id)
		s.startV2(w, r)
		return
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "NOT_FOUND", Message: "Unknown fleet plan action."})
}
func (s *Server) previewV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PlanActionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.Preview(r.PathValue("id"), r.PathValue("plan_id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) authorizeV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PlanActionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.Authorize(r.PathValue("id"), r.PathValue("plan_id"), req)
	respondV2(w, v, err, http.StatusCreated)
}
func (s *Server) startV2(w http.ResponseWriter, r *http.Request) {
	var req fleetops.PlanActionRequest
	if !decode(w, r, &req) {
		return
	}
	v, err := s.fleetops.Start(r.PathValue("id"), r.PathValue("plan_id"), req)
	respondV2(w, v, err, http.StatusOK)
}
func (s *Server) voicesV2(w http.ResponseWriter, r *http.Request) {
	if s.speechURL != "" {
		request, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, s.speechURL+"/v1/voices", nil)
		if response, err := http.DefaultClient.Do(request); err == nil {
			defer response.Body.Close()
			if response.StatusCode == http.StatusOK {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.Copy(w, response.Body)
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"voices": s.fleetops.Voices(), "degraded": true})
}
func (s *Server) synthesizeV2(w http.ResponseWriter, r *http.Request) {
	if s.speechURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "TTS_UNAVAILABLE", Message: "The node-local Pocket TTS service is not configured; visible text remains available."})
		return
	}
	var payload struct {
		Text      string `json:"text"`
		Voice     string `json:"voice"`
		RequestID string `json:"request_id"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if payload.Voice == "" {
		payload.Voice = "jarvis"
	}
	if payload.RequestID == "" || len(payload.Text) > 1200 {
		writeJSON(w, http.StatusUnprocessableEntity, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: "request_id and text up to 1200 characters are required."})
		return
	}
	body, _ := json.Marshal(payload)
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.speechURL+"/v1/synthesize", bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "TTS_UNAVAILABLE", Message: "Speech request could not be created."})
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "TTS_UNAVAILABLE", Message: "Pocket TTS node is unavailable; use visible text."})
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "TTS_UNAVAILABLE", Message: "Pocket TTS could not synthesize this utterance."})
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-KeelMesh-Request-ID", payload.RequestID)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, response.Body)
}
func (s *Server) speechCapabilitiesV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.fleetops.SpeechCapabilities())
}
func (s *Server) transcriptionV2(w http.ResponseWriter, r *http.Request) {
	if s.speechURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "STT_UNAVAILABLE", Message: "No node-local transcription route is configured; typed input remains available."})
		return
	}
	audio, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8*1024*1024))
	if err != nil || len(audio) < 128 {
		writeJSON(w, http.StatusUnprocessableEntity, domain.APIError{Code: "AUDIO_SIZE_INVALID", Message: "A bounded audio recording is required."})
		return
	}
	body, status, err := s.transcribeAudio(r.Context(), audio, r.Header.Get("Content-Type"), r.URL.Query().Get("request_id"))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "STT_UNAVAILABLE", Message: "Node transcription failed; use typed input."})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
func (s *Server) transcriptionStreamV2(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{r.Host}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(8 * 1024 * 1024)
	for {
		kind, audio, readErr := c.Read(r.Context())
		if readErr != nil {
			return
		}
		if kind != websocket.MessageBinary {
			failure, _ := json.Marshal(domain.APIError{Code: "AUDIO_REQUIRED", Message: "Send one recorded utterance as a binary WebM frame."})
			_ = c.Write(r.Context(), websocket.MessageText, failure)
			continue
		}
		body, _, transcribeErr := s.transcribeAudio(r.Context(), audio, "audio/webm", "websocket")
		if transcribeErr != nil {
			body, _ = json.Marshal(domain.APIError{Code: "STT_UNAVAILABLE", Message: "Node transcription failed; use typed input."})
		}
		if writeErr := c.Write(r.Context(), websocket.MessageText, body); writeErr != nil {
			return
		}
	}
}
func (s *Server) transcribeAudio(ctx context.Context, audio []byte, contentType, requestID string) ([]byte, int, error) {
	if contentType == "" {
		contentType = "audio/webm"
	}
	endpoint := s.speechURL + "/v1/transcribe?request_id=" + url.QueryEscape(requestID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(audio))
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Content-Type", contentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if err != nil || response.StatusCode != http.StatusOK {
		return body, response.StatusCode, &fleetops.Error{Code: "STT_UNAVAILABLE", Message: "Speech node rejected transcription."}
	}
	return body, response.StatusCode, nil
}
func (s *Server) inferenceRoutesV2(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"routes": []map[string]any{{"id": "cloud-openai", "model": "gpt-5.6-luna", "state": "available_when_connected", "priority": 1}, {"id": "node-provider-service", "state": "preferred_when_available", "priority": 2}, {"id": "deterministic-target-aware", "state": "available", "priority": 3}}, "authority": "advisory_only", "physical_gpu_claim": false})
}

func respondV2(w http.ResponseWriter, v any, err error, status int) {
	if err == nil {
		writeJSON(w, status, v)
		return
	}
	if typed, ok := err.(*fleetops.Error); ok {
		code := http.StatusConflict
		if typed.Code == "VESSEL_NOT_FOUND" || typed.Code == "MISSION_NOT_FOUND" || typed.Code == "PLAN_NOT_FOUND" || typed.Code == "GROUP_NOT_FOUND" || typed.Code == "COLLECTION_NOT_FOUND" {
			code = http.StatusNotFound
		}
		writeJSON(w, code, domain.APIError{Code: typed.Code, Message: typed.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "INTERNAL", Message: err.Error()})
}
