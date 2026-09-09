package domain

import "time"

type PlatformPlaneHealthV1 struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	State      string    `json:"state"`
	Detail     string    `json:"detail"`
	Source     string    `json:"source"`
	Measured   bool      `json:"measured"`
	SampledAt  time.Time `json:"sampled_at"`
	StaleAgeMS int64     `json:"stale_age_ms"`
}

type PlatformSLORecordV1 struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Objective   float64   `json:"objective"`
	Comparison  string    `json:"comparison"`
	State       string    `json:"state"`
	Source      string    `json:"source"`
	Workload    string    `json:"workload"`
	Window      string    `json:"window"`
	Environment string    `json:"environment"`
	Measured    bool      `json:"measured"`
	SampledAt   time.Time `json:"sampled_at"`
}

type PlatformProofSummaryV1 struct {
	SchemaVersion int                     `json:"schema_version"`
	SampledAt     time.Time               `json:"sampled_at"`
	Commit        string                  `json:"commit"`
	Planes        []PlatformPlaneHealthV1 `json:"planes"`
	SLOs          []PlatformSLORecordV1   `json:"slos"`
	LatestTrace   *TraceSnapshotV1        `json:"latest_trace,omitempty"`
	Summary       string                  `json:"summary"`
}

type TraceSummaryV1 struct {
	TraceID    string    `json:"trace_id"`
	StartedAt  time.Time `json:"started_at"`
	DurationMS float64   `json:"duration_ms"`
	SpanCount  int       `json:"span_count"`
	Services   []string  `json:"services"`
	State      string    `json:"state"`
}

type CapacityProfileV1 struct {
	AssetCount      int       `json:"asset_count"`
	EvidenceClass   string    `json:"evidence_class"`
	EventsPerSecond float64   `json:"events_per_second"`
	IngestP95MS     float64   `json:"ingest_p95_ms"`
	ConsumerLag     int64     `json:"consumer_lag"`
	WorkerCount     int       `json:"worker_count"`
	TraceSpans24H   int64     `json:"trace_spans_24h"`
	Assumption      string    `json:"assumption,omitempty"`
	Source          string    `json:"source"`
	SampledAt       time.Time `json:"sampled_at"`
}

type CostProfileV1 struct {
	AssetCount          int     `json:"asset_count"`
	EvidenceClass       string  `json:"evidence_class"`
	ComputeMonthlyUSD   float64 `json:"compute_monthly_usd"`
	StorageMonthlyUSD   float64 `json:"storage_monthly_usd"`
	TransferMonthlyUSD  float64 `json:"transfer_monthly_usd"`
	ModelMonthlyUSD     float64 `json:"model_monthly_usd"`
	EstimatedMonthlyUSD float64 `json:"estimated_monthly_usd"`
	Assumption          string  `json:"assumption"`
}

type CapacityCostEvidenceV1 struct {
	SchemaVersion int                 `json:"schema_version"`
	SampledAt     time.Time           `json:"sampled_at"`
	Capacity      []CapacityProfileV1 `json:"capacity"`
	Cost          []CostProfileV1     `json:"cost"`
	Currency      string              `json:"currency"`
	Disclaimer    string              `json:"disclaimer"`
}

type PlatformDrillRequestV1 struct {
	PlatformMutationV1
	Type          string `json:"type"`
	TargetID      string `json:"target_id"`
	ActorIdentity string `json:"actor_identity"`
	Confirmed     bool   `json:"confirmed"`
}

type PlatformDrillObservationV1 struct {
	At       time.Time `json:"at"`
	Kind     string    `json:"kind"`
	Summary  string    `json:"summary"`
	SourceID string    `json:"source_id,omitempty"`
}

type PlatformDrillReceiptV1 struct {
	SchemaVersion      int                          `json:"schema_version"`
	ID                 string                       `json:"id"`
	Type               string                       `json:"type"`
	TargetID           string                       `json:"target_id"`
	ActorIdentity      string                       `json:"actor_identity"`
	State              string                       `json:"state"`
	ExpectedInvariants []string                     `json:"expected_invariants"`
	Observations       []PlatformDrillObservationV1 `json:"observations"`
	InitialState       map[string]string            `json:"initial_state"`
	FinalState         map[string]string            `json:"final_state,omitempty"`
	RecoveryMS         int64                        `json:"recovery_ms,omitempty"`
	EvidenceHash       string                       `json:"evidence_hash,omitempty"`
	StartedAt          time.Time                    `json:"started_at"`
	CompletedAt        *time.Time                   `json:"completed_at,omitempty"`
	Outcome            string                       `json:"outcome"`
	Reason             string                       `json:"reason,omitempty"`
}

type PlatformEvaluationStageV1 struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	State    string `json:"state"`
	SourceID string `json:"source_id,omitempty"`
	Detail   string `json:"detail"`
}

type PlatformPolicyResultV1 struct {
	Policy                  string  `json:"policy"`
	Detected                bool    `json:"detected"`
	DetectionLatencyTicks   int     `json:"detection_latency_ticks"`
	FalseAcceptance         int     `json:"false_acceptance"`
	RouteDeviationM         float64 `json:"route_deviation_m"`
	UncertaintyGrowthM      float64 `json:"uncertainty_growth_m"`
	TimeToSafeBehaviorTicks int     `json:"time_to_safe_behavior_ticks"`
}

type PlatformEvaluationEvidenceV1 struct {
	SchemaVersion       int                         `json:"schema_version"`
	EpisodeID           string                      `json:"episode_id"`
	IncidentID          string                      `json:"incident_id"`
	ScenarioSeed        int64                       `json:"scenario_seed"`
	SourceStateChecksum string                      `json:"source_state_checksum"`
	SourceEventIDs      []string                    `json:"source_event_ids"`
	ArtifactChecksum    string                      `json:"artifact_checksum"`
	Stages              []PlatformEvaluationStageV1 `json:"stages"`
	Baseline            PlatformPolicyResultV1      `json:"baseline"`
	Candidate           PlatformPolicyResultV1      `json:"candidate"`
	PromotionState      string                      `json:"promotion_state"`
	Redactions          []string                    `json:"redactions"`
	SampledAt           time.Time                   `json:"sampled_at"`
}
