package domain

import "time"

const SchemaVersion = 1

type Point [2]float64

type Polygon struct {
	Type        string    `json:"type"`
	Coordinates [][]Point `json:"coordinates"`
}

func NewPolygon(ring []Point) Polygon {
	return Polygon{Type: "Polygon", Coordinates: [][]Point{ring}}
}

type VesselV1 struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Position      Point   `json:"position"`
	HeadingDeg    float64 `json:"heading_deg"`
	Reserve       float64 `json:"reserve"`
	SpeedMPS      float64 `json:"speed_mps"`
	Available     bool    `json:"available"`
	RouteIndex    int     `json:"route_index"`
	RouteProgress float64 `json:"route_progress"`
}

type ZoneV1 struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Geometry Polygon `json:"geometry"`
}

type MissionStateV1 struct {
	ID        string  `json:"id,omitempty"`
	Phase     string  `json:"phase"`
	PlanID    string  `json:"plan_id,omitempty"`
	PlanHash  string  `json:"plan_hash,omitempty"`
	LeaseID   string  `json:"lease_id,omitempty"`
	StartedAt string  `json:"started_at,omitempty"`
	Progress  float64 `json:"progress"`
}

type FleetSnapshotV1 struct {
	SchemaVersion  int                   `json:"schema_version"`
	StateVersion   int64                 `json:"state_version"`
	ScenarioID     string                `json:"scenario_id"`
	ScenarioName   string                `json:"scenario_name"`
	SimulationRate float64               `json:"simulation_rate"`
	Vessels        []VesselV1            `json:"vessels"`
	Mission        MissionStateV1        `json:"mission"`
	Resilience     *ResilienceSnapshotV1 `json:"resilience,omitempty"`
}

type IntentConstraintsV1 struct {
	MinimumReserve         float64  `json:"minimum_reserve"`
	MaximumDurationMinutes float64  `json:"maximum_duration_minutes"`
	AvoidZones             []string `json:"avoid_zones"`
}

type MissionIntentV1 struct {
	SchemaVersion       int                 `json:"schema_version"`
	ID                  string              `json:"id"`
	TraceID             string              `json:"trace_id"`
	SourceStateVersion  int64               `json:"source_state_version"`
	Objective           string              `json:"objective"`
	Area                Polygon             `json:"area"`
	RequestedAssetCount int                 `json:"requested_asset_count"`
	Constraints         IntentConstraintsV1 `json:"constraints"`
	SourceText          string              `json:"source_text"`
	ContentHash         string              `json:"content_hash"`
}

type PolicyDecisionV1 struct {
	Status      string   `json:"status"`
	ReasonCodes []string `json:"reason_codes"`
	Summary     string   `json:"summary"`
}

type ScoreBreakdownV1 struct {
	Coverage float64 `json:"coverage"`
	Reserve  float64 `json:"reserve"`
	Duration float64 `json:"duration"`
	Total    float64 `json:"total"`
}

type AssignmentV1 struct {
	VesselID   string  `json:"vessel_id"`
	Route      []Point `json:"route"`
	SpeedMPS   float64 `json:"speed_mps"`
	DistanceKM float64 `json:"distance_km"`
}

type PlanMetricsV1 struct {
	CoveragePercent      float64 `json:"coverage_percent"`
	DurationMinutes      float64 `json:"duration_minutes"`
	MinimumReserve       float64 `json:"minimum_reserve"`
	TotalRouteDistanceKM float64 `json:"total_route_distance_km"`
}

type PlanCandidateV1 struct {
	SchemaVersion      int              `json:"schema_version"`
	ID                 string           `json:"id"`
	IntentID           string           `json:"intent_id"`
	SourceStateVersion int64            `json:"source_state_version"`
	Name               string           `json:"name"`
	Strategy           string           `json:"strategy"`
	Summary            string           `json:"summary"`
	Assignments        []AssignmentV1   `json:"assignments"`
	Metrics            PlanMetricsV1    `json:"metrics"`
	Policy             PolicyDecisionV1 `json:"policy"`
	Score              ScoreBreakdownV1 `json:"score"`
	Recommended        bool             `json:"recommended"`
	ContentHash        string           `json:"content_hash"`
}

type PreviewSampleV1 struct {
	Second    int              `json:"second"`
	Positions map[string]Point `json:"positions"`
}

type PlanPreviewV1 struct {
	PlanID          string            `json:"plan_id"`
	PlanHash        string            `json:"plan_hash"`
	DurationSeconds int               `json:"duration_seconds"`
	Samples         []PreviewSampleV1 `json:"samples"`
}

type MissionLeaseV1 struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	MissionID     string    `json:"mission_id"`
	PlanID        string    `json:"plan_id"`
	PlanHash      string    `json:"plan_hash"`
	OperatorID    string    `json:"operator_id"`
	AssetIDs      []string  `json:"asset_ids"`
	Area          Polygon   `json:"area"`
	MinReserve    float64   `json:"minimum_reserve"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Signature     string    `json:"signature"`
}

type AuditEventV1 struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	TraceID       string         `json:"trace_id"`
	Kind          string         `json:"kind"`
	At            time.Time      `json:"at"`
	Summary       string         `json:"summary"`
	Details       map[string]any `json:"details,omitempty"`
}

type StreamMessageV1 struct {
	SchemaVersion int                   `json:"schema_version"`
	Sequence      int64                 `json:"sequence"`
	Kind          string                `json:"kind"`
	Snapshot      *FleetSnapshotV1      `json:"snapshot,omitempty"`
	Audit         *AuditEventV1         `json:"audit,omitempty"`
	Resilience    *ResilienceSnapshotV1 `json:"resilience,omitempty"`
	Platform      *PlatformSnapshotV1   `json:"platform,omitempty"`
}

type BootstrapV1 struct {
	SchemaVersion int             `json:"schema_version"`
	Snapshot      FleetSnapshotV1 `json:"snapshot"`
	Boundary      ZoneV1          `json:"boundary"`
	SuggestedArea ZoneV1          `json:"suggested_area"`
	ExclusionZone ZoneV1          `json:"exclusion_zone"`
	HoldingArea   ZoneV1          `json:"holding_area"`
	Capabilities  []string        `json:"capabilities"`
	Audit         []AuditEventV1  `json:"audit"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
