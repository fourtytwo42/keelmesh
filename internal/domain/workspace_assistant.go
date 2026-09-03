package domain

// WorkspaceAssistantRequestV1 is intentionally limited to operator-visible
// context. The assistant may arrange the workspace and draft a mission, but it
// cannot authorize movement or apply effects through this interface.
type WorkspaceAssistantRequestV1 struct {
	SchemaVersion  int      `json:"schema_version"`
	RequestID      string   `json:"request_id"`
	IdempotencyKey string   `json:"idempotency_key"`
	Text           string   `json:"text"`
	Persona        string   `json:"persona"`
	SelectedIDs    []string `json:"selected_ids"`
	OpenWindows    []string `json:"open_windows"`
}

type WorkspaceAssistantActionV1 struct {
	Kind   string  `json:"kind"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
}

type WorkspaceAssistantResponseV1 struct {
	SchemaVersion int                          `json:"schema_version"`
	Mode          string                       `json:"mode"`
	Speech        string                       `json:"speech"`
	MissionIntent string                       `json:"mission_intent"`
	Actions       []WorkspaceAssistantActionV1 `json:"actions"`
	Provider      string                       `json:"provider"`
	Model         string                       `json:"model"`
	Attempts      []ProviderAttemptV1          `json:"attempts"`
}
