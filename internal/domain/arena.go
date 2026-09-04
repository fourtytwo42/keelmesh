package domain

import "time"

type ArenaMutationV1 struct {
	RequestID       string `json:"request_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
}

type ArenaNodeV1 struct {
	ID                  string     `json:"id"`
	Faction             string     `json:"faction"`
	VesselID            string     `json:"vessel_id"`
	PlannedVMID         int        `json:"planned_vm_id"`
	PlannedManagementIP string     `json:"planned_management_ip"`
	Host                string     `json:"host"`
	Role                string     `json:"role"`
	Status              string     `json:"status"`
	RadioState          string     `json:"radio_state"`
	ManagementConnected bool       `json:"management_connected"`
	InferenceConnected  bool       `json:"inference_connected"`
	Provider            string     `json:"provider"`
	Position            GeoPointV2 `json:"position"`
	HeadingDeg          float64    `json:"heading_deg"`
	SpeedMPS            float64    `json:"speed_mps"`
	BatteryKWh          float64    `json:"battery_kwh"`
	BatteryCapacityKWh  float64    `json:"battery_capacity_kwh"`
	SolarKW             float64    `json:"solar_kw"`
	Hull                int        `json:"hull"`
	HullMaximum         int        `json:"hull_maximum"`
	Class               string     `json:"class"`
	Equipment           []string   `json:"equipment"`
	TapeDepthSeconds    int        `json:"tape_depth_seconds"`
	PNTIntegrity        string     `json:"pnt_integrity"`
	NavigationSource    string     `json:"navigation_source"`
	GNSSState           string     `json:"gnss_state"`
	GNSSAccepted        bool       `json:"gnss_accepted"`
	UncertaintyM        float64    `json:"uncertainty_m"`
}

type CoordinatorV1 struct {
	Faction        string `json:"faction"`
	NodeID         string `json:"node_id"`
	Epoch          int64  `json:"epoch"`
	Votes          int    `json:"votes"`
	QuorumRequired int    `json:"quorum_required"`
	State          string `json:"state"`
	FailoverCount  int    `json:"failover_count"`
	RecoveryMS     int64  `json:"recovery_ms"`
}

type ContactTrackV1 struct {
	ID             string     `json:"id"`
	Faction        string     `json:"faction"`
	Classification string     `json:"classification"`
	Hostility      string     `json:"hostility"`
	Position       GeoPointV2 `json:"position"`
	HeadingDeg     float64    `json:"heading_deg"`
	SpeedMPS       float64    `json:"speed_mps"`
	Confidence     float64    `json:"confidence"`
	UncertaintyM   float64    `json:"uncertainty_m"`
	Source         string     `json:"source"`
	LastSeenTick   int64      `json:"last_seen_tick"`
	State          string     `json:"state"`
}

type WorkspaceActionV1 struct {
	ID             string         `json:"id"`
	SessionID      string         `json:"session_id"`
	Faction        string         `json:"faction"`
	Kind           string         `json:"kind"`
	AuthorityClass string         `json:"authority_class"`
	Arguments      map[string]any `json:"arguments"`
	State          string         `json:"state"`
	ResultHash     string         `json:"result_hash"`
	At             time.Time      `json:"at"`
}

type AgentSessionV1 struct {
	ID               string              `json:"id"`
	Faction          string              `json:"faction"`
	State            string              `json:"state"`
	CoordinatorID    string              `json:"coordinator_id"`
	CoordinatorEpoch int64               `json:"coordinator_epoch"`
	Message          string              `json:"message"`
	Actions          []WorkspaceActionV1 `json:"actions"`
	AwaitingApproval bool                `json:"awaiting_approval"`
}

type EngagementPlanV1 struct {
	ID               string   `json:"id"`
	Faction          string   `json:"faction"`
	CoordinatorEpoch int64    `json:"coordinator_epoch"`
	FriendlyNodeIDs  []string `json:"friendly_node_ids"`
	TargetTrackIDs   []string `json:"target_track_ids"`
	Equipment        []string `json:"equipment"`
	MaximumEffects   int      `json:"maximum_effects"`
	StartsTick       int64    `json:"starts_tick"`
	ExpiresTick      int64    `json:"expires_tick"`
	Summary          string   `json:"summary"`
	ContentHash      string   `json:"content_hash"`
	PolicyStatus     string   `json:"policy_status"`
}

type EngagementLeaseV1 struct {
	ID               string `json:"id"`
	PlanID           string `json:"plan_id"`
	PlanHash         string `json:"plan_hash"`
	Faction          string `json:"faction"`
	CoordinatorEpoch int64  `json:"coordinator_epoch"`
	OperatorID       string `json:"operator_id"`
	RemainingEffects int    `json:"remaining_effects"`
	ExpiresTick      int64  `json:"expires_tick"`
	Signature        string `json:"signature"`
}

type ArenaEffectV1 struct {
	ID            string `json:"id"`
	LeaseID       string `json:"lease_id"`
	PlanID        string `json:"plan_id"`
	Faction       string `json:"faction"`
	TargetTrackID string `json:"target_track_id"`
	Equipment     string `json:"equipment"`
	Outcome       string `json:"outcome"`
	RemainingUses int    `json:"remaining_uses"`
	WorldTick     int64  `json:"world_tick"`
	ReceiptHash   string `json:"receipt_hash"`
}

type ArenaEventV1 struct {
	Sequence int64          `json:"sequence"`
	Tick     int64          `json:"tick"`
	Kind     string         `json:"kind"`
	Faction  string         `json:"faction,omitempty"`
	Summary  string         `json:"summary"`
	Details  map[string]any `json:"details,omitempty"`
}

type KnowledgeSnapshotV1 struct {
	Faction     string           `json:"faction"`
	FriendlyIDs []string         `json:"friendly_ids"`
	Contacts    []ContactTrackV1 `json:"contacts"`
	Checksum    string           `json:"checksum"`
}

type ArenaSnapshotV1 struct {
	SchemaVersion        int                 `json:"schema_version"`
	StateVersion         int64               `json:"state_version"`
	MatchID              string              `json:"match_id"`
	Mode                 string              `json:"mode"`
	Phase                string              `json:"phase"`
	SimulationRate       int                 `json:"simulation_rate"`
	MissionTick          int64               `json:"mission_tick"`
	SimulatedTime        string              `json:"simulated_time"`
	ViewerFaction        string              `json:"viewer_faction"`
	Credits              map[string]int      `json:"credits"`
	Nodes                []ArenaNodeV1       `json:"nodes"`
	Coordinators         []CoordinatorV1     `json:"coordinators"`
	Knowledge            KnowledgeSnapshotV1 `json:"knowledge"`
	WorkspaceActions     []WorkspaceActionV1 `json:"workspace_actions"`
	EngagementPlan       *EngagementPlanV1   `json:"engagement_plan,omitempty"`
	EngagementLease      *EngagementLeaseV1  `json:"engagement_lease,omitempty"`
	Events               []ArenaEventV1      `json:"events"`
	ManagementPlane      string              `json:"management_plane"`
	InferencePlane       string              `json:"inference_plane"`
	RadioPlane           string              `json:"radio_plane"`
	ProvisioningState    string              `json:"provisioning_state"`
	ProvisioningBlockers []string            `json:"provisioning_blockers"`
}
