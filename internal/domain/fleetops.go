package domain

import "time"

type GeoPointV2 [2]float64

type VesselClassV2 struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Role               string  `json:"role"`
	MaxSpeedMPS        float64 `json:"max_speed_mps"`
	MinimumReserve     float64 `json:"minimum_reserve"`
	EnduranceHours     float64 `json:"endurance_hours"`
	CommunicationsRole bool    `json:"communications_role"`
}

type EnvironmentV2 struct {
	WindSpeedMPS        float64  `json:"wind_speed_mps"`
	WindDirectionDeg    float64  `json:"wind_direction_deg"`
	CurrentSpeedMPS     float64  `json:"current_speed_mps"`
	CurrentDirectionDeg float64  `json:"current_direction_deg"`
	WaveHeightM         float64  `json:"wave_height_m"`
	WaterTemperatureC   float64  `json:"water_temperature_c"`
	FixtureAt           string   `json:"fixture_at"`
	Label               string   `json:"label"`
	SourceIDs           []string `json:"source_ids"`
}

type VesselTelemetryV2 struct {
	Position         GeoPointV2    `json:"position"`
	HeadingDeg       float64       `json:"heading_deg"`
	SpeedMPS         float64       `json:"speed_mps"`
	Reserve          float64       `json:"reserve"`
	ProjectedReserve float64       `json:"projected_reserve"`
	Mode             string        `json:"mode"`
	Health           string        `json:"health"`
	PNTIntegrity     string        `json:"pnt_integrity"`
	UncertaintyM     float64       `json:"uncertainty_m"`
	TapeDepthSeconds int           `json:"tape_depth_seconds"`
	MissionID        string        `json:"mission_id,omitempty"`
	Route            []GeoPointV2  `json:"route"`
	Environment      EnvironmentV2 `json:"environment"`
}

type VesselProfileV2 struct {
	SchemaVersion   int               `json:"schema_version"`
	ID              string            `json:"id"`
	Designation     string            `json:"designation"`
	Callsign        string            `json:"callsign"`
	DisplayName     string            `json:"display_name"`
	Class           VesselClassV2     `json:"class"`
	GroupID         string            `json:"group_id"`
	GroupCode       string            `json:"group_code"`
	GroupColor      string            `json:"group_color"`
	GroupColorName  string            `json:"group_color_name"`
	GroupPattern    string            `json:"group_pattern"`
	Available       bool              `json:"available"`
	DecisionCapable bool              `json:"decision_capable"`
	Telemetry       VesselTelemetryV2 `json:"telemetry"`
}

type OperationalGroupV2 struct {
	SchemaVersion      int                 `json:"schema_version"`
	ID                 string              `json:"id"`
	Code               string              `json:"code"`
	Name               string              `json:"name"`
	Color              string              `json:"color"`
	ColorName          string              `json:"color_name"`
	Pattern            string              `json:"pattern"`
	MemberIDs          []string            `json:"member_ids"`
	Formation          string              `json:"formation"`
	FormationSpacingM  float64             `json:"formation_spacing_m"`
	AssemblyPoint      *GeoPointV2         `json:"assembly_point,omitempty"`
	AssemblySource     string              `json:"assembly_source,omitempty"`
	AssemblyWaypointID string              `json:"assembly_waypoint_id,omitempty"`
	RouteWaypoints     []MissionWaypointV2 `json:"route_waypoints,omitempty"`
	RouteMode          string              `json:"route_mode"`
	RouteIndex         int                 `json:"route_index"`
	RouteRevision      int64               `json:"route_revision"`
	DecisionPolicy     string              `json:"decision_policy"`
	DecisionNodeID     string              `json:"decision_node_id,omitempty"`
	DecisionEpoch      int64               `json:"decision_epoch"`
	FallbackPolicy     string              `json:"fallback_policy"`
	Revision           int64               `json:"revision"`
}

