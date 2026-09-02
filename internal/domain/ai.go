package domain

import "time"

type ProviderSnapshotV1 struct {
	Mode         string              `json:"mode"`
	Selected     string              `json:"selected"`
	Models       []string            `json:"models"`
	Attempts     []ProviderAttemptV1 `json:"attempts"`
	CircuitOpen  []string            `json:"circuit_open"`
	LocalEnabled bool                `json:"local_enabled"`
	CloudEnabled bool                `json:"cloud_enabled"`
}

type ProviderAttemptV1 struct {
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	State      string    `json:"state"`
	StartedAt  time.Time `json:"started_at"`
	LatencyMS  int64     `json:"latency_ms"`
	StatusCode int       `json:"status_code,omitempty"`
	ErrorCode  string    `json:"error_code,omitempty"`
}

type IncidentEvidenceV1 struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	SourceID   string         `json:"source_id"`
	Summary    string         `json:"summary"`
	Trust      string         `json:"trust"`
	Tick       int64          `json:"tick,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type IncidentManifestV1 struct {
	SchemaVersion  int                  `json:"schema_version"`
	ID             string               `json:"id"`
	Title          string               `json:"title"`
	Summary        string               `json:"summary"`
	ScenarioSeed   int64                `json:"scenario_seed"`
	FaultSchedule  []string             `json:"fault_schedule"`
	Evidence       []IncidentEvidenceV1 `json:"evidence"`
	StateChecksum  string               `json:"state_checksum"`
	BuildCommit    string               `json:"build_commit"`
	Fixture        bool                 `json:"fixture"`
	Classification string               `json:"classification"`
	CapturedAt     time.Time            `json:"captured_at"`
}

type CitationV1 struct {
	SourceID string `json:"source_id"`
	ChunkID  string `json:"chunk_id"`
	Title    string `json:"title"`
	Trust    string `json:"trust"`
	Excerpt  string `json:"excerpt"`
}

type ToolReceiptV1 struct {
	ID         string    `json:"id"`
	Tool       string    `json:"tool"`
	State      string    `json:"state"`
	Arguments  string    `json:"arguments"`
	ResultHash string    `json:"result_hash"`
	At         time.Time `json:"at"`
	DurationMS int64     `json:"duration_ms"`
}

type ReplayResultV1 struct {
	IncidentID       string `json:"incident_id"`
	State            string `json:"state"`
	ExpectedChecksum string `json:"expected_checksum"`
	ActualChecksum   string `json:"actual_checksum"`
	Matches          bool   `json:"matches"`
	TransitionCount  int    `json:"transition_count"`
	FirstDivergence  string `json:"first_divergence,omitempty"`
	LiveStateChanged bool   `json:"live_state_changed"`
}

type InvestigationRunV1 struct {
	SchemaVersion      int                 `json:"schema_version"`
	ID                 string              `json:"id"`
	IncidentID         string              `json:"incident_id"`
	State              string              `json:"state"`
	Diagnosis          string              `json:"diagnosis"`
	Confidence         float64             `json:"confidence"`
	EvidenceIDs        []string            `json:"evidence_ids"`
	Citations          []CitationV1        `json:"citations"`
	ToolReceipts       []ToolReceiptV1     `json:"tool_receipts"`
	Providers          []ProviderAttemptV1 `json:"provider_attempts"`
	ProposedAssertions []string            `json:"proposed_assertions"`
	Replay             *ReplayResultV1     `json:"replay,omitempty"`
	CandidateID        string              `json:"candidate_id,omitempty"`
	TraceID            string              `json:"trace_id"`
	StartedAt          time.Time           `json:"started_at"`
	CompletedAt        *time.Time          `json:"completed_at,omitempty"`
	Failure            string              `json:"failure,omitempty"`
}

type EvalCandidateV1 struct {
	SchemaVersion   int        `json:"schema_version"`
	ID              string     `json:"id"`
	IncidentID      string     `json:"incident_id"`
	InvestigationID string     `json:"investigation_id"`
	Version         int        `json:"version"`
	Assertions      []string   `json:"assertions"`
	EvidenceIDs     []string   `json:"evidence_ids"`
	CandidateHash   string     `json:"candidate_hash"`
	State           string     `json:"state"`
	CreatedAt       time.Time  `json:"created_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
}

type EvalCaseV1 struct {
	ID         string   `json:"id"`
	Version    int      `json:"version"`
	Name       string   `json:"name"`
	Assertions []string `json:"assertions"`
	SourceHash string   `json:"source_hash"`
}

type EvalResultV1 struct {
	CaseID    string   `json:"case_id"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	State     string   `json:"state"`
	Passed    int      `json:"passed"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	Failures  []string `json:"failures"`
	LatencyMS int64    `json:"latency_ms"`
}

type EvalRunV1 struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	CandidateID   string         `json:"candidate_id"`
	State         string         `json:"state"`
	SuiteVersion  string         `json:"suite_version"`
	Results       []EvalResultV1 `json:"results"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
}

type SpanSnapshotV1 struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Service      string            `json:"service"`
	State        string            `json:"state"`
	StartedAt    time.Time         `json:"started_at"`
	DurationMS   float64           `json:"duration_ms"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

type TraceSnapshotV1 struct {
	TraceID string           `json:"trace_id"`
	Spans   []SpanSnapshotV1 `json:"spans"`
}

type AgentSnapshotV1 struct {
	SchemaVersion   int                  `json:"schema_version"`
	StateVersion    int64                `json:"state_version"`
	Available       bool                 `json:"available"`
	Phase           string               `json:"phase"`
	Provider        ProviderSnapshotV1   `json:"provider"`
	Incidents       []IncidentManifestV1 `json:"incidents"`
	Investigation   *InvestigationRunV1  `json:"investigation,omitempty"`
	Candidate       *EvalCandidateV1     `json:"candidate,omitempty"`
	Evaluation      *EvalRunV1           `json:"evaluation,omitempty"`
	Trace           *TraceSnapshotV1     `json:"trace,omitempty"`
	SecurityDenials int64                `json:"security_denials"`
	Summary         string               `json:"summary"`
}

type AIMutationV1 struct {
	RequestID              string `json:"request_id"`
	IdempotencyKey         string `json:"idempotency_key"`
	ExpectedAIStateVersion int64  `json:"expected_ai_state_version"`
}

type InvestigateRequestV1 struct{ AIMutationV1 }
type ReplayRequestV1 struct{ AIMutationV1 }
type ApproveEvalCandidateRequestV1 struct {
	AIMutationV1
	CandidateHash    string `json:"candidate_hash"`
	OperatorIdentity string `json:"operator_identity"`
}
type StartEvalRunRequestV1 struct {
	AIMutationV1
	CandidateID string `json:"candidate_id"`
}
type AIFaultCommandV1 struct {
	AIMutationV1
	Kind string `json:"kind"`
}

type AIEvidenceReportV1 struct {
	RunID           string              `json:"run_id"`
	Commit          string              `json:"commit"`
	GeneratedAt     time.Time           `json:"generated_at"`
	Provider        ProviderSnapshotV1  `json:"provider"`
	Investigation   *InvestigationRunV1 `json:"investigation,omitempty"`
	Candidate       *EvalCandidateV1    `json:"candidate,omitempty"`
	Evaluation      *EvalRunV1          `json:"evaluation,omitempty"`
	Trace           *TraceSnapshotV1    `json:"trace,omitempty"`
	SecurityDenials int64               `json:"security_denials"`
}
