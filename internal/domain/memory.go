package domain

import "time"

type MemoryScopeV1 struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type MemorySourceV1 struct {
	ID                     string    `json:"id"`
	Kind                   string    `json:"kind"`
	Trust                  string    `json:"trust"`
	Confidence             float64   `json:"confidence"`
	Checksum               string    `json:"checksum"`
	SecurityClassification string    `json:"security_classification"`
	CreatedAt              time.Time `json:"created_at"`
}

type MemoryItemV1 struct {
	SchemaVersion    int            `json:"schema_version"`
	ID               string         `json:"id"`
	Scope            MemoryScopeV1  `json:"scope"`
	Kind             string         `json:"kind"`
	Content          string         `json:"content"`
	Revision         int            `json:"revision"`
	Source           MemorySourceV1 `json:"source"`
	EmbeddingVersion string         `json:"embedding_version"`
	OutcomeQuality   float64        `json:"outcome_quality"`
	Inferred         bool           `json:"inferred"`
	SupersedesID     string         `json:"supersedes_id,omitempty"`
	Tombstoned       bool           `json:"tombstoned"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type MemoryRevisionV1 struct {
	ItemID      string    `json:"item_id"`
	Revision    int       `json:"revision"`
	Content     string    `json:"content"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

type MemoryCandidateV1 struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	Scope         MemoryScopeV1  `json:"scope"`
	Kind          string         `json:"kind"`
	Content       string         `json:"content"`
	Source        MemorySourceV1 `json:"source"`
	CandidateHash string         `json:"candidate_hash"`
	State         string         `json:"state"`
	RequiresHuman bool           `json:"requires_human"`
	CreatedAt     time.Time      `json:"created_at"`
}

type MemoryEntityV1 struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Scope     MemoryScopeV1  `json:"scope"`
	Version   int64          `json:"version"`
	Metadata  map[string]any `json:"metadata"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type MemoryEdgeV1 struct {
	ID        string    `json:"id"`
	FromID    string    `json:"from_id"`
	ToID      string    `json:"to_id"`
	Kind      string    `json:"kind"`
	SourceID  string    `json:"source_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RetrievalHitV2 struct {
	ItemID           string        `json:"item_id"`
	Kind             string        `json:"kind"`
	Content          string        `json:"content"`
	Scope            MemoryScopeV1 `json:"scope"`
	SourceID         string        `json:"source_id"`
	Trust            string        `json:"trust"`
	VectorScore      float64       `json:"vector_score"`
	KeywordScore     float64       `json:"keyword_score"`
	FreshnessScore   float64       `json:"freshness_score"`
	TrustScore       float64       `json:"trust_score"`
	CombinedScore    float64       `json:"combined_score"`
	EmbeddingVersion string        `json:"embedding_version"`
}

type RetrievalReceiptV1 struct {
	ID               string           `json:"id"`
	TurnID           string           `json:"turn_id,omitempty"`
	ActorID          string           `json:"actor_identity"`
	QueryHash        string           `json:"query_hash"`
	Scopes           []MemoryScopeV1  `json:"scopes"`
	Mode             string           `json:"mode"`
	EmbeddingVersion string           `json:"embedding_version"`
	Hits             []RetrievalHitV2 `json:"hits"`
	DurationMS       int64            `json:"duration_ms"`
	CreatedAt        time.Time        `json:"created_at"`
}

type ConversationTurnV1 struct {
	ID        string    `json:"id"`
	ActorID   string    `json:"actor_identity"`
	SessionID string    `json:"session_id"`
	MissionID string    `json:"mission_id,omitempty"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	SourceID  string    `json:"source_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ContextAssemblyV1 struct {
	SchemaVersion   int                  `json:"schema_version"`
	ID              string               `json:"id"`
	TurnID          string               `json:"turn_id"`
	ActorID         string               `json:"actor_identity"`
	SessionID       string               `json:"session_id"`
	MissionID       string               `json:"mission_id,omitempty"`
	RecentTurns     []ConversationTurnV1 `json:"recent_turns"`
	Semantic        []RetrievalHitV2     `json:"semantic_memories"`
	Procedural      []RetrievalHitV2     `json:"procedural_chunks"`
	Episodes        []RetrievalHitV2     `json:"operational_episodes"`
	EstimatedTokens int                  `json:"estimated_tokens"`
	TokenBudget     int                  `json:"token_budget"`
	FallbackMode    string               `json:"fallback_mode"`
	ReceiptID       string               `json:"retrieval_receipt_id"`
	CreatedAt       time.Time            `json:"created_at"`
}

type MemoryBundleV1 struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	NodeID        string         `json:"node_id"`
	Scope         MemoryScopeV1  `json:"scope"`
	FromWatermark int64          `json:"from_watermark"`
	ToWatermark   int64          `json:"to_watermark"`
	Items         []MemoryItemV1 `json:"items"`
	Tombstones    []string       `json:"tombstones"`
	ContentHash   string         `json:"content_hash"`
	Signature     string         `json:"signature"`
	CreatedAt     time.Time      `json:"created_at"`
}

type MemorySyncStateV1 struct {
	NodeID           string    `json:"node_id"`
	CentralWatermark int64     `json:"central_watermark"`
	LocalWatermark   int64     `json:"local_watermark"`
	PendingBundles   int       `json:"pending_bundles"`
	LastReceiptAt    time.Time `json:"last_receipt_at"`
	State            string    `json:"state"`
}

type MemoryReplayV1 struct {
	ID                  string    `json:"id"`
	State               string    `json:"state"`
	SourceEvents        int64     `json:"source_events"`
	ProjectedItems      int64     `json:"projected_items"`
	ProjectedRevisions  int64     `json:"projected_revisions"`
	ProjectedTombstones int64     `json:"projected_tombstones"`
	LiveChecksum        string    `json:"live_checksum"`
	ReplayChecksum      string    `json:"replay_checksum"`
	Matches             bool      `json:"matches"`
	StartedAt           time.Time `json:"started_at"`
	CompletedAt         time.Time `json:"completed_at"`
}

type MemorySnapshotV1 struct {
	SchemaVersion     int                 `json:"schema_version"`
	StateVersion      int64               `json:"state_version"`
	Available         bool                `json:"available"`
	Phase             string              `json:"phase"`
	EmbeddingState    string              `json:"embedding_state"`
	EmbeddingVersion  string              `json:"embedding_version"`
	RetrievalMode     string              `json:"retrieval_mode"`
	CommittedItems    int64               `json:"committed_items"`
	PendingCandidates int64               `json:"pending_candidates"`
	ConversationTurns int64               `json:"conversation_turns"`
	Tombstones        int64               `json:"tombstones"`
	KafkaLag          int64               `json:"kafka_lag"`
	LastContext       *ContextAssemblyV1  `json:"last_context,omitempty"`
	LastReceipt       *RetrievalReceiptV1 `json:"last_receipt,omitempty"`
	Sync              []MemorySyncStateV1 `json:"sync"`
	MemoryLab         map[string]any      `json:"memory_lab"`
	Summary           string              `json:"summary"`
	UpdatedAt         time.Time           `json:"updated_at"`
}
