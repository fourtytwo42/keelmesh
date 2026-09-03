package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

var sceneTypes = map[string]bool{
	"operational_brief": true, "decision_board": true, "comparison_matrix": true,
	"evidence_chain": true, "mission_canvas": true, "simulation_sandbox": true,
	"status_matrix": true, "approval_card": true,
}

func (m *Manager) CreateAssistantTurn(ctx context.Context, request domain.AssistantTurnRequestV2, fleet domain.FleetSnapshotV2) (domain.AssistantTurnV2, error) {
	if strings.TrimSpace(request.ActorIdentity) == "" {
		request.ActorIdentity = "demo-operator"
	}
	if strings.TrimSpace(request.SessionID) == "" {
		request.SessionID = "default"
	}
	if request.WorkspaceVersion != 0 && request.WorkspaceVersion != fleet.FleetVersion {
		return domain.AssistantTurnV2{}, problem("SCENE_STALE", "Workspace state changed; refresh before composing another command scene.")
	}
	fingerprint := hashCanonical(request.ActorIdentity, request.SessionID, request.Text, strings.Join(request.SelectedIDs, ","), request.ActiveMissionID, fmt.Sprint(request.WorkspaceVersion))
	lookupKey := "scene-turn:" + request.ActorIdentity + ":" + request.SessionID + ":" + request.IdempotencyKey
	if request.IdempotencyKey != "" {
		m.mu.RLock()
		prior := m.idempotency[lookupKey]
		if parts := strings.SplitN(prior, "|", 2); len(parts) == 2 {
			turn, ok := m.turns[parts[1]]
			m.mu.RUnlock()
			if parts[0] != fingerprint {
				return domain.AssistantTurnV2{}, problem("ACTION_STALE", "Idempotency key was already used for a different assistant turn.")
			}
			if ok {
				return clone(turn), nil
			}
		} else {
			m.mu.RUnlock()
		}
	}
	now := time.Now().UTC()
	turnID := stableSceneID("turn", request.RequestID, request.IdempotencyKey, now.Format(time.RFC3339Nano))
	stages := []string{"accepted", "gathering", "composing", "validating", "rendering"}
	assistant, err := m.WorkspaceCommand(ctx, request.WorkspaceAssistantRequestV1, fleet)
	if err != nil {
		return domain.AssistantTurnV2{}, err
	}
	scene := composeCommandScene(request, assistant, fleet, now)
	stages = append(stages, "speaking")
	state := "completed"
	if scene.PendingApproval {
		state = "awaiting_decision"
	}
	turn := domain.AssistantTurnV2{SchemaVersion: 2, ID: turnID, ActorID: request.ActorIdentity, SessionID: request.SessionID, State: state, Stages: stages, Scene: scene, Assistant: assistant, CreatedAt: now, CompletedAt: time.Now().UTC()}

	m.mu.Lock()
	activeKey := sceneSessionKey(request.ActorIdentity, request.SessionID)
	if previousID := m.activeScenes[activeKey]; previousID != "" {
		previous := m.scenes[previousID]
		if !previous.Pinned && !previous.Critical && !previous.PendingApproval {
			previous.State, previous.UpdatedAt = "replaced", now
			m.scenes[previousID] = previous
			m.appendSceneEventLocked("a2ui.message", previous.ID, turnID, map[string]any{
				"version":       "v1.0",
				"deleteSurface": map[string]any{"surfaceId": previous.PrimarySurface.ID},
			})
			m.appendSceneEventLocked("scene.deleted", previous.ID, turnID, map[string]any{"reason": "replaced"})
		}
	}
	m.scenes[scene.ID] = scene
	m.sceneOrder = append(m.sceneOrder, scene.ID)
	if len(m.sceneOrder) > 50 {
		old := m.sceneOrder[0]
		m.sceneOrder = m.sceneOrder[1:]
		if value := m.scenes[old]; !value.Pinned && value.ID != m.activeScenes[sceneSessionKey(value.ActorID, value.SessionID)] {
			delete(m.scenes, old)
		}
	}
	m.activeScenes[activeKey] = scene.ID
	m.turns[turn.ID] = turn
	if request.IdempotencyKey != "" {
		m.idempotency[lookupKey] = fingerprint + "|" + turn.ID
	}
	for _, stage := range stages {
		m.appendSceneEventLocked("turn.stage", scene.ID, turnID, map[string]any{"stage": stage})
	}
	for _, surface := range append([]domain.WorkspaceSurfaceV1{scene.PrimarySurface}, scene.Supporting...) {
		for _, message := range surface.Messages {
			m.appendSceneEventLocked("a2ui.message", scene.ID, turnID, message)
		}
	}
	m.appendSceneEventLocked("scene.ready", scene.ID, turnID, scene)
	m.mu.Unlock()
	return turn, nil
}