type SavedCollectionV2 struct {
	SchemaVersion int      `json:"schema_version"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MemberIDs     []string `json:"member_ids"`
	Revision      int64    `json:"revision"`
}

type ReachabilityPathV2 struct {
	VesselID  string   `json:"vessel_id"`
	State     string   `json:"state"`
	Hops      []string `json:"hops"`
	Underlay  []string `json:"underlay"`
	LatencyMS float64  `json:"latency_ms"`
}

type ReachabilityV2 struct {
	SchemaVersion int                  `json:"schema_version"`
	VesselID      string               `json:"vessel_id"`
	Authority     string               `json:"authority"`
	DirectPeers   []ReachabilityPathV2 `json:"direct_peers"`
	RelayedPeers  []ReachabilityPathV2 `json:"relayed_peers"`
	Unreachable   []string             `json:"unreachable_group_members"`
	ExternalPeers []ReachabilityPathV2 `json:"reachable_outside_group"`
}

type ConstraintSetV2 struct {
	MinimumReserve              float64 `json:"minimum_reserve"`
	MaximumSpeedMPS             float64 `json:"maximum_speed_mps"`
	MinimumVesselSeparationM    float64 `json:"minimum_vessel_separation_m"`
	MinimumObjectSeparationM    float64 `json:"minimum_object_separation_m"`
	MaximumWaveHeightM          float64 `json:"maximum_wave_height_m"`
	MaximumWindMPS              float64 `json:"maximum_wind_mps"`
	MaximumPNTUncertaintyM      float64 `json:"maximum_pnt_uncertainty_m"`
	MaximumDurationMinutes      float64 `json:"maximum_duration_minutes"`
	MaximumRouteDistanceKM      float64 `json:"maximum_route_distance_km"`
	MaximumShoreDistanceM       float64 `json:"maximum_shore_distance_m,omitempty"`
	MinimumTapeWatermarkSeconds int     `json:"minimum_tape_watermark_seconds"`
	Formation                   string  `json:"formation"`
	FormationSpacingM           float64 `json:"formation_spacing_m"`
	LeaderPolicy                string  `json:"leader_policy"`
	RegroupThresholdM           float64 `json:"regroup_threshold_m"`
}

type MissionGeometryV2 struct {
	Revision        int64               `json:"revision"`
	IncludedAreas   [][][]float64       `json:"included_areas"`
	ExclusionAreas  [][][]float64       `json:"exclusion_areas"`
	Waypoints       []GeoPointV2        `json:"waypoints"`
	WaypointDetails []MissionWaypointV2 `json:"waypoint_details,omitempty"`
	POIs            []MissionPOIV2      `json:"pois"`
}

type MissionWaypointV2 struct {
	ID           string     `json:"id"`
	Position     GeoPointV2 `json:"position"`
	Color        string     `json:"color"`
	Sequence     int        `json:"sequence"`
	OwnerGroupID string     `json:"owner_group_id,omitempty"`
}

type MissionPOIV2 struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Position GeoPointV2 `json:"position"`
	RadiusM  float64    `json:"radius_m"`
}

type MissionWorkspaceV2 struct {
	SchemaVersion      int                         `json:"schema_version"`
	ID                 string                      `json:"id"`
	Name               string                      `json:"name"`
	NameSource         string                      `json:"name_source,omitempty"`
	Objective          string                      `json:"objective"`
	Status             string                      `json:"status"`
	TargetIDs          []string                    `json:"target_ids"`
	TargetSnapshotHash string                      `json:"target_snapshot_hash"`
	FleetVersion       int64                       `json:"fleet_version"`
	Version            int64                       `json:"version"`
	Geometry           MissionGeometryV2           `json:"geometry"`
	Constraints        ConstraintSetV2             `json:"constraints"`
	Formation          string                      `json:"formation"`
	PlanIDs            []string                    `json:"plan_ids"`
	AuthorizedPlanID   string                      `json:"authorized_plan_id,omitempty"`
	Conversation       []MissionChatMessageV2      `json:"conversation"`
	Trajectory         *TrajectoryProgramSummaryV1 `json:"trajectory,omitempty"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

type MissionChatMessageV2 struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Markdown  string    `json:"markdown"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type CommandDraftV2 struct {
	SchemaVersion       int              `json:"schema_version"`
	ID                  string           `json:"id"`
	MissionID           string           `json:"mission_id"`
	SourceText          string           `json:"source_text"`
	Objective           string           `json:"objective"`
	TargetIDs           []string         `json:"target_ids"`
	TargetSnapshotHash  string           `json:"target_snapshot_hash"`
	GeometryRevision    int64            `json:"geometry_revision"`
	FleetVersion        int64            `json:"fleet_version"`
	Constraints         ConstraintSetV2  `json:"constraints"`
	FormationPreference string           `json:"formation_preference"`
	GuidanceKind        string           `json:"guidance_kind"`
	FollowContactID     string           `json:"follow_contact_id,omitempty"`
	Waypoints           []GeoPointV2     `json:"waypoints"`
	GeometrySource      string           `json:"geometry_source,omitempty"`
	ResolutionNotes     []string         `json:"resolution_notes,omitempty"`
	Ambiguities         []string         `json:"unresolved_ambiguities"`
	Advisor             MissionAdvisorV2 `json:"advisor"`
	ContentHash         string           `json:"content_hash"`
}

// SurfaceContactV2 is fictional non-fleet traffic in the local operating
// picture. It carries no KeelMesh command authority and is safe to expose to
// the operator and bounded mission advisor.
type SurfaceContactV2 struct {
	ID              string       `json:"id"`
	BoatID          string       `json:"boat_id"`
	Name            string       `json:"name"`
	Callsign        string       `json:"callsign"`
	Class           string       `json:"class"`
	Activity        string       `json:"activity"`
	ColorName       string       `json:"color_name"`
	Color           string       `json:"color"`
	Position        GeoPointV2   `json:"position"`
	HeadingDeg      float64      `json:"heading_deg"`
	SpeedMPS        float64      `json:"speed_mps"`
	SpeedKnots      float64      `json:"speed_knots"`
	LengthM         float64      `json:"length_m"`
	DraftM          float64      `json:"draft_m"`
	NavigationState string       `json:"navigation_state"`
	RouteName       string       `json:"route_name"`
	Route           []GeoPointV2 `json:"route"`
	Looping         bool         `json:"looping"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// MissionStrategyV2 is advisory model output. It may choose a bounded planning
// posture, but it never contains routes, signatures, leases, or authority.
type MissionStrategyV2 struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Formation    string   `json:"formation"`
	GuidanceKind string   `json:"guidance_kind"`
	SpeedFactor  float64  `json:"speed_factor"`
	ReserveBias  float64  `json:"reserve_bias"`
	Maneuvers    []string `json:"maneuvers"`
}

