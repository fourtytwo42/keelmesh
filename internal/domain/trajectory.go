package domain

import "time"

// TrajectoryProgramV2 is the complete finite execution authority installed on
// every assigned vessel. Unlike the deprecated V1 hot-tape projection, it
// carries the entire signed program plus an explicit authorization boundary.
type TrajectoryProgramV2 struct {
	SchemaVersion           int                          `json:"schema_version"`
	ProgramID               string                       `json:"program_id"`
	MissionID               string                       `json:"mission_id"`
	PlanID                  string                       `json:"plan_id"`
	PlanHash                string                       `json:"plan_hash"`
	LeaseID                 string                       `json:"lease_id"`
	IssuedAt                time.Time                    `json:"issued_at"`
	IssuedTick              int64                        `json:"issued_tick"`
	AuthorizationExpiryTick int64                        `json:"authorization_expiry_tick"`
	CompletionPolicy        string                       `json:"completion_policy"`
	TerminalContingency     string                       `json:"terminal_contingency"`
	ActiveRevision          int                          `json:"active_revision"`
	PendingRevision         int                          `json:"pending_revision,omitempty"`
	ActivationTick          int64                        `json:"activation_tick,omitempty"`
	MissionTickMS           int64                        `json:"mission_tick_ms"`
	Revisions               map[int]TrajectoryRevisionV1 `json:"revisions"`
	Cursors                 map[string]ExecutionCursorV2 `json:"cursors"`
	LastAdaptations         map[string]GroupAdaptationV1 `json:"last_adaptations,omitempty"`
	InstallReceipts         []ProgramInstallReceiptV1    `json:"install_receipts,omitempty"`
	ContentHash             string                       `json:"content_hash"`
	Signature               string                       `json:"signature"`
}

type ExecutionCursorV2 struct {
	VesselID                       string `json:"vessel_id"`
	Revision                       int    `json:"revision"`
	Sequence                       int    `json:"sequence"`
	MissionTick                    int64  `json:"mission_tick"`
	ProgramRemainingSeconds        int    `json:"program_remaining_seconds"`
	AuthorizedTimeRemainingSeconds int64  `json:"authorized_time_remaining_seconds"`
	Lifecycle                      string `json:"lifecycle"`
}

type ExecutionAuthorityV1 struct {
	SchemaVersion                  int    `json:"schema_version"`
	ProgramID                      string `json:"program_id"`
	MissionID                      string `json:"mission_id"`
	PlanID                         string `json:"plan_id"`
	Revision                       int    `json:"revision"`
	Status                         string `json:"status"`
	CompleteProgramOnboard         bool   `json:"complete_program_onboard"`
	IssuedTick                     int64  `json:"issued_tick"`
	AuthorizationExpiryTick        int64  `json:"authorization_expiry_tick"`
	AuthorizedTimeRemainingSeconds int64  `json:"authorized_time_remaining_seconds"`
	DecisionNodeID                 string `json:"decision_node_id,omitempty"`
	DecisionScope                  string `json:"decision_scope"`
	DecisionEpoch                  int64  `json:"decision_epoch"`
	TerminalContingency            string `json:"terminal_contingency"`
	ProgramHash                    string `json:"program_hash"`
}

type ProgramManifestV1 struct {
	SchemaVersion           int      `json:"schema_version"`
	ProgramID               string   `json:"program_id"`
	MissionID               string   `json:"mission_id"`
	PlanHash                string   `json:"plan_hash"`
	Revision                int      `json:"revision"`
	VesselIDs               []string `json:"vessel_ids"`
	SegmentCount            int      `json:"segment_count"`
	AuthorizationExpiryTick int64    `json:"authorization_expiry_tick"`
	ContentHash             string   `json:"content_hash"`
}

type ProgramInstallReceiptV1 struct {
	SchemaVersion int       `json:"schema_version"`
	ProgramID     string    `json:"program_id"`
	MissionID     string    `json:"mission_id"`
	NodeID        string    `json:"node_id"`
	Revision      int       `json:"revision"`
	ProgramHash   string    `json:"program_hash"`
	InstalledAt   time.Time `json:"installed_at"`
	State         string    `json:"state"`
	ReceiptHash   string    `json:"receipt_hash"`
	Signature     string    `json:"signature,omitempty"`
}