// ResetCommandScenes clears transient command-scene state for an operator when
// the explicit Fleet Operations scenario reset is requested. Pinned scenes are
// retained unless includePinned is true; an ordinary browser reload does not
// call this method, so pinned and active scenes still survive reload.
func (m *Manager) ResetCommandScenes(actor string, includePinned bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept := make([]string, 0, len(m.sceneOrder))
	removedScenes := map[string]bool{}
	for _, id := range m.sceneOrder {
		scene, ok := m.scenes[id]
		if !ok {
			continue
		}
		if scene.ActorID == actor && (includePinned || !scene.Pinned) {
			removedScenes[id] = true
			delete(m.scenes, id)
			delete(m.activeScenes, sceneSessionKey(scene.ActorID, scene.SessionID))
			continue
		}
		kept = append(kept, id)
	}
	m.sceneOrder = kept
	for id, turn := range m.turns {
		if turn.ActorID == actor && removedScenes[turn.Scene.ID] {
			delete(m.turns, id)
		}
	}
	filteredEvents := m.sceneEvents[:0]
	for _, event := range m.sceneEvents {
		if !removedScenes[event.SceneID] {
			filteredEvents = append(filteredEvents, event)
		}
	}
	m.sceneEvents = filteredEvents
	for key := range m.idempotency {
		if strings.HasPrefix(key, "scene-turn:"+actor+":") {
			delete(m.idempotency, key)
		}
	}
	for key := range m.proactiveSeen {
		delete(m.proactiveSeen, key)
	}
}

func composeCommandScene(request domain.AssistantTurnRequestV2, assistant domain.WorkspaceAssistantResponseV1, fleet domain.FleetSnapshotV2, now time.Time) domain.CommandSceneV1 {
	intent := sceneIntent(request, assistant, fleet)
	sceneID := stableSceneID("scene", request.ActorIdentity, request.RequestID, intent.Type, now.Format(time.RFC3339Nano))
	entities := resolveSceneEntities(request, assistant, fleet)
	data := map[string]any{"eyebrow": strings.ReplaceAll(strings.ToUpper(intent.Type), "_", " "), "title": intent.Title, "summary": intent.Summary, "entities": entities, "provider": assistant.Provider, "model": assistant.Model, "state": "LIVE"}
	components := []map[string]any{
		{"id": "root", "component": "Column", "children": []string{"eyebrow", "title", "summary", "divider", "entities", "actions"}},
		{"id": "eyebrow", "component": "Text", "text": map[string]any{"path": "/eyebrow"}},
		{"id": "title", "component": "Text", "text": map[string]any{"path": "/title"}},
		{"id": "summary", "component": "Text", "text": map[string]any{"path": "/summary"}},
		{"id": "divider", "component": "Divider"},
		{"id": "entities", "component": "Text", "text": entitySummary(entities)},
		{"id": "actions", "component": "Text", "text": actionSummary(intent.SuggestedActions)},
	}
	surface := domain.WorkspaceSurfaceV1{ID: sceneID + "-primary", Role: "primary", Title: intent.Title, Sequence: 3}
	surface.Messages = []map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": surface.ID, "catalogId": "https://keelmesh.local/catalogs/" + domain.KeelMeshOperationsCatalogV1}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": surface.ID, "components": components}},
		{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": surface.ID, "path": "/", "value": data}},
	}
	actions := make([]domain.ArtifactActionV1, 0, len(intent.SuggestedActions))
	for index, label := range intent.SuggestedActions {
		action := domain.ArtifactActionV1{ID: fmt.Sprintf("%s-action-%d", sceneID, index+1), Kind: sceneActionKind(label), Label: label, AuthorityClass: "presentation", Payload: map[string]any{}}
		action.ActionHash = hashCanonical(action.Kind, action.Label, fmt.Sprint(fleet.FleetVersion))
		actions = append(actions, action)
	}
	annotations, camera := sceneMapContext(entities, fleet, sceneID)
	receipt := domain.SceneReceiptV1{ID: sceneID + "-receipt", Kind: "scene_composition", State: "accepted", Detail: fmt.Sprintf("Trusted catalog %s composed from %d visible entities.", domain.KeelMeshOperationsCatalogV1, len(entities)), CreatedAt: now}
	return domain.CommandSceneV1{SchemaVersion: 1, ID: sceneID, ActorID: request.ActorIdentity, SessionID: request.SessionID, Type: intent.Type, Title: intent.Title, Summary: intent.Summary, State: "active", WorkspaceVersion: fleet.FleetVersion, CatalogID: domain.KeelMeshOperationsCatalogV1, PrimarySurface: surface, Supporting: []domain.WorkspaceSurfaceV1{}, Bindings: sceneBindings(entities), MapCamera: camera, MapAnnotations: annotations, SpokenSummary: assistant.Speech, SuggestedActions: actions, Receipts: []domain.SceneReceiptV1{receipt}, CreatedAt: now, UpdatedAt: now}
}