type MissionAdvisorV2 struct {
	State            string              `json:"state"`
	Provider         string              `json:"provider"`
	Model            string              `json:"model"`
	Summary          string              `json:"summary"`
	MissionName      string              `json:"mission_name,omitempty"`
	GeometryOptionID string              `json:"geometry_option_id,omitempty"`
	Strategies       []MissionStrategyV2 `json:"strategies"`
	Attempts         []ProviderAttemptV1 `json:"attempts"`
}

type MissionGeometryOptionV2 struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Center              GeoPointV2   `json:"center"`
	Boundary            []GeoPointV2 `json:"boundary"`
	Waypoints           []GeoPointV2 `json:"waypoints"`
	DistanceToTargetsKM float64      `json:"distance_to_targets_km"`
	DepthValidated      bool         `json:"depth_validated"`
}

type MissionPlanningVesselV2 struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Class          string     `json:"class"`
	Position       GeoPointV2 `json:"position"`
	Reserve        float64    `json:"reserve"`
	MaxSpeedMPS    float64    `json:"max_speed_mps"`
	PNTIntegrity   string     `json:"pnt_integrity"`
	UncertaintyM   float64    `json:"uncertainty_m"`
	GroupCode      string     `json:"group_code"`
	GroupName      string     `json:"group_name"`
	GroupColorName string     `json:"group_color_name"`
	Communications string     `json:"communications"`
}

type MissionPlanningContextV2 struct {
	SchemaVersion    int                       `json:"schema_version"`
	MissionID        string                    `json:"mission_id"`
	Intent           string                    `json:"intent"`
	GuidanceKind     string                    `json:"guidance_kind"`
	TargetCount      int                       `json:"target_count"`
	Targets          []MissionPlanningVesselV2 `json:"targets"`
	Constraints      ConstraintSetV2           `json:"constraints"`
	Environment      EnvironmentV2             `json:"environment"`
	OperatingAreas   int                       `json:"operating_areas"`
	ExclusionAreas   int                       `json:"exclusion_areas"`
	WaypointCount    int                       `json:"waypoint_count"`
	GeometrySource   string                    `json:"geometry_source,omitempty"`
	GeometryOptions  []MissionGeometryOptionV2 `json:"geometry_options,omitempty"`
	MapBounds        [][]float64               `json:"map_bounds"`
	FormationCurrent string                    `json:"formation_current"`
	Conversation     []MissionChatMessageV2    `json:"conversation,omitempty"`
	SurfaceContacts  []SurfaceContactV2        `json:"surface_contacts"`
	FollowContact    *SurfaceContactV2         `json:"follow_contact,omitempty"`
}