type GroupAdaptationV1 struct {
	SchemaVersion  int     `json:"schema_version"`
	AdaptationID   string  `json:"adaptation_id"`
	ProgramID      string  `json:"program_id"`
	MissionID      string  `json:"mission_id"`
	VesselID       string  `json:"vessel_id"`
	DecisionNodeID string  `json:"decision_node_id"`
	DecisionScope  string  `json:"decision_scope"`
	DecisionEpoch  int64   `json:"decision_epoch"`
	Tick           int64   `json:"tick"`
	Kind           string  `json:"kind"`
	Reason         string  `json:"reason"`
	HeadingDelta   float64 `json:"heading_delta_deg"`
	SpeedFactor    float64 `json:"speed_factor"`
	LateralOffsetM float64 `json:"lateral_offset_m"`
	InsideEnvelope bool    `json:"inside_envelope"`
	Escalation     string  `json:"escalation"`
	Contingency    string  `json:"contingency,omitempty"`
	ContentHash    string  `json:"content_hash"`
	Signature      string  `json:"signature,omitempty"`
}

type GroupDecisionStateV1 struct {
	SchemaVersion  int                `json:"schema_version"`
	GroupID        string             `json:"group_id"`
	DecisionNodeID string             `json:"decision_node_id,omitempty"`
	DecisionScope  string             `json:"decision_scope"`
	DecisionEpoch  int64              `json:"decision_epoch"`
	ReachableNodes []string           `json:"reachable_nodes"`
	LastAdaptation *GroupAdaptationV1 `json:"last_adaptation,omitempty"`
}

type ExecutionReconciliationV1 struct {
	SchemaVersion      int        `json:"schema_version"`
	ProgramID          string     `json:"program_id"`
	MissionID          string     `json:"mission_id"`
	NodeID             string     `json:"node_id"`
	ProgramRevision    int        `json:"program_revision"`
	ExecutionWatermark int        `json:"execution_watermark"`
	DecisionEpoch      int64      `json:"decision_epoch"`
	FusedPosition      GeoPointV2 `json:"fused_position"`
	ReconciledAt       time.Time  `json:"reconciled_at"`
	DiscardedSegments  int        `json:"discarded_segments"`
	BridgeRequired     bool       `json:"bridge_required"`
	State              string     `json:"state"`
}

type TrajectoryProgramSummaryV2 struct {
	SchemaVersion                  int                          `json:"schema_version"`
	ProgramID                      string                       `json:"program_id"`
	MissionID                      string                       `json:"mission_id"`
	PlanID                         string                       `json:"plan_id"`
	ActiveRevision                 int                          `json:"active_revision"`
	PendingRevision                int                          `json:"pending_revision,omitempty"`
	ActivationTick                 int64                        `json:"activation_tick,omitempty"`
	MissionTick                    int64                        `json:"mission_tick"`
	DurationSeconds                int                          `json:"duration_seconds"`
	TotalSegments                  int                          `json:"total_segments"`
	CompleteProgramOnboard         bool                         `json:"complete_program_onboard"`
	InstalledNodeCount             int                          `json:"installed_node_count"`
	InstallationState              string                       `json:"installation_state"`
	AuthorizationExpiryTick        int64                        `json:"authorization_expiry_tick"`
	AuthorizedTimeRemainingSeconds int64                        `json:"authorized_time_remaining_seconds"`
	CompletionPolicy               string                       `json:"completion_policy"`
	TerminalContingency            string                       `json:"terminal_contingency"`
	Execution                      map[string]ExecutionCursorV2 `json:"execution"`
	LastAdaptations                map[string]GroupAdaptationV1 `json:"last_adaptations,omitempty"`
	ContentHash                    string                       `json:"content_hash"`
}

type TrajectoryProgramViewV2 struct {
	Summary  TrajectoryProgramSummaryV2 `json:"summary"`
	Manifest ProgramManifestV1          `json:"manifest"`
	Program  TrajectoryProgramV2        `json:"program"`
}

// TrajectoryProgramV1 is the complete signed execution program for one
// mission. It may contain any finite number of ten-second segments. Nodes
// materialize only a bounded rolling hot tape from this durable program.
type TrajectoryProgramV1 struct {
	SchemaVersion   int                          `json:"schema_version"`
	MissionID       string                       `json:"mission_id"`
	ActiveRevision  int                          `json:"active_revision"`
	PendingRevision int                          `json:"pending_revision,omitempty"`
	ActivationTick  int64                        `json:"activation_tick,omitempty"`
	MissionTickMS   int64                        `json:"mission_tick_ms"`
	HotTapeHorizonS int                          `json:"hot_tape_horizon_seconds"`
	Revisions       map[int]TrajectoryRevisionV1 `json:"revisions"`
	Cursors         map[string]ExecutionCursorV1 `json:"cursors"`
	LastAdjustments map[string]LocalAdjustmentV1 `json:"last_adjustments,omitempty"`
	ContentHash     string                       `json:"content_hash"`
}