func sceneIntent(request domain.AssistantTurnRequestV2, response domain.WorkspaceAssistantResponseV1, fleet domain.FleetSnapshotV2) domain.SceneIntentV1 {
	lower := strings.ToLower(request.Text)
	typeName, title := "operational_brief", "Operational brief"
	if response.Mode == "mission" {
		typeName, title = "mission_canvas", "Mission Canvas"
	} else if len(request.PlanOptions) > 0 || strings.Contains(lower, "options") || strings.Contains(lower, "alternatives") {
		typeName, title = "decision_board", "Decision Board"
	} else if strings.Contains(lower, "compare") {
		typeName, title = "comparison_matrix", "Comparison Matrix"
	} else if strings.Contains(lower, "why") || strings.Contains(lower, "evidence") || strings.Contains(lower, "explain") {
		typeName, title = "evidence_chain", "Evidence Chain"
	} else if strings.Contains(lower, "simulate") || strings.Contains(lower, "what if") {
		typeName, title = "simulation_sandbox", "Simulation Sandbox"
	} else if strings.Contains(lower, "status") || strings.Contains(lower, "list") || strings.Contains(lower, "all ") {
		typeName, title = "status_matrix", "Status Matrix"
	}
	if !sceneTypes[typeName] {
		typeName = "operational_brief"
	}
	actions := []string{"Frame on map", "Keep this", "Dismiss"}
	if typeName == "mission_canvas" {
		actions = []string{"Open Mission Canvas", "Review options", "Edit mission"}
	}
	return domain.SceneIntentV1{Type: typeName, Title: title, Summary: response.Speech, EntityIDs: entityIDs(resolveSceneEntities(request, response, fleet)), MissionID: request.ActiveMissionID, SuggestedActions: actions}
}