type FleetAssignmentV2 struct {
	VesselID   string       `json:"vessel_id"`
	Route      []GeoPointV2 `json:"route"`
	SpeedMPS   float64      `json:"speed_mps"`
	DistanceKM float64      `json:"distance_km"`
}

type FleetPlanV2 struct {
	SchemaVersion        int                 `json:"schema_version"`
	ID                   string              `json:"id"`
	MissionID            string              `json:"mission_id"`
	DraftID              string              `json:"draft_id"`
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	Formation            string              `json:"formation"`
	AdvisorSource        string              `json:"advisor_source"`
	AdvisorModel         string              `json:"advisor_model,omitempty"`
	Maneuvers            []string            `json:"maneuvers"`
	Assignments          []FleetAssignmentV2 `json:"assignments"`
	CoveragePercent      float64             `json:"coverage_percent"`
	MinimumReserve       float64             `json:"minimum_reserve"`
	DurationMinutes      float64             `json:"duration_minutes"`
	EnergyKWH            float64             `json:"energy_kwh"`
	LinkExposureSeconds  float64             `json:"link_exposure_seconds"`
	MinimumSeparationM   float64             `json:"minimum_separation_m"`
	PolicyStatus         string              `json:"policy_status"`
	ReasonCodes          []string            `json:"reason_codes"`
	Recommended          bool                `json:"recommended"`
	ContentHash          string              `json:"content_hash"`
	SourceMissionVersion int64               `json:"source_mission_version"`
}

type FleetPreviewV2 struct {
	PlanID          string                  `json:"plan_id"`
	PlanHash        string                  `json:"plan_hash"`
	DurationSeconds int                     `json:"duration_seconds"`
	Routes          map[string][]GeoPointV2 `json:"routes"`
	NothingSent     bool                    `json:"nothing_sent"`
}

type FleetLeaseV2 struct {
	ID         string    `json:"id"`
	MissionID  string    `json:"mission_id"`
	PlanID     string    `json:"plan_id"`
	PlanHash   string    `json:"plan_hash"`
	OperatorID string    `json:"operator_id"`
	TargetIDs  []string  `json:"target_ids"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Signature  string    `json:"signature"`
}

type FleetSnapshotV2 struct {
	SchemaVersion   int                  `json:"schema_version"`
	FleetVersion    int64                `json:"fleet_version"`
	SimulationRate  int                  `json:"simulation_rate"`
	SimulationTick  int64                `json:"simulation_tick_ms"`
	GeneratedAt     time.Time            `json:"generated_at"`
	Vessels         []VesselProfileV2    `json:"vessels"`
	SurfaceContacts []SurfaceContactV2   `json:"surface_contacts"`
	Groups          []OperationalGroupV2 `json:"groups"`
	Collections     []SavedCollectionV2  `json:"collections"`
	Missions        []MissionWorkspaceV2 `json:"missions"`
	Environment     EnvironmentV2        `json:"environment"`
	Map             map[string]any       `json:"map"`
}

type VoiceV2 struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Default   bool   `json:"default"`
	Available bool   `json:"available"`
}

type SpeechCapabilitiesV2 struct {
	TTSNode             string   `json:"tts_node"`
	TTSEngine           string   `json:"tts_engine"`
	TTSVersion          string   `json:"tts_version"`
	DefaultVoice        string   `json:"default_voice"`
	Streaming           bool     `json:"streaming"`
	BargeIn             bool     `json:"barge_in"`
	TranscriptionRoutes []string `json:"transcription_routes"`
	HTTPSRequired       bool     `json:"https_required_for_browser"`
	DemoLimitations     []string `json:"demo_limitations"`
}
