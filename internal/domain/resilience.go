package domain

type TapeSummaryV1 struct {
	DepthSeconds int                    `json:"depth_seconds"`
	Watermark    string                 `json:"watermark"`
	Segments     []MissionTapeSegmentV1 `json:"segments"`
}

type NodeSnapshotV1 struct {
	SchemaVersion      int                `json:"schema_version"`
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Position           Point              `json:"position"`
	Behavior           string             `json:"behavior"`
	ActiveLeaseID      string             `json:"active_lease_id,omitempty"`
	Tape               TapeSummaryV1      `json:"tape"`
	ActiveRoute        []string           `json:"active_route"`
	BufferedBundles    int                `json:"buffered_bundles"`
	BufferedEvents     int                `json:"buffered_events"`
	ExecutionWatermark int                `json:"execution_watermark"`
	PNT                PntEstimateV1      `json:"pnt"`
	PNTObservations    []PntObservationV1 `json:"pnt_observations"`
	LocalSequence      int64              `json:"local_sequence"`
}

type LinkStateV1 struct {
	SchemaVersion      int     `json:"schema_version"`
	ID                 string  `json:"id"`
	SourceID           string  `json:"source_id"`
	DestinationID      string  `json:"destination_id"`
	Underlay           string  `json:"underlay"`
	Reachable          bool    `json:"reachable"`
	LatencyMS          int     `json:"latency_ms"`
	LossPercent        float64 `json:"loss_percent"`
	CapacityClass      string  `json:"capacity_class"`
	Trusted            bool    `json:"trusted"`
	EnergyCost         float64 `json:"energy_cost"`
	LastTransitionTick int64   `json:"last_transition_tick"`
}

type EgressAdvertisementV1 struct {
	SchemaVersion int    `json:"schema_version"`
	NodeID        string `json:"node_id"`
	Internet      bool   `json:"internet"`
	ExpiresTick   int64  `json:"expires_tick"`
	CapacityClass string `json:"capacity_class"`
	Sequence      int64  `json:"sequence"`
	Signature     string `json:"signature"`
}

type PeerBundleV1 struct {
	SchemaVersion   int    `json:"schema_version"`
	ID              string `json:"id"`
	IdempotencyKey  string `json:"idempotency_key"`
	OriginID        string `json:"origin_id"`
	DestinationID   string `json:"destination_id"`
	MessageClass    string `json:"message_class"`
	Priority        int    `json:"priority"`
	AuthorityEpoch  int64  `json:"authority_epoch"`
	MissionID       string `json:"mission_id"`
	PlanHash        string `json:"plan_hash"`
	CreatedTick     int64  `json:"created_tick"`
	ExpiresTick     int64  `json:"expires_tick"`
	HopLimit        int    `json:"hop_limit"`
	PayloadSchema   string `json:"payload_schema"`
	PayloadHash     string `json:"payload_hash"`
	ContentHash     string `json:"content_hash"`
	OriginSignature string `json:"origin_signature"`
}

type HopReceiptV1 struct {
	SchemaVersion int    `json:"schema_version"`
	BundleID      string `json:"bundle_id"`
	RelayID       string `json:"relay_id"`
	IngressLinkID string `json:"ingress_link_id"`
	EgressLinkID  string `json:"egress_link_id"`
	ObservedTick  int64  `json:"observed_tick"`
	Result        string `json:"result"`
}

type MissionTapeSegmentV1 struct {
	SchemaVersion       int     `json:"schema_version"`
	MissionID           string  `json:"mission_id"`
	LeaseID             string  `json:"lease_id"`
	PlanID              string  `json:"plan_id"`
	PlanHash            string  `json:"plan_hash"`
	Revision            int     `json:"revision"`
	Sequence            int     `json:"sequence"`
	ActivationTick      int64   `json:"activation_tick"`
	ExpiryTick          int64   `json:"expiry_tick"`
	PredecessorHash     string  `json:"predecessor_hash,omitempty"`
	RouteCorridor       []Point `json:"route_corridor"`
	SpeedMinMPS         float64 `json:"speed_min_mps"`
	SpeedMaxMPS         float64 `json:"speed_max_mps"`
	ExpectedPosition    Point   `json:"expected_position"`
	MinimumReserve      float64 `json:"minimum_reserve"`
	MaximumUncertaintyM float64 `json:"maximum_uncertainty_m"`
	FailureBehavior     string  `json:"failure_behavior"`
	Lifecycle           string  `json:"lifecycle"`
	ContentHash         string  `json:"content_hash"`
	Signature           string  `json:"signature"`
}

