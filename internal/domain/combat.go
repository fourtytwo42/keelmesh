package domain

import "time"

// CombatProfileV1 contains fictional simulation balance values. It is not a
// description of a real vessel or weapon system.
type CombatProfileV1 struct {
	EntityID       string           `json:"entity_id"`
	Class          string           `json:"class"`
	HullMaximum    float64          `json:"hull_maximum"`
	Armor          string           `json:"armor"`
	Hostility      string           `json:"hostility"`
	Controlled     bool             `json:"controlled"`
	FireRisk       bool             `json:"fire_risk"`
	Weapons        []WeaponSystemV1 `json:"weapons"`
	RecoveryPoint  GeoPointV2       `json:"recovery_point"`
	RecoveryDelayS int64            `json:"recovery_delay_seconds"`
}

type WeaponSystemV1 struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	BaseDamage      float64 `json:"base_damage"`
	EffectiveRangeM float64 `json:"effective_range_m"`
	ReloadSeconds   int64   `json:"reload_seconds"`
	Accuracy        float64 `json:"accuracy"`
	Ammunition      int     `json:"ammunition"`
}

type DamageStateV1 struct {
	Hull                   float64 `json:"hull"`
	HullMaximum            float64 `json:"hull_maximum"`
	IntegrityPercent       float64 `json:"integrity_percent"`
	PropulsionPercent      float64 `json:"propulsion_percent"`
	SensorsPercent         float64 `json:"sensors_percent"`
	WeaponsPercent         float64 `json:"weapons_percent"`
	LastDamageTickMS       int64   `json:"last_damage_tick_ms"`
	LastRegenerationTickMS int64   `json:"last_regeneration_tick_ms"`
	Disabled               bool    `json:"disabled"`
	Sunk                   bool    `json:"sunk"`
}

type CombatEntityStateV1 struct {
	SchemaVersion       int               `json:"schema_version"`
	EntityID            string            `json:"entity_id"`
	BoatID              string            `json:"boat_id"`
	Name                string            `json:"name"`
	EntityKind          string            `json:"entity_kind"`
	Position            GeoPointV2        `json:"position"`
	HeadingDeg          float64           `json:"heading_deg"`
	SpeedMPS            float64           `json:"speed_mps"`
	Profile             CombatProfileV1   `json:"profile"`
	Damage              DamageStateV1     `json:"damage"`
	BehaviorState       string            `json:"behavior_state"`
	Armed               bool              `json:"armed"`
	ArmStateSource      string            `json:"arm_state_source,omitempty"`
	ArmedByMissionID    string            `json:"armed_by_mission_id,omitempty"`
	LastAttackerID      string            `json:"last_attacker_id,omitempty"`
	LastAttackedTickMS  int64             `json:"last_attacked_tick_ms,omitempty"`
	CurrentTargetID     string            `json:"current_target_id,omitempty"`
	ActiveEngagementID  string            `json:"active_engagement_id,omitempty"`
	LastDecision        *RaiderDecisionV1 `json:"last_decision,omitempty"`
	WeaponReadyAtTickMS map[string]int64  `json:"weapon_ready_at_tick_ms"`
	RepairReadyAtTickMS int64             `json:"repair_ready_at_tick_ms"`
	Respawn             RespawnStateV1    `json:"respawn"`
	SpawnGeneration     int64             `json:"spawn_generation"`
	RouteHistory        [][]GeoPointV2    `json:"route_history,omitempty"`
	RouteHistoryTicks   []int64           `json:"route_history_ticks,omitempty"`
	RandomSeed          int64             `json:"random_seed"`
	StateVersion        int64             `json:"state_version"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// EngagementPolicyV1 is the operator-authored rules-of-engagement envelope
// carried by a mission. Arming permits self-defense; initiating fire still
// requires a confirmed mission or engagement whose policy names the target.
type EngagementPolicyV1 struct {
	Enabled                    bool     `json:"enabled"`
	TargetScope                string   `json:"target_scope"`
	DesignatedTargetIDs        []string `json:"designated_target_ids,omitempty"`
	AutoArm                    bool     `json:"auto_arm"`
	ReturnFire                 bool     `json:"return_fire"`
	RequireTargetInMissionArea bool     `json:"require_target_in_mission_area"`
	MaximumRangeM              float64  `json:"maximum_range_m"`
	MaximumEffects             int      `json:"maximum_effects"`
	DurationSeconds            int64    `json:"duration_seconds"`
	DisengageHullPercent       float64  `json:"disengage_hull_percent"`
}

type EngagementProgramV1 struct {
	SchemaVersion        int           `json:"schema_version"`
	ID                   string        `json:"id"`
	RequestID            string        `json:"request_id"`
	IdempotencyKey       string        `json:"idempotency_key"`
	TargetID             string        `json:"target_id"`
	EligibleTargetIDs    []string      `json:"eligible_target_ids"`
	ParticipantIDs       []string      `json:"participant_ids"`
	AllowedWeaponIDs     []string      `json:"allowed_weapon_ids"`
	MaximumRangeM        float64       `json:"maximum_range_m"`
	MaximumEffects       int           `json:"maximum_effects"`
	EffectsApplied       int           `json:"effects_applied"`
	DurationSeconds      int64         `json:"duration_seconds"`
	IssuedTickMS         int64         `json:"issued_tick_ms"`
	ExpiresTickMS        int64         `json:"expires_tick_ms"`
	DisengageHullPercent float64       `json:"disengage_hull_percent"`
	MinimumReserve       float64       `json:"minimum_reserve"`
	Status               string        `json:"status"`
	ContentHash          string        `json:"content_hash"`
	OperatorID           string        `json:"operator_id,omitempty"`
	MissionID            string        `json:"mission_id,omitempty"`
	TargetScope          string        `json:"target_scope"`
	ReturnFire           bool          `json:"return_fire"`
	RequireTargetInArea  bool          `json:"require_target_in_mission_area"`
	OperatingAreas       [][][]float64 `json:"operating_areas,omitempty"`
	AutoArmedIDs         []string      `json:"auto_armed_ids,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	AuthorizedAt         *time.Time    `json:"authorized_at,omitempty"`
}

