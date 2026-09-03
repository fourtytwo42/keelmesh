package domain

import "time"

const KeelMeshOperationsCatalogV1 = "keelmesh-operations-v1"

type AssistantTurnV2 struct {
	SchemaVersion int                          `json:"schema_version"`
	ID            string                       `json:"id"`
	ActorID       string                       `json:"actor_identity"`
	SessionID     string                       `json:"session_id"`
	State         string                       `json:"state"`
	Stages        []string                     `json:"stages"`
	Scene         CommandSceneV1               `json:"scene"`
	Assistant     WorkspaceAssistantResponseV1 `json:"assistant"`
	CreatedAt     time.Time                    `json:"created_at"`
	CompletedAt   time.Time                    `json:"completed_at"`
}

type AssistantTurnRequestV2 struct {
	WorkspaceAssistantRequestV1
	ActorIdentity    string `json:"actor_identity"`
	SessionID        string `json:"session_id"`
	WorkspaceVersion int64  `json:"workspace_version"`
}

type SceneIntentV1 struct {
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	EntityIDs        []string `json:"entity_ids"`
	MissionID        string   `json:"mission_id,omitempty"`
	SuggestedActions []string `json:"suggested_actions"`
}

type CommandSceneV1 struct {
	SchemaVersion    int                     `json:"schema_version"`
	ID               string                  `json:"id"`
	ActorID          string                  `json:"actor_identity"`
	SessionID        string                  `json:"session_id"`
	Type             string                  `json:"type"`
	Title            string                  `json:"title"`
	Summary          string                  `json:"summary"`
	State            string                  `json:"state"`
	Pinned           bool                    `json:"pinned"`
	Critical         bool                    `json:"critical"`
	PendingApproval  bool                    `json:"pending_approval"`
	WorkspaceVersion int64                   `json:"workspace_version"`
	CatalogID        string                  `json:"catalog_id"`
	PrimarySurface   WorkspaceSurfaceV1      `json:"primary_surface"`
	Supporting       []WorkspaceSurfaceV1    `json:"supporting_surfaces"`
	Bindings         []ArtifactDataBindingV1 `json:"bindings"`
	MapCamera        *MapCameraInstructionV1 `json:"map_camera,omitempty"`
	MapAnnotations   []SceneMapAnnotationV1  `json:"map_annotations"`
	SpokenSummary    string                  `json:"spoken_summary"`
	SuggestedActions []ArtifactActionV1      `json:"suggested_actions"`
	Approval         *ApprovalRequestV1      `json:"approval,omitempty"`
	Receipts         []SceneReceiptV1        `json:"receipts"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

type WorkspaceSurfaceV1 struct {
	ID       string           `json:"id"`
	Role     string           `json:"role"`
	Title    string           `json:"title"`
	Sequence int64            `json:"sequence"`
	Messages []map[string]any `json:"messages"`
}

type ArtifactDataBindingV1 struct {
	ID         string `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Field      string `json:"field"`
	Path       string `json:"path"`
}

type MapCameraInstructionV1 struct {
	Center  GeoPointV2 `json:"center"`
	Zoom    float64    `json:"zoom"`
	Bearing float64    `json:"bearing"`
	Pitch   float64    `json:"pitch"`
}

type SceneMapAnnotationV1 struct {
	ID       string       `json:"id"`
	Kind     string       `json:"kind"`
	Label    string       `json:"label"`
	Color    string       `json:"color"`
	Points   []GeoPointV2 `json:"points"`
	RadiusM  float64      `json:"radius_m,omitempty"`
	EntityID string       `json:"entity_id,omitempty"`
}

type ArtifactActionV1 struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Label          string         `json:"label"`
	AuthorityClass string         `json:"authority_class"`
	TargetID       string         `json:"target_id,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	ActionHash     string         `json:"action_hash"`
}

type SceneReceiptV1 struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	State     string    `json:"state"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type ApprovalRequestV1 struct {
	ID            string    `json:"id"`
	ActionID      string    `json:"action_id"`
	ActionHash    string    `json:"action_hash"`
	EntityVersion int64     `json:"entity_version"`
	Summary       string    `json:"summary"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type ProactiveSceneTriggerV1 struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Condition   string    `json:"condition"`
	Transition  string    `json:"transition"`
	Severity    string    `json:"severity"`
	EvidenceIDs []string  `json:"evidence_ids"`
	DetectedAt  time.Time `json:"detected_at"`
}

type SceneMutationV1 struct {
	RequestID        string `json:"request_id"`
	IdempotencyKey   string `json:"idempotency_key"`
	ActorIdentity    string `json:"actor_identity"`
	SessionID        string `json:"session_id"`
	WorkspaceVersion int64  `json:"workspace_version"`
}

type SceneActionRequestV1 struct {
	SceneMutationV1
	ActionID   string         `json:"action_id"`
	ActionHash string         `json:"action_hash,omitempty"`
	Confirmed  bool           `json:"confirmed"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type SceneEventV1 struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	SceneID   string    `json:"scene_id"`
	TurnID    string    `json:"turn_id"`
	Payload   any       `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type SceneCatalogV1 struct {
	ID                string   `json:"id"`
	ProtocolVersion   string   `json:"protocol_version"`
	RendererVersion   string   `json:"renderer_version"`
	AllowedComponents []string `json:"allowed_components"`
	DeniedContent     []string `json:"denied_content"`
}