type SegmentEventV1 struct {
	SchemaVersion int     `json:"schema_version"`
	NodeID        string  `json:"node_id"`
	SegmentHash   string  `json:"segment_hash"`
	Lifecycle     string  `json:"lifecycle"`
	LocalSequence int64   `json:"local_sequence"`
	Position      Point   `json:"position"`
	Reserve       float64 `json:"reserve"`
	MissionTick   int64   `json:"mission_tick"`
	ReasonCode    string  `json:"reason_code,omitempty"`
}

type PntObservationV1 struct {
	SchemaVersion int      `json:"schema_version"`
	Source        string   `json:"source"`
	Position      Point    `json:"position"`
	SpeedMPS      float64  `json:"speed_mps"`
	HeadingDeg    float64  `json:"heading_deg"`
	Sequence      int64    `json:"sequence"`
	AgeSeconds    float64  `json:"age_seconds"`
	UncertaintyM  float64  `json:"uncertainty_m"`
	IntegrityOK   bool     `json:"integrity_ok"`
	ReasonCodes   []string `json:"reason_codes"`
}

type PntEstimateV1 struct {
	SchemaVersion       int      `json:"schema_version"`
	Position            Point    `json:"position"`
	SpeedMPS            float64  `json:"speed_mps"`
	HeadingDeg          float64  `json:"heading_deg"`
	UncertaintyM        float64  `json:"uncertainty_m"`
	Integrity           string   `json:"integrity"`
	ContributingSources []string `json:"contributing_sources"`
	ExcludedSources     []string `json:"excluded_sources"`
	ReasonCodes         []string `json:"reason_codes"`
	LeaseThresholdM     float64  `json:"lease_threshold_m"`
	Behavior            string   `json:"behavior"`
}

type RejoinBridgeV1 struct {
	SchemaVersion      int      `json:"schema_version"`
	ActualStateHash    string   `json:"actual_state_hash"`
	ExecutionWatermark int      `json:"execution_watermark"`
	DiscardedSequences []int    `json:"discarded_sequences"`
	TargetSequence     int      `json:"target_sequence"`
	TargetTick         int64    `json:"target_tick"`
	Route              []Point  `json:"route"`
	PolicyStatus       string   `json:"policy_status"`
	ReasonCodes        []string `json:"reason_codes"`
	ContentHash        string   `json:"content_hash"`
	RequiresApproval   bool     `json:"requires_approval"`
}

type FaultCommandV1 struct {
	SchemaVersion        int            `json:"schema_version"`
	Kind                 string         `json:"kind"`
	TargetID             string         `json:"target_id"`
	ScenarioTick         int64          `json:"scenario_tick"`
	RequestID            string         `json:"request_id"`
	IdempotencyKey       string         `json:"idempotency_key"`
	ExpectedStateVersion int64          `json:"expected_state_version"`
	Parameters           map[string]any `json:"parameters,omitempty"`
}

type ResilienceMutationV1 struct {
	RequestID            string `json:"request_id"`
	IdempotencyKey       string `json:"idempotency_key"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
}

type ResilienceSnapshotV1 struct {
	SchemaVersion       int                     `json:"schema_version"`
	StateVersion        int64                   `json:"state_version"`
	ScenarioID          string                  `json:"scenario_id"`
	Phase               string                  `json:"phase"`
	MissionTick         int64                   `json:"mission_tick"`
	IncidentNodeID      string                  `json:"incident_node_id"`
	RelayNodeID         string                  `json:"relay_node_id"`
	Nodes               []NodeSnapshotV1        `json:"nodes"`
	Links               []LinkStateV1           `json:"links"`
	Advertisements      []EgressAdvertisementV1 `json:"advertisements"`
	ActivePath          []string                `json:"active_path"`
	HopReceipts         []HopReceiptV1          `json:"hop_receipts"`
	QueuedBundles       int                     `json:"queued_bundles"`
	DuplicateDeliveries int                     `json:"duplicate_deliveries"`
	DiscardedSequences  []int                   `json:"discarded_sequences"`
	RawGNSSPosition     *Point                  `json:"raw_gnss_position,omitempty"`
	Bridge              *RejoinBridgeV1         `json:"bridge,omitempty"`
	PNTTransitions      []PntEstimateV1         `json:"pnt_transitions"`
	Summary             string                  `json:"summary"`
	NextAction          string                  `json:"next_action,omitempty"`
	AutoRunAvailable    bool                    `json:"auto_run_available"`
}