type TrajectoryRevisionV1 struct {
	Revision       int                              `json:"revision"`
	PlanID         string                           `json:"plan_id"`
	PlanHash       string                           `json:"plan_hash"`
	LeaseID        string                           `json:"lease_id"`
	CreatedTick    int64                            `json:"created_tick"`
	ActivationTick int64                            `json:"activation_tick"`
	DurationS      int                              `json:"duration_seconds"`
	Segments       map[string][]TrajectorySegmentV2 `json:"segments"`
	ContentHash    string                           `json:"content_hash"`
	Signature      string                           `json:"signature"`
}

type TrajectorySegmentV2 struct {
	SchemaVersion       int        `json:"schema_version"`
	MissionID           string     `json:"mission_id"`
	PlanHash            string     `json:"plan_hash"`
	Revision            int        `json:"revision"`
	VesselID            string     `json:"vessel_id"`
	Sequence            int        `json:"sequence"`
	ActivationTick      int64      `json:"activation_tick"`
	ExpiryTick          int64      `json:"expiry_tick"`
	Start               GeoPointV2 `json:"start"`
	End                 GeoPointV2 `json:"end"`
	TargetSpeedMPS      float64    `json:"target_speed_mps"`
	MaximumSpeedMPS     float64    `json:"maximum_speed_mps"`
	CorridorRadiusM     float64    `json:"corridor_radius_m"`
	MaxLateralAdjustM   float64    `json:"max_lateral_adjustment_m"`
	ScheduleToleranceS  int        `json:"schedule_tolerance_seconds"`
	MinimumReserve      float64    `json:"minimum_reserve"`
	MinimumSeparationM  float64    `json:"minimum_separation_m"`
	MaximumUncertaintyM float64    `json:"maximum_uncertainty_m"`
	FailureBehavior     string     `json:"failure_behavior"`
	PredecessorHash     string     `json:"predecessor_hash"`
	ContentHash         string     `json:"content_hash"`
	Signature           string     `json:"signature"`
}

type ExecutionCursorV1 struct {
	VesselID          string `json:"vessel_id"`
	Revision          int    `json:"revision"`
	Sequence          int    `json:"sequence"`
	MissionTick       int64  `json:"mission_tick"`
	HotTapeDepthS     int    `json:"hot_tape_depth_seconds"`
	ProgramRemainingS int    `json:"program_remaining_seconds"`
	Lifecycle         string `json:"lifecycle"`
}

type LocalAdjustmentV1 struct {
	VesselID       string  `json:"vessel_id"`
	Tick           int64   `json:"tick"`
	Kind           string  `json:"kind"`
	Reason         string  `json:"reason"`
	HeadingDelta   float64 `json:"heading_delta_deg"`
	SpeedFactor    float64 `json:"speed_factor"`
	LateralOffsetM float64 `json:"lateral_offset_m"`
	InsideEnvelope bool    `json:"inside_envelope"`
	DecisionNodeID string  `json:"decision_node_id"`
	DecisionScope  string  `json:"decision_scope"`
	Escalation     string  `json:"escalation"`
	Contingency    string  `json:"contingency,omitempty"`
}

type TrajectoryProgramSummaryV1 struct {
	MissionID       string                       `json:"mission_id"`
	ActiveRevision  int                          `json:"active_revision"`
	PendingRevision int                          `json:"pending_revision,omitempty"`
	ActivationTick  int64                        `json:"activation_tick,omitempty"`
	MissionTick     int64                        `json:"mission_tick"`
	DurationS       int                          `json:"duration_seconds"`
	TotalSegments   int                          `json:"total_segments"`
	HotTapeHorizonS int                          `json:"hot_tape_horizon_seconds"`
	Execution       map[string]ExecutionCursorV1 `json:"execution"`
	LastAdjustments map[string]LocalAdjustmentV1 `json:"last_adjustments,omitempty"`
	ContentHash     string                       `json:"content_hash"`
}

type TrajectoryProgramViewV1 struct {
	Summary TrajectoryProgramSummaryV1       `json:"summary"`
	HotTape map[string][]TrajectorySegmentV2 `json:"hot_tape"`
}