func resolveSceneEntities(request domain.AssistantTurnRequestV2, response domain.WorkspaceAssistantResponseV1, fleet domain.FleetSnapshotV2) []map[string]any {
	seen, values := map[string]bool{}, []map[string]any{}
	addVessel := func(v domain.VesselProfileV2) {
		if seen[v.ID] {
			return
		}
		seen[v.ID] = true
		values = append(values, map[string]any{"id": v.ID, "type": "vessel", "name": v.DisplayName, "status": v.Telemetry.Mode, "reserve": v.Telemetry.Reserve, "position": v.Telemetry.Position, "group": v.GroupCode})
	}
	addGroup := func(g domain.OperationalGroupV2) {
		if seen[g.ID] {
			return
		}
		seen[g.ID] = true
		value := map[string]any{"id": g.ID, "type": "group", "name": g.Code + " · " + g.Name, "status": g.Formation, "color": g.ColorName, "members": len(g.MemberIDs)}
		if g.AssemblyPoint != nil {
			value["position"] = *g.AssemblyPoint
		}
		values = append(values, value)
	}
	addContact := func(c domain.SurfaceContactV2) {
		if seen[c.ID] {
			return
		}
		seen[c.ID] = true
		values = append(values, map[string]any{"id": c.ID, "type": "contact", "name": c.Name, "status": c.Activity, "speed_mps": c.SpeedMPS, "position": c.Position, "class": c.Class})
	}
	for _, id := range request.SelectedIDs {
		for _, v := range fleet.Vessels {
			if v.ID == id {
				addVessel(v)
			}
		}
	}
	text := strings.ToLower(request.Text + " " + response.Speech)
	for _, g := range fleet.Groups {
		if strings.Contains(text, strings.ToLower(g.Name)) || strings.Contains(text, strings.ToLower(g.Code+" group")) || strings.Contains(text, strings.ToLower(g.ColorName+" group")) {
			addGroup(g)
		}
	}
	for _, v := range fleet.Vessels {
		if strings.Contains(text, strings.ToLower(v.Callsign)) || strings.Contains(text, strings.ToLower(v.Designation)) {
			addVessel(v)
		}
	}
	for _, c := range fleet.SurfaceContacts {
		if strings.Contains(text, strings.ToLower(c.Name)) || strings.Contains(text, strings.ToLower(c.BoatID)) {
			addContact(c)
		}
	}
	if len(values) > 12 {
		values = values[:12]
	}
	return values
}

func entityIDs(values []map[string]any) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if id, ok := v["id"].(string); ok {
			result = append(result, id)
		}
	}
	return result
}
func entitySummary(values []map[string]any) string {
	if len(values) == 0 {
		return "No entity was required for this answer."
	}
	names := []string{}
	for _, v := range values {
		names = append(names, fmt.Sprint(v["name"]))
	}
	return strings.Join(names, " · ")
}
func actionSummary(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return "Available: " + strings.Join(values, " · ")
}
func sceneActionKind(label string) string {
	lower := strings.ToLower(label)
	if strings.Contains(lower, "frame") {
		return "frame_entities"
	}
	if strings.Contains(lower, "keep") {
		return "pin_scene"
	}
	if strings.Contains(lower, "dismiss") {
		return "dismiss_scene"
	}
	if strings.Contains(lower, "mission") {
		return "open_window"
	}
	if strings.Contains(lower, "edit") {
		return "open_edit_drawer"
	}
	return "focus_surface"
}
func sceneBindings(values []map[string]any) []domain.ArtifactDataBindingV1 {
	result := []domain.ArtifactDataBindingV1{}
	for _, v := range values {
		id := fmt.Sprint(v["id"])
		result = append(result, domain.ArtifactDataBindingV1{ID: "binding-" + id, EntityType: fmt.Sprint(v["type"]), EntityID: id, Field: "live", Path: "/entities/" + id})
	}
	return result
}

func sceneMapContext(values []map[string]any, fleet domain.FleetSnapshotV2, sceneID string) ([]domain.SceneMapAnnotationV1, *domain.MapCameraInstructionV1) {
	points := []domain.GeoPointV2{}
	annotations := []domain.SceneMapAnnotationV1{}
	for index, value := range values {
		raw, ok := value["position"]
		if !ok {
			continue
		}
		point, ok := raw.(domain.GeoPointV2)
		if !ok {
			continue
		}
		points = append(points, point)
		annotations = append(annotations, domain.SceneMapAnnotationV1{ID: fmt.Sprintf("%s-callout-%d", sceneID, index+1), Kind: "callout", Label: fmt.Sprintf("%d · %s", index+1, value["name"]), Color: "#e9a936", Points: []domain.GeoPointV2{point}, EntityID: fmt.Sprint(value["id"])})
	}
	if len(points) == 0 {
		return annotations, nil
	}
	center := domain.GeoPointV2{}
	for _, p := range points {
		center[0] += p[0]
		center[1] += p[1]
	}
	center[0] /= float64(len(points))
	center[1] /= float64(len(points))
	return annotations, &domain.MapCameraInstructionV1{Center: center, Zoom: 11, Bearing: -8, Pitch: 28}
}

