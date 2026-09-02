package domain

type GroupMissionContractV1 struct {
	SchemaVersion         int      `json:"schema_version"`
	ID                    string   `json:"id"`
	MissionID             string   `json:"mission_id"`
	PlanHash              string   `json:"plan_hash"`
	AuthorityEpoch        int64    `json:"authority_epoch"`
	Members               []string `json:"members"`
	CoordinatorOrder      []string `json:"coordinator_order"`
	Quorum                int      `json:"quorum"`
	WindowIntervalSeconds int64    `json:"window_interval_seconds"`
	MaximumBytesPerWindow int      `json:"maximum_bytes_per_window"`
	MaximumRounds         int      `json:"maximum_rounds"`
	MinimumActivationLead int64    `json:"minimum_activation_lead_seconds"`
	TapeBoundarySeconds   int64    `json:"tape_boundary_seconds"`
	AllowedAdaptations    []string `json:"allowed_adaptations"`
	BulkTrafficSuppressed bool     `json:"bulk_traffic_suppressed"`
	ContentHash           string   `json:"content_hash"`
	Signature             string   `json:"signature"`
}

type CoordinationWindowV1 struct {
	Round        int    `json:"round"`
	OpensTick    int64  `json:"opens_tick"`
	ClosesTick   int64  `json:"closes_tick"`
	BytesUsed    int    `json:"bytes_used"`
	ByteBudget   int    `json:"byte_budget"`
	MessageCount int    `json:"message_count"`
	State        string `json:"state"`
}

type GroupAdaptationProposalV1 struct {
	SchemaVersion  int            `json:"schema_version"`
	ID             string         `json:"id"`
	Revision       int            `json:"revision"`
	AuthorityEpoch int64          `json:"authority_epoch"`
	CoordinatorID  string         `json:"coordinator_id"`
	Reason         string         `json:"reason"`
	Source         string         `json:"source"`
	CreatedTick    int64          `json:"created_tick"`
	ExpiresTick    int64          `json:"expires_tick"`
	AffectedNodes  []string       `json:"affected_nodes"`
	Assignments    []AssignmentV1 `json:"assignments"`
	ContentHash    string         `json:"content_hash"`
	Signature      string         `json:"signature"`
}

type NodeDecisionV1 struct {
	SchemaVersion int    `json:"schema_version"`
	NodeID        string `json:"node_id"`
	ProposalHash  string `json:"proposal_hash"`
	Decision      string `json:"decision"`
	ReasonCode    string `json:"reason_code,omitempty"`
	DecidedTick   int64  `json:"decided_tick"`
	Signature     string `json:"signature"`
}

type GroupCommitV1 struct {
	SchemaVersion  int      `json:"schema_version"`
	ID             string   `json:"id"`
	ProposalHash   string   `json:"proposal_hash"`
	AuthorityEpoch int64    `json:"authority_epoch"`
	CommitTick     int64    `json:"commit_tick"`
	ActivationTick int64    `json:"activation_tick"`
	ArmedNodes     []string `json:"armed_nodes"`
	ContentHash    string   `json:"content_hash"`
	Signature      string   `json:"signature"`
}

type CoordinationMetricsV1 struct {
	Rounds                 int `json:"rounds"`
	BytesSent              int `json:"bytes_sent"`
	ByteBudget             int `json:"byte_budget"`
	MessagesSent           int `json:"messages_sent"`
	BulkMessagesSuppressed int `json:"bulk_messages_suppressed"`
	QuorumCount            int `json:"quorum_count"`
	QuorumRequired         int `json:"quorum_required"`
	AffectedArmed          int `json:"affected_armed"`
	AffectedRequired       int `json:"affected_required"`
}

type QuietFleetMutationV1 struct {
	RequestID            string `json:"request_id"`
	IdempotencyKey       string `json:"idempotency_key"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
}

type QuietFleetCommandV1 struct {
	SchemaVersion        int            `json:"schema_version"`
	Kind                 string         `json:"kind"`
	RequestID            string         `json:"request_id"`
	IdempotencyKey       string         `json:"idempotency_key"`
	ExpectedStateVersion int64          `json:"expected_state_version"`
	ProposalHash         string         `json:"proposal_hash,omitempty"`
	Parameters           map[string]any `json:"parameters,omitempty"`
}

type QuietFleetSnapshotV1 struct {
	SchemaVersion     int                        `json:"schema_version"`
	StateVersion      int64                      `json:"state_version"`
	ScenarioID        string                     `json:"scenario_id"`
	Phase             string                     `json:"phase"`
	MissionTick       int64                      `json:"mission_tick"`
	Contract          GroupMissionContractV1     `json:"contract"`
	CoordinatorID     string                     `json:"coordinator_id"`
	Vessel4SpeedMPS   float64                    `json:"vessel4_speed_mps"`
	ActiveAssignments []AssignmentV1             `json:"active_assignments"`
	Proposal          *GroupAdaptationProposalV1 `json:"proposal,omitempty"`
	Decisions         []NodeDecisionV1           `json:"decisions"`
	Commit            *GroupCommitV1             `json:"commit,omitempty"`
	Windows           []CoordinationWindowV1     `json:"windows"`
	Metrics           CoordinationMetricsV1      `json:"metrics"`
	Summary           string                     `json:"summary"`
	NextAction        string                     `json:"next_action,omitempty"`
	AutoRunAvailable  bool                       `json:"auto_run_available"`
	InferenceLabel    string                     `json:"inference_label"`
}
