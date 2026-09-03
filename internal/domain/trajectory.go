package domain

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