func stableSceneID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return parts[0] + "-" + hex.EncodeToString(sum[:8])
}
func hashCanonical(parts ...string) string {
	raw, _ := json.Marshal(parts)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (m *Manager) appendSceneEventLocked(kind, sceneID, turnID string, payload any) {
	event := domain.SceneEventV1{ID: m.nextSceneEvent, Kind: kind, SceneID: sceneID, TurnID: turnID, Payload: payload, CreatedAt: time.Now().UTC()}
	m.nextSceneEvent++
	m.sceneEvents = append(m.sceneEvents, event)
	if len(m.sceneEvents) > 2000 {
		m.sceneEvents = m.sceneEvents[len(m.sceneEvents)-2000:]
	}
}

func (m *Manager) Scenes(actor, session string) []domain.CommandSceneV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []domain.CommandSceneV1{}
	for _, id := range m.sceneOrder {
		value := m.scenes[id]
		if (actor == "" || value.ActorID == actor) && (session == "" || value.SessionID == session || value.Critical) {
			result = append(result, clone(value))
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}
func (m *Manager) Scene(id string) (domain.CommandSceneV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.scenes[id]
	if !ok {
		return value, problem("SURFACE_NOT_FOUND", "Command scene was not found.")
	}
	return clone(value), nil
}

func (m *Manager) SceneForSession(id, actor, session string) (domain.CommandSceneV1, error) {
	value, err := m.Scene(id)
	if err != nil {
		return value, err
	}
	if actor == "" || session == "" || value.ActorID != actor || (value.SessionID != session && !value.Critical) {
		return domain.CommandSceneV1{}, problem("ENTITY_NOT_VISIBLE", "Command scene is not visible to this operator session.")
	}
	return value, nil
}
func (m *Manager) Turn(id string) (domain.AssistantTurnV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.turns[id]
	if !ok {
		return value, problem("SURFACE_NOT_FOUND", "Assistant turn was not found.")
	}
	return clone(value), nil
}
func (m *Manager) SceneEvents(turnID string, after int64) []domain.SceneEventV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []domain.SceneEventV1{}
	for _, event := range m.sceneEvents {
		if event.ID > after && (turnID == "" || event.TurnID == turnID) {
			result = append(result, clone(event))
		}
	}
	return result
}

func (m *Manager) MutateScene(id, operation string, request domain.SceneMutationV1) (domain.CommandSceneV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.scenes[id]
	if !ok {
		return value, problem("SURFACE_NOT_FOUND", "Command scene was not found.")
	}
	if request.ActorIdentity == "" || request.ActorIdentity != value.ActorID {
		return value, problem("ACTION_NOT_PERMITTED", "Scene belongs to another operator session.")
	}
	if request.SessionID != "" && request.SessionID != value.SessionID {
		return value, problem("ACTION_NOT_PERMITTED", "Scene belongs to another operator session.")
	}
	if request.WorkspaceVersion != 0 && request.WorkspaceVersion != value.WorkspaceVersion {
		return value, problem("SCENE_STALE", "Scene was composed against a different workspace version.")
	}
	switch operation {
	case "pin":
		count := 0
		for _, v := range m.scenes {
			if v.ActorID == value.ActorID && v.Pinned {
				count++
			}
		}
		if !value.Pinned && count >= 4 {
			return value, problem("SCENE_LIMIT_REACHED", "At most four scenes may be pinned.")
		}
		value.Pinned = true
	case "unpin":
		value.Pinned = false
	case "dismiss":
		if value.PendingApproval {
			return value, problem("APPROVAL_REQUIRED", "Resolve the pending approval before dismissing this scene.")
		}
		value.State = "dismissed"
		key := sceneSessionKey(value.ActorID, value.SessionID)
		if m.activeScenes[key] == id {
			delete(m.activeScenes, key)
		}
	default:
		return value, problem("ACTION_NOT_PERMITTED", "Unknown scene operation.")
	}
	value.UpdatedAt = time.Now().UTC()
	m.scenes[id] = value
	m.appendSceneEventLocked("scene."+operation, id, "", value)
	return clone(value), nil
}

func (m *Manager) ApplySceneAction(id string, request domain.SceneActionRequestV1) (domain.CommandSceneV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	scene, ok := m.scenes[id]
	if !ok {
		return scene, problem("SURFACE_NOT_FOUND", "Command scene was not found.")
	}
	if request.ActorIdentity != scene.ActorID {
		return scene, problem("ACTION_NOT_PERMITTED", "Scene belongs to another operator session.")
	}
	if request.SessionID != "" && request.SessionID != scene.SessionID {
		return scene, problem("ACTION_NOT_PERMITTED", "Scene belongs to another operator session.")
	}
	if request.WorkspaceVersion != 0 && request.WorkspaceVersion != scene.WorkspaceVersion {
		return scene, problem("ACTION_STALE", "Action was composed against stale workspace state.")
	}
	var action *domain.ArtifactActionV1
	for index := range scene.SuggestedActions {
		if scene.SuggestedActions[index].ID == request.ActionID {
			action = &scene.SuggestedActions[index]
			break
		}
	}
	if action == nil {
		return scene, problem("ACTION_STALE", "Artifact action is not part of this scene.")
	}
	if action.AuthorityClass == "effect" && (!request.Confirmed || request.ActionHash != action.ActionHash) {
		return scene, problem("APPROVAL_REQUIRED", "This action requires exact operator confirmation.")
	}
	if action.Kind == "pin_scene" {
		scene.Pinned = true
	}
	if action.Kind == "dismiss_scene" {
		scene.State = "dismissed"
	}
	scene.Receipts = append(scene.Receipts, domain.SceneReceiptV1{ID: stableSceneID("receipt", request.RequestID, request.ActionID), Kind: action.Kind, State: "accepted", Detail: "Action validated against the trusted catalog and current scene version.", CreatedAt: time.Now().UTC()})
	scene.UpdatedAt = time.Now().UTC()
	m.scenes[id] = scene
	m.appendSceneEventLocked("scene.action", id, "", scene.Receipts[len(scene.Receipts)-1])
	return clone(scene), nil
}

// RefreshScenes projects current operator-visible state into existing trusted
// bindings. It never invokes a model and never changes mission authority.
func (m *Manager) RefreshScenes(fleet domain.FleetSnapshotV2) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, scene := range m.scenes {
		if scene.State != "active" && !scene.Pinned {
			continue
		}
		entities := []map[string]any{}
		for _, binding := range scene.Bindings {
			if value, ok := liveSceneEntity(binding.EntityID, fleet); ok {
				entities = append(entities, value)
			}
		}
		if len(scene.PrimarySurface.Messages) >= 3 {
			if update, ok := scene.PrimarySurface.Messages[2]["updateDataModel"].(map[string]any); ok {
				if data, ok := update["value"].(map[string]any); ok {
					data["entities"] = entities
					data["state"] = "LIVE"
				}
			}
		}
		scene.MapAnnotations, scene.MapCamera = sceneMapContext(entities, fleet, scene.ID)
		scene.WorkspaceVersion = fleet.FleetVersion
		scene.PrimarySurface.Sequence++
		scene.UpdatedAt = time.Now().UTC()
		m.scenes[id] = scene
	}
	m.evaluateProactiveLocked(fleet)
}