type CombatEffectV1 struct {
	SchemaVersion       int       `json:"schema_version"`
	ID                  string    `json:"id"`
	EngagementID        string    `json:"engagement_id"`
	ShotSequence        int64     `json:"shot_sequence"`
	SourceID            string    `json:"source_id"`
	TargetID            string    `json:"target_id"`
	WeaponID            string    `json:"weapon_id"`
	RangeM              float64   `json:"range_m"`
	Hit                 bool      `json:"hit"`
	Damage              float64   `json:"damage"`
	Component           string    `json:"component,omitempty"`
	TargetHullRemaining float64   `json:"target_hull_remaining"`
	WorldTickMS         int64     `json:"world_tick_ms"`
	EffectHash          string    `json:"effect_hash"`
	CreatedAt           time.Time `json:"created_at"`
}

type ProjectileEventV1 struct {
	EventID       string     `json:"event_id"`
	EffectID      string     `json:"effect_id"`
	SourceID      string     `json:"source_id"`
	TargetID      string     `json:"target_id"`
	Kind          string     `json:"kind"`
	Start         GeoPointV2 `json:"start"`
	End           GeoPointV2 `json:"end"`
	Hit           bool       `json:"hit"`
	Damage        float64    `json:"damage"`
	WorldTickMS   int64      `json:"world_tick_ms"`
	DisplayTimeMS int64      `json:"display_time_ms"`
	CreatedAt     time.Time  `json:"created_at"`
}

type RepairReceiptV1 struct {
	SchemaVersion      int       `json:"schema_version"`
	ID                 string    `json:"id"`
	VesselID           string    `json:"vessel_id"`
	ActorIdentity      string    `json:"actor_identity"`
	RestoredHull       float64   `json:"restored_hull"`
	HullAfter          float64   `json:"hull_after"`
	ClearedComponent   string    `json:"cleared_component,omitempty"`
	WorldTickMS        int64     `json:"world_tick_ms"`
	ReadyAtTickMS      int64     `json:"ready_at_tick_ms"`
	IdempotencyKey     string    `json:"idempotency_key"`
	ResultingStateHash string    `json:"resulting_state_hash"`
	CreatedAt          time.Time `json:"created_at"`
}

type RespawnStateV1 struct {
	Status       string `json:"status"`
	SunkAtTickMS int64  `json:"sunk_at_tick_ms,omitempty"`
	DueAtTickMS  int64  `json:"due_at_tick_ms,omitempty"`
	WreckUntilMS int64  `json:"wreck_until_tick_ms,omitempty"`
}

type RaiderDecisionV1 struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	TargetID    string     `json:"target_id,omitempty"`
	Destination GeoPointV2 `json:"destination"`
	Reason      string     `json:"reason"`
	WorldTickMS int64      `json:"world_tick_ms"`
	Seed        int64      `json:"seed"`
}

type CombatEventV1 struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	EntityID    string         `json:"entity_id"`
	TargetID    string         `json:"target_id,omitempty"`
	WorldTickMS int64          `json:"world_tick_ms"`
	Payload     map[string]any `json:"payload,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type CombatInterceptEstimateV1 struct {
	ParticipantID       string  `json:"participant_id"`
	TargetID            string  `json:"target_id"`
	DistanceM           float64 `json:"distance_m"`
	ParticipantSpeedMPS float64 `json:"participant_speed_mps"`
	TargetSpeedMPS      float64 `json:"target_speed_mps"`
	EstimatedSeconds    float64 `json:"estimated_seconds"`
	Feasible            bool    `json:"feasible"`
}

type CombatSnapshotV1 struct {
	SchemaVersion int                         `json:"schema_version"`
	StateVersion  int64                       `json:"state_version"`
	WorldTickMS   int64                       `json:"world_tick_ms"`
	Entities      []CombatEntityStateV1       `json:"entities"`
	Engagements   []EngagementProgramV1       `json:"engagements"`
	Projectiles   []ProjectileEventV1         `json:"projectiles"`
	RecentEffects []CombatEffectV1            `json:"recent_effects"`
	Intercepts    []CombatInterceptEstimateV1 `json:"intercept_estimates"`
	GeneratedAt   time.Time                   `json:"generated_at"`
	Disclaimer    string                      `json:"disclaimer"`
}
