package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/coordination"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (s *Server) coordinationMutationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.coordGateway == nil || s.coordGateway.Mode() == coordination.ModeSimulated || !requiresCoordination(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: "Unable to read coordinated mutation."})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		metadata := map[string]any{}
		if len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &metadata) != nil {
			next.ServeHTTP(w, r)
			return
		}
		requestID := stringField(metadata, "request_id")
		idempotencyKey := stringField(metadata, "idempotency_key")
		if requestID == "" || idempotencyKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		actor := stringField(metadata, "actor_identity")
		if actor == "" {
			actor = stringField(metadata, "operator_id")
		}
		if actor == "" {
			actor = "demo-operator"
		}
		expectedVersion := int64Field(metadata, "expected_version")
		cells := s.mutationCells(r, metadata)
		payload := map[string]any{"method": r.Method, "path": r.URL.Path, "body": json.RawMessage(body)}
		digest := sha256.Sum256([]byte(r.Method + "\n" + r.URL.Path + "\n" + string(body)))
		baseCommandID := "http-" + hex.EncodeToString(digest[:12])
		kind := mutationKind(r.Method, r.URL.Path)
		entityID := mutationEntityID(r.URL.Path)
		var coordinationErr error
		proofIDs := make([]string, 0, len(cells))
		if len(cells) > 1 {
			activation := time.Now().UTC().Add(1500 * time.Millisecond).Truncate(time.Second).Add(time.Second)
			operationID := baseCommandID + "-cross"
			ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
			operation, crossErr := s.coordGateway.CrossCell(ctx, operationID, requestID, idempotencyKey, actor, kind, entityID, expectedVersion, payload, activation)
			cancel()
			if crossErr != nil {
				coordinationErr = crossErr
			} else {
				proofIDs = append(proofIDs, operation.ID)
				if delay := time.Until(operation.ActivationAt); delay > 0 {
					timer := time.NewTimer(delay)
					select {
					case <-r.Context().Done():
						timer.Stop()
						coordinationErr = r.Context().Err()
					case <-timer.C:
					}
				}
			}
		}
		for _, cellID := range cells {
			if len(cells) > 1 {
				break
			}
			commandID := baseCommandID + "-" + strings.ToLower(cellID)
			command, commandErr := s.coordGateway.CanonicalCommand(cellID, commandID, requestID, idempotencyKey+":"+strings.ToLower(cellID), actor, kind, entityID, expectedVersion, payload, nil)
			if commandErr != nil {
				coordinationErr = commandErr
				break
			}
			ctx, cancel := context.WithTimeout(r.Context(), 9*time.Second)
			_, proof, commitErr := s.coordGateway.Commit(ctx, command)
			cancel()
			if commitErr != nil {
				coordinationErr = commitErr
				break
			}
			if acceptErr := s.coordGateway.AcceptEffect(proof); acceptErr != nil {
				coordinationErr = acceptErr
				break
			}
			proofIDs = append(proofIDs, proof.CommandID)
		}
		if coordinationErr != nil {
			if s.coordGateway.Mode() == coordination.ModeRaft {
				writeCoordinationPublicError(w, coordinationErr)
				return
			}
			s.logger.Warn("shadow coordination comparison failed", "method", r.Method, "path", r.URL.Path, "error", coordinationErr)
			w.Header().Set("X-KeelMesh-Coordination-State", "shadow-diverged")
		} else {
			w.Header().Set("X-KeelMesh-Coordination-State", string(s.coordGateway.Mode())+"-committed")
			w.Header().Set("X-KeelMesh-Coordination-Proofs", strings.Join(proofIDs, ","))
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

func requiresCoordination(method, path string) bool {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return false
	}
	if strings.HasPrefix(path, "/api/v2/missions") || strings.HasPrefix(path, "/api/v2/groups") {
		return true
	}
	return strings.HasPrefix(path, "/api/v3/matches")
}

func (s *Server) mutationCells(r *http.Request, metadata map[string]any) []string {
	targets := stringSliceField(metadata, "target_ids")
	targets = append(targets, stringSliceField(metadata, "member_ids")...)
	if vesselID := stringField(metadata, "vessel_id"); vesselID != "" {
		targets = append(targets, vesselID)
	}
	pathID := mutationEntityID(r.URL.Path)
	if s.fleetops != nil && pathID != "" {
		snapshot := s.fleetops.Snapshot()
		for _, mission := range snapshot.Missions {
			if mission.ID == pathID {
				targets = append(targets, mission.TargetIDs...)
			}
		}
		for _, group := range snapshot.Groups {
			if group.ID == pathID {
				targets = append(targets, group.MemberIDs...)
			}
		}
	}
	set := map[string]bool{}
	if s.fleetops != nil {
		for _, vessel := range s.fleetops.Snapshot().Vessels {
			for _, target := range targets {
				if vessel.ID == target && (vessel.NodeFaction == "A" || vessel.NodeFaction == "B") {
					set[vessel.NodeFaction] = true
				}
			}
		}
	}
	if len(set) == 0 {
		faction := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-KeelMesh-Faction")))
		if faction != "A" && faction != "B" {
			faction = "A"
		}
		set[faction] = true
	}
	result := make([]string, 0, len(set))
	for cellID := range set {
		result = append(result, cellID)
	}
	sort.Strings(result)
	return result
}

func mutationKind(method, path string) string {
	resource := "authority"
	if strings.Contains(path, "/missions") {
		resource = "mission"
	} else if strings.Contains(path, "/groups") {
		resource = "group"
	} else if strings.Contains(path, "/matches") {
		resource = "arena"
	}
	action := strings.ToLower(method)
	if index := strings.LastIndex(path, ":"); index >= 0 {
		action = path[index+1:]
	} else if method == http.MethodDelete {
		action = "delete"
	} else if method == http.MethodPatch {
		action = "revise"
	} else if method == http.MethodPost {
		action = "create"
	}
	return resource + "." + action
}

func mutationEntityID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if (part == "missions" || part == "groups" || part == "matches") && index+1 < len(parts) {
			return strings.Split(parts[index+1], ":")[0]
		}
	}
	return ""
}

func stringField(values map[string]any, name string) string {
	value, _ := values[name].(string)
	return strings.TrimSpace(value)
}

func int64Field(values map[string]any, name string) int64 {
	switch value := values[name].(type) {
	case float64:
		return int64(value)
	case string:
		parsed, _ := strconv.ParseInt(value, 10, 64)
		return parsed
	default:
		return 0
	}
}

func stringSliceField(values map[string]any, name string) []string {
	raw, _ := values[name].([]any)
	result := make([]string, 0, len(raw))
	for _, value := range raw {
		if text, ok := value.(string); ok && text != "" {
			result = append(result, text)
		}
	}
	return result
}