func liveSceneEntity(id string, fleet domain.FleetSnapshotV2) (map[string]any, bool) {
	for _, v := range fleet.Vessels {
		if v.ID == id {
			return map[string]any{"id": v.ID, "type": "vessel", "name": v.DisplayName, "status": v.Telemetry.Mode, "reserve": v.Telemetry.Reserve, "position": v.Telemetry.Position, "group": v.GroupCode, "pnt": v.Telemetry.PNTIntegrity, "uncertainty_m": v.Telemetry.UncertaintyM, "tape_seconds": v.Telemetry.TapeDepthSeconds}, true
		}
	}
	for _, g := range fleet.Groups {
		if g.ID == id {
			value := map[string]any{"id": g.ID, "type": "group", "name": g.Code + " · " + g.Name, "status": g.Formation, "color": g.ColorName, "members": len(g.MemberIDs)}
			if g.AssemblyPoint != nil {
				value["position"] = *g.AssemblyPoint
			}
			return value, true
		}
	}
	for _, c := range fleet.SurfaceContacts {
		if c.ID == id {
			return map[string]any{"id": c.ID, "type": "contact", "name": c.Name, "status": c.Activity, "speed_mps": c.SpeedMPS, "position": c.Position, "class": c.Class}, true
		}
	}
	return nil, false
}

func (m *Manager) evaluateProactiveLocked(fleet domain.FleetSnapshotV2) {
	for _, vessel := range fleet.Vessels {
		condition, detail := "", ""
		if vessel.Telemetry.PNTIntegrity == "unsafe" || vessel.Telemetry.UncertaintyM > 45 {
			condition, detail = "unsafe_pnt", fmt.Sprintf("%s PNT uncertainty is %.0f m; mission motion requires operator review.", vessel.DisplayName, vessel.Telemetry.UncertaintyM)
		} else if vessel.Telemetry.TapeDepthSeconds > 0 && vessel.Telemetry.TapeDepthSeconds <= 14 {
			condition, detail = "critical_tape", fmt.Sprintf("%s has %d seconds of cached authority remaining.", vessel.DisplayName, vessel.Telemetry.TapeDepthSeconds)
		} else if vessel.Telemetry.Reserve <= .2 {
			condition, detail = "reserve_threshold", fmt.Sprintf("%s reserve crossed the 20 percent critical threshold.", vessel.DisplayName)
		}
		key := vessel.ID + ":" + condition
		if condition == "" {
			continue
		}
		transition := fmt.Sprintf("%s:%d", condition, fleet.SimulationTick/1000)
		if m.proactiveSeen[key] != "" {
			continue
		}
		m.proactiveSeen[key] = transition
		now := time.Now().UTC()
		sceneID := stableSceneID("critical", key, transition)
		entities := []map[string]any{}
		if value, ok := liveSceneEntity(vessel.ID, fleet); ok {
			entities = append(entities, value)
		}
		request := domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: sceneID, Text: detail, SelectedIDs: []string{vessel.ID}}, ActorIdentity: "demo-operator", SessionID: "system-critical", WorkspaceVersion: fleet.FleetVersion}
		assistant := domain.WorkspaceAssistantResponseV1{SchemaVersion: 1, Mode: "conversation", Speech: detail, Provider: "deterministic", Model: "critical-trigger-v1"}
		scene := composeCommandScene(request, assistant, fleet, now)
		scene.ID, scene.Type, scene.Title, scene.Critical = sceneID, "status_matrix", "Critical operational transition", true
		scene.Summary, scene.SpokenSummary = detail, detail
		scene.MapAnnotations, scene.MapCamera = sceneMapContext(entities, fleet, sceneID)
		m.scenes[scene.ID] = scene
		m.sceneOrder = append(m.sceneOrder, scene.ID)
		m.activeScenes[sceneSessionKey(scene.ActorID, scene.SessionID)] = scene.ID
		m.appendSceneEventLocked("proactive.critical", scene.ID, "", domain.ProactiveSceneTriggerV1{ID: key, EntityID: vessel.ID, Condition: condition, Transition: transition, Severity: "critical", DetectedAt: now})
	}
}

func sceneSessionKey(actor, session string) string { return actor + "|" + session }

func SceneCatalog() domain.SceneCatalogV1 {
	return domain.SceneCatalogV1{ID: domain.KeelMeshOperationsCatalogV1, ProtocolVersion: "1.0 (ordered createSurface/updateComponents/updateDataModel profile)", RendererVersion: "@a2ui/react@0.11.0", AllowedComponents: []string{"EntityChip", "StatusChip", "MetricStrip", "Sparkline", "PlanOptionCard", "FormationControl", "ConstraintControl", "EvidenceEvent", "Citation", "MissionSummary", "MissionProgress", "ApprovalSummary", "MapFocusAction", "RoutePreviewAction"}, DeniedContent: []string{"html", "javascript", "css", "remote_image", "arbitrary_url", "unregistered_component"}}
}
