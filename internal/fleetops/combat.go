package fleetops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const blackwakeID = "HOSTILE-0001"
const blackwakeMaximumSpeedMPS = 2.7

var blackwakePatrol = []domain.GeoPointV2{
	{-71.78, 40.93}, {-71.48, 40.88}, {-71.12, 40.92}, {-70.78, 41.02},
	{-70.86, 41.20}, {-71.14, 41.12}, {-71.48, 41.04}, {-71.78, 40.93},
}

type CombatEngagementRequest struct {
	Mutation
	TargetID        string   `json:"target_id"`
	ParticipantIDs  []string `json:"participant_ids"`
	DurationSeconds int64    `json:"duration_seconds,omitempty"`
	MaximumEffects  int      `json:"maximum_effects,omitempty"`
	MissionID       string   `json:"mission_id,omitempty"`
}

type CombatAuthorizeRequest struct {
	Mutation
	PlanHash   string `json:"plan_hash"`
	OperatorID string `json:"operator_id"`
}

type CombatRepairRequest struct {
	Mutation
	VesselID string `json:"vessel_id,omitempty"`
}

type CombatArmRequest struct {
	Mutation
	Armed bool `json:"armed"`
}

func controlledCombatProfile(vessel domain.VesselProfileV2) domain.CombatProfileV1 {
	profile := domain.CombatProfileV1{EntityID: vessel.ID, Class: vessel.Class.ID, Armor: "light", Hostility: "friendly", Controlled: true, RecoveryPoint: vessel.Telemetry.Position, RecoveryDelayS: 300, Weapons: []domain.WeaponSystemV1{}}
	switch vessel.Class.ID {
	case "kestrel":
		profile.HullMaximum = 110
		profile.Weapons = []domain.WeaponSystemV1{{ID: "remote-cannon", Name: "Remote cannon", Kind: "cannon", BaseDamage: 8, EffectiveRangeM: 650, ReloadSeconds: 10, Accuracy: .78, Ammunition: -1}}
	case "mariner":
		profile.HullMaximum, profile.Armor = 150, "medium"
		profile.Weapons = []domain.WeaponSystemV1{{ID: "remote-cannon", Name: "Remote cannon", Kind: "cannon", BaseDamage: 12, EffectiveRangeM: 900, ReloadSeconds: 14, Accuracy: .8, Ammunition: -1}}
	default:
		profile.HullMaximum, profile.Armor = 190, "medium"
		profile.Weapons = []domain.WeaponSystemV1{{ID: "defensive-system", Name: "Defensive system", Kind: "cannon", BaseDamage: 14, EffectiveRangeM: 1100, ReloadSeconds: 18, Accuracy: .82, Ammunition: -1}}
	}
	return profile
}

func contactCombatProfile(contact domain.SurfaceContactV2) domain.CombatProfileV1 {
	profile := domain.CombatProfileV1{EntityID: contact.ID, Class: contact.Class, Armor: "light", Hostility: "neutral", RecoveryPoint: contact.Position, RecoveryDelayS: 180, Weapons: []domain.WeaponSystemV1{}}
	switch contact.Class {
	case "yacht":
		profile.HullMaximum = 40
	case "trawler":
		profile.HullMaximum = 70
	case "ferry":
		profile.HullMaximum, profile.Armor = 140, "medium"
	case "container":
		profile.HullMaximum, profile.Armor = 260, "heavy"
	case "tanker":
		profile.HullMaximum, profile.Armor, profile.FireRisk = 320, "heavy", true
	case "patrol":
		profile.HullMaximum, profile.Armor = 190, "medium"
		profile.Weapons = []domain.WeaponSystemV1{{ID: "deck-system", Name: "Defensive deck system", Kind: "cannon", BaseDamage: 11, EffectiveRangeM: 900, ReloadSeconds: 16, Accuracy: .76, Ammunition: -1}}
	default:
		profile.HullMaximum = 70
	}
	return profile
}

func newCombatEntity(id, boatID, name, kind string, position domain.GeoPointV2, profile domain.CombatProfileV1, seed int64) domain.CombatEntityStateV1 {
	now := time.Now().UTC()
	return domain.CombatEntityStateV1{
		SchemaVersion: 1, EntityID: id, BoatID: boatID, Name: name, EntityKind: kind,
		Position: position, Profile: profile, BehaviorState: "operational", Armed: !profile.Controlled && len(profile.Weapons) > 0,
		Damage:              domain.DamageStateV1{Hull: profile.HullMaximum, HullMaximum: profile.HullMaximum, IntegrityPercent: 100, PropulsionPercent: 100, SensorsPercent: 100, WeaponsPercent: 100},
		WeaponReadyAtTickMS: map[string]int64{}, Respawn: domain.RespawnStateV1{Status: "active"}, SpawnGeneration: 1,
		RandomSeed: seed, StateVersion: 1, UpdatedAt: now,
	}
}

func (m *Manager) initializeCombatLocked() {
	for _, vessel := range m.vessels {
		if _, exists := m.combatEntities[vessel.ID]; exists {
			continue
		}
		profile := controlledCombatProfile(vessel)
		m.combatEntities[vessel.ID] = newCombatEntity(vessel.ID, vessel.Designation, vessel.Callsign, "controlled", vessel.Telemetry.Position, profile, int64(stableSeed(vessel.ID)))
	}
	for _, contact := range surfaceContactsAt(time.UnixMilli(m.simulationEpochMS + m.simTickMS).UTC()) {
		if existing, exists := m.combatEntities[contact.ID]; exists {
			if len(existing.Profile.Weapons) > 0 && !existing.Armed {
				existing.Armed, existing.ArmStateSource = true, "npc_self_defense"
				m.combatEntities[contact.ID] = existing
			}
			continue
		}
		profile := contactCombatProfile(contact)
		for _, spec := range surfaceTraffic {
			if spec.ID == contact.ID && len(spec.Route) > 0 {
				profile.RecoveryPoint = spec.Route[0]
				break
			}
		}
		m.combatEntities[contact.ID] = newCombatEntity(contact.ID, contact.BoatID, contact.Name, "contact", contact.Position, profile, int64(stableSeed(contact.ID)))
	}
	if _, exists := m.combatEntities[blackwakeID]; !exists {
		profile := domain.CombatProfileV1{EntityID: blackwakeID, Class: "pirate-raider", HullMaximum: 170, Armor: "medium", Hostility: "hostile", RecoveryPoint: blackwakePatrol[0], RecoveryDelayS: 300, Weapons: []domain.WeaponSystemV1{
			{ID: "twin-deck-cannons", Name: "Twin deck cannons", Kind: "cannon", BaseDamage: 14, EffectiveRangeM: 750, ReloadSeconds: 12, Accuracy: .74, Ammunition: -1},
			{ID: "limited-rockets", Name: "Limited rockets", Kind: "rocket", BaseDamage: 24, EffectiveRangeM: 1400, ReloadSeconds: 45, Accuracy: .66, Ammunition: 8},
		}}
		blackwake := newCombatEntity(blackwakeID, blackwakeID, "Blackwake", "hostile", blackwakePatrol[0], profile, 15001337)
		blackwake.BehaviorState = "roam"
		blackwake.RouteHistory = [][]domain.GeoPointV2{{blackwakePatrol[0], blackwakePatrol[1]}}
		blackwake.RouteHistoryTicks = []int64{m.simTickMS}
		blackwake.LastDecision = &domain.RaiderDecisionV1{ID: "raider-initial", State: "roam", Destination: blackwakePatrol[1], Reason: "water-safe offshore patrol", Seed: blackwake.RandomSeed}
		m.combatEntities[blackwakeID] = blackwake
	} else {
		blackwake := m.combatEntities[blackwakeID]
		blackwake.Armed, blackwake.ArmStateSource = true, "hostile_autonomy"
		m.combatEntities[blackwakeID] = blackwake
	}
}

func cloneCombatEntity(value *domain.CombatEntityStateV1) *domain.CombatEntityStateV1 {
	if value == nil || value.EntityID == "" {
		return nil
	}
	copy := *value
	copy.Profile.Weapons = append(make([]domain.WeaponSystemV1, 0, len(value.Profile.Weapons)), value.Profile.Weapons...)
	copy.WeaponReadyAtTickMS = map[string]int64{}
	for key, tick := range value.WeaponReadyAtTickMS {
		copy.WeaponReadyAtTickMS[key] = tick
	}
	return &copy
}

func blackwakeContact(state domain.CombatEntityStateV1, now time.Time) domain.SurfaceContactV2 {
	navigation := "hostile raider · " + state.BehaviorState
	if state.Damage.Sunk {
		navigation = "sunk · respawn pending"
	}
	return domain.SurfaceContactV2{ID: state.EntityID, BoatID: state.BoatID, Name: state.Name, Callsign: "BLACKWAKE", Class: "pirate-raider", Activity: "persistent hostile raider · fictional simulation", ColorName: "hostile red", Color: "#e3544f", Position: state.Position, HeadingDeg: state.HeadingDeg, SpeedMPS: state.SpeedMPS, SpeedKnots: state.SpeedMPS * 1.94384, LengthM: 46, DraftM: 4.8, NavigationState: navigation, RouteName: "adaptive offshore hunt", Route: []domain.GeoPointV2{}, Looping: false, Hostility: "hostile", Combat: cloneCombatEntity(&state), UpdatedAt: now}
}

func (m *Manager) combatSnapshotLocked() domain.CombatSnapshotV1 {
	entities := make([]domain.CombatEntityStateV1, 0, len(m.combatEntities))
	for _, entity := range m.combatEntities {
		entity.Profile.Weapons = append(make([]domain.WeaponSystemV1, 0, len(entity.Profile.Weapons)), entity.Profile.Weapons...)
		entities = append(entities, entity)
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].BoatID < entities[j].BoatID })
	engagements := make([]domain.EngagementProgramV1, 0, len(m.combatEngagements))
	for _, engagement := range m.combatEngagements {
		engagements = append(engagements, engagement)
	}
	sort.Slice(engagements, func(i, j int) bool { return engagements[i].CreatedAt.After(engagements[j].CreatedAt) })
	intercepts := make([]domain.CombatInterceptEstimateV1, 0)
	for _, participant := range entities {
		if !participant.Profile.Controlled || participant.Damage.Disabled || participant.Damage.Sunk {
			continue
		}
		vessel, exists := m.vessels[participant.EntityID]
		if !exists {
			continue
		}
		for _, target := range entities {
			if target.Profile.Hostility != "hostile" || target.Damage.Sunk {
				continue
			}
			distance := math.Max(0, combatDistanceM(participant.Position, target.Position)-maxWeaponRange(participant.Profile.Weapons))
			participantSpeed := vessel.Class.MaxSpeedMPS * math.Max(.35, participant.Damage.PropulsionPercent/100)
			closingSpeed := participantSpeed - target.SpeedMPS
			feasible := distance == 0 || closingSpeed > .05
			seconds := 0.0
			if distance > 0 && feasible {
				seconds = distance / closingSpeed
			}
			intercepts = append(intercepts, domain.CombatInterceptEstimateV1{ParticipantID: participant.EntityID, TargetID: target.EntityID, DistanceM: distance, ParticipantSpeedMPS: participantSpeed, TargetSpeedMPS: target.SpeedMPS, EstimatedSeconds: seconds, Feasible: feasible})
		}
	}
	return domain.CombatSnapshotV1{SchemaVersion: 1, StateVersion: m.combatVersion, WorldTickMS: m.simTickMS, Entities: entities, Engagements: engagements, Projectiles: append([]domain.ProjectileEventV1(nil), m.combatProjectiles...), RecentEffects: append([]domain.CombatEffectV1(nil), m.combatEffects...), Intercepts: intercepts, GeneratedAt: time.Now().UTC(), Disclaimer: "Fictional deterministic combat simulation — not a real weapon-control system."}
}

func (m *Manager) CombatSnapshot() domain.CombatSnapshotV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.combatSnapshotLocked()
}

func (m *Manager) CombatEntity(id string) (domain.CombatEntityStateV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, entity := range m.combatEntities {
		if strings.EqualFold(entity.EntityID, id) || strings.EqualFold(entity.BoatID, id) || strings.EqualFold(entity.Name, id) {
			return entity, nil
		}
	}
	return domain.CombatEntityStateV1{}, &Error{"COMBAT_ENTITY_NOT_FOUND", "Combat entity not found."}
}

func (m *Manager) PlanCombatEngagement(req CombatEngagementRequest) (domain.EngagementProgramV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.EngagementProgramV1{}, &Error{"INVALID_REQUEST", "request_id and idempotency_key are required."}
	}
	target, ok := m.combatEntities[req.TargetID]
	if !ok {
		return domain.EngagementProgramV1{}, &Error{"COMBAT_ENTITY_NOT_FOUND", "Target not found."}
	}
	if target.Profile.Controlled {
		return domain.EngagementProgramV1{}, &Error{"TARGET_NOT_HOSTILE", "A controlled Fleet vessel cannot be designated as an engagement target."}
	}
	participants := uniqueStrings(req.ParticipantIDs)
	weapons, maxRange := []string{}, 0.0
	for _, id := range participants {
		entity, exists := m.combatEntities[id]
		if !exists || !entity.Profile.Controlled || entity.Damage.Disabled || entity.Damage.Sunk {
			return domain.EngagementProgramV1{}, &Error{"WEAPON_UNAVAILABLE", "Every participant must be an operational controlled vessel."}
		}
		for _, weapon := range entity.Profile.Weapons {
			weapons = append(weapons, id+":"+weapon.ID)
			maxRange = math.Max(maxRange, weapon.EffectiveRangeM)
		}
	}
	if len(participants) == 0 || len(weapons) == 0 {
		return domain.EngagementProgramV1{}, &Error{"WEAPON_UNAVAILABLE", "Select at least one operational, weapon-capable controlled vessel."}
	}
	duration := req.DurationSeconds
	if duration <= 0 {
		duration = 900
	}
	if duration > 3600 {
		duration = 3600
	}
	maximumEffects := req.MaximumEffects
	if maximumEffects <= 0 {
		maximumEffects = len(participants) * 80
	}
	program := domain.EngagementProgramV1{SchemaVersion: 1, ID: "engagement-" + shortHash(req.IdempotencyKey), RequestID: req.RequestID, IdempotencyKey: req.IdempotencyKey, TargetID: target.EntityID, EligibleTargetIDs: []string{target.EntityID}, ParticipantIDs: participants, AllowedWeaponIDs: weapons, MaximumRangeM: maxRange, MaximumEffects: maximumEffects, DurationSeconds: duration, IssuedTickMS: m.simTickMS, ExpiresTickMS: m.simTickMS + duration*1000, DisengageHullPercent: 20, MinimumReserve: .2, Status: "pending_approval", MissionID: req.MissionID, TargetScope: "designated", ReturnFire: true, CreatedAt: time.Now().UTC()}
	program.ContentHash = engagementContentHash(program)
	if existingID, exists := m.combatIdempotency[req.IdempotencyKey]; exists {
		existing := m.combatEngagements[existingID]
		if existing.ContentHash != program.ContentHash {
			return domain.EngagementProgramV1{}, &Error{"COMBAT_STATE_STALE", "Idempotency key was already used for different engagement content."}
		}
		return existing, nil
	}
	m.combatEngagements[program.ID], m.combatIdempotency[req.IdempotencyKey] = program, program.ID
	m.combatVersion++
	m.recordCombatEventLocked("engagement_planned", program.ID, target.EntityID, map[string]any{"hash": program.ContentHash, "participants": participants})
	m.persistCombatAsync(true)
	return program, nil
}

func (m *Manager) AuthorizeCombatEngagement(id string, req CombatAuthorizeRequest) (domain.EngagementProgramV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" || req.OperatorID == "" {
		return domain.EngagementProgramV1{}, &Error{"INVALID_REQUEST", "request_id, idempotency_key, and operator_id are required."}
	}
	program, ok := m.combatEngagements[id]
	if !ok {
		return program, &Error{"ENGAGEMENT_NOT_FOUND", "Engagement not found."}
	}
	if req.PlanHash == "" || req.PlanHash != program.ContentHash {
		return program, &Error{"COMBAT_STATE_STALE", "Exact engagement hash does not match."}
	}
	if program.Status == "active" {
		return program, nil
	}
	if program.Status != "pending_approval" {
		return program, &Error{"ENGAGEMENT_EXPIRED", "Engagement can no longer be authorized."}
	}
	now := time.Now().UTC()
	program.Status, program.OperatorID, program.AuthorizedAt = "active", req.OperatorID, &now
	program.IssuedTickMS, program.ExpiresTickMS = m.simTickMS, m.simTickMS+program.DurationSeconds*1000
	m.combatEngagements[id] = program
	for _, participant := range program.ParticipantIDs {
		entity := m.combatEntities[participant]
		if !entity.Armed {
			program.AutoArmedIDs = append(program.AutoArmedIDs, participant)
		}
		entity.Armed, entity.ArmStateSource = true, "engagement"
		entity.ActiveEngagementID = id
		entity.CurrentTargetID = program.TargetID
		entity.BehaviorState = "engage"
		entity.StateVersion++
		m.combatEntities[participant] = entity
	}
	m.combatVersion++
	m.combatEngagements[id] = program
	m.recordCombatEventLocked("engagement_authorized", id, program.TargetID, map[string]any{"hash": program.ContentHash, "operator": req.OperatorID})
	m.persistCombatAsync(true)
	return program, nil
}

func (m *Manager) ArmCombatVessel(id string, req CombatArmRequest) (domain.CombatEntityStateV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.CombatEntityStateV1{}, &Error{"INVALID_REQUEST", "request_id and idempotency_key are required."}
	}
	actionKey := fmt.Sprintf("arm:%s:%t", id, req.Armed)
	if previous, exists := m.combatIdempotency[req.IdempotencyKey]; exists {
		if previous != actionKey {
			return domain.CombatEntityStateV1{}, &Error{"COMBAT_STATE_STALE", "Idempotency key was already used for a different combat action."}
		}
		if entity, ok := m.combatEntities[id]; ok {
			return entity, nil
		}
	}
	entity, ok := m.combatEntities[id]
	if !ok || !entity.Profile.Controlled {
		return domain.CombatEntityStateV1{}, &Error{"ARM_TARGET_DENIED", "Only controlled Fleet vessels may be armed or disarmed."}
	}
	if entity.Damage.Sunk || entity.Damage.Disabled {
		return domain.CombatEntityStateV1{}, &Error{"VESSEL_SUNK", "A disabled vessel cannot change arm state."}
	}
	if !req.Armed && entity.ActiveEngagementID != "" {
		// Disarming is always allowed and immediately prevents further fire;
		// movement may continue until the mission or engagement is stopped.
		entity.BehaviorState = "weapons_safe"
	}
	entity.Armed = req.Armed
	entity.ArmStateSource = map[bool]string{true: "operator", false: "weapons_safe"}[req.Armed]
	if !req.Armed {
		entity.ArmedByMissionID = ""
	}
	entity.StateVersion++
	entity.UpdatedAt = time.Now().UTC()
	m.combatEntities[id] = entity
	m.combatIdempotency[req.IdempotencyKey] = actionKey
	m.combatVersion++
	m.recordCombatEventLocked("arm_state_changed", id, "", map[string]any{"armed": req.Armed, "source": entity.ArmStateSource, "actor": req.ActorIdentity})
	m.persistCombatAsync(true)
	return entity, nil
}

func (m *Manager) StopCombatEngagement(id string, req Mutation) (domain.EngagementProgramV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.EngagementProgramV1{}, &Error{"INVALID_REQUEST", "request_id and idempotency_key are required."}
	}
	program, ok := m.combatEngagements[id]
	if !ok {
		return program, &Error{"ENGAGEMENT_NOT_FOUND", "Engagement not found."}
	}
	if program.Status == "stopped" || program.Status == "completed" {
		return program, nil
	}
	program.Status = "stopped"
	m.combatEngagements[id] = program
	m.releaseEngagementParticipantsLocked(program)
	m.combatVersion++
	m.recordCombatEventLocked("engagement_stopped", id, program.TargetID, nil)
	m.persistCombatAsync(true)
	return program, nil
}

func (m *Manager) RepairCombatVessel(id string, req CombatRepairRequest) (domain.RepairReceiptV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.RepairReceiptV1{}, &Error{"INVALID_REQUEST", "request_id and idempotency_key are required."}
	}
	if existingID, exists := m.combatIdempotency[req.IdempotencyKey]; exists {
		if receipt, ok := m.combatRepairs[existingID]; ok {
			if receipt.VesselID != id {
				return domain.RepairReceiptV1{}, &Error{"COMBAT_STATE_STALE", "Idempotency key was already used for a different repair target."}
			}
			return receipt, nil
		}
		return domain.RepairReceiptV1{}, &Error{"COMBAT_STATE_STALE", "Idempotency key was already used for another combat action."}
	}
	entity, ok := m.combatEntities[id]
	if !ok || !entity.Profile.Controlled {
		return domain.RepairReceiptV1{}, &Error{"REPAIR_TARGET_DENIED", "Repairs are available only for controlled vessels."}
	}
	if entity.Damage.Sunk || entity.Damage.Hull <= 0 {
		return domain.RepairReceiptV1{}, &Error{"VESSEL_SUNK", "A sunk or disabled vessel cannot receive an explicit repair."}
	}
	if m.simTickMS < entity.RepairReadyAtTickMS {
		return domain.RepairReceiptV1{}, &Error{"REPAIR_COOLDOWN_ACTIVE", fmt.Sprintf("Repair is available in %.1f world minutes.", float64(entity.RepairReadyAtTickMS-m.simTickMS)/60000)}
	}
	before := entity.Damage.Hull
	restored := math.Min(entity.Profile.HullMaximum*.20, entity.Profile.HullMaximum-before)
	if restored <= 0 {
		return domain.RepairReceiptV1{}, &Error{"REPAIR_TARGET_DENIED", "Vessel is already at full integrity."}
	}
	entity.Damage.Hull += restored
	entity.Damage.IntegrityPercent = entity.Damage.Hull / entity.Profile.HullMaximum * 100
	cleared := clearOneComponent(&entity.Damage)
	entity.RepairReadyAtTickMS = m.simTickMS + 300000
	entity.StateVersion++
	entity.UpdatedAt = time.Now().UTC()
	m.combatEntities[id] = entity
	receiptID := "repair-" + shortHash(req.IdempotencyKey)
	receipt := domain.RepairReceiptV1{SchemaVersion: 1, ID: receiptID, VesselID: id, ActorIdentity: req.ActorIdentity, RestoredHull: restored, HullAfter: entity.Damage.Hull, ClearedComponent: cleared, WorldTickMS: m.simTickMS, ReadyAtTickMS: entity.RepairReadyAtTickMS, IdempotencyKey: req.IdempotencyKey, ResultingStateHash: combatHash(entity), CreatedAt: time.Now().UTC()}
	m.combatRepairs[receiptID], m.combatIdempotency[req.IdempotencyKey] = receipt, receiptID
	m.combatVersion++
	m.recordCombatEventLocked("repair_applied", id, "", map[string]any{"restored_hull": restored, "ready_at_tick_ms": entity.RepairReadyAtTickMS})
	m.persistCombatAsync(true)
	return receipt, nil
}

func (m *Manager) ResetCombat(req Mutation) (domain.CombatSnapshotV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.CombatSnapshotV1{}, &Error{"INVALID_REQUEST", "request_id and idempotency_key are required."}
	}
	m.combatEntities, m.combatEngagements, m.combatRepairs, m.combatIdempotency = map[string]domain.CombatEntityStateV1{}, map[string]domain.EngagementProgramV1{}, map[string]domain.RepairReceiptV1{}, map[string]string{}
	m.combatEffects, m.combatProjectiles, m.combatEvents = nil, nil, nil
	m.combatSequence = 0
	m.combatVersion++
	m.initializeCombatLocked()
	m.recordCombatEventLocked("combat_reset", "combat", "", nil)
	m.resetCombatPersistenceAsync()
	return m.combatSnapshotLocked(), nil
}

func (m *Manager) advanceCombatLocked(deltaMS int64) {
	if len(m.combatEntities) == 0 {
		m.initializeCombatLocked()
	}
	m.syncCombatPositionsLocked()
	m.respawnAndRegenerateLocked()
	m.advanceBlackwakeLocked(deltaMS)
	m.advanceCommercialEscapeLocked(deltaMS)
	m.advanceEngagementsLocked()
	m.advanceMilitaryDefenseLocked()
	m.advanceControlledSelfDefenseLocked()
	cutoff := time.Now().UTC().Add(-4 * time.Second)
	projectiles := m.combatProjectiles[:0]
	for _, event := range m.combatProjectiles {
		if event.CreatedAt.After(cutoff) {
			projectiles = append(projectiles, event)
		}
	}
	m.combatProjectiles = projectiles
}

func (m *Manager) syncCombatPositionsLocked() {
	for id, vessel := range m.vessels {
		entity := m.combatEntities[id]
		entity.Position, entity.HeadingDeg, entity.SpeedMPS = vessel.Telemetry.Position, vessel.Telemetry.HeadingDeg, vessel.Telemetry.SpeedMPS
		m.combatEntities[id] = entity
	}
	at := time.UnixMilli(m.simulationEpochMS + m.simTickMS).UTC()
	for _, contact := range surfaceContactsAt(at) {
		entity := m.combatEntities[contact.ID]
		if !entity.Damage.Sunk && entity.BehaviorState != "escape" && entity.BehaviorState != "rejoining_route" {
			entity.Position, entity.HeadingDeg, entity.SpeedMPS = contact.Position, contact.HeadingDeg, contact.SpeedMPS
		} else if !entity.Damage.Sunk && entity.BehaviorState == "rejoining_route" {
			speed := combatContactSpeed(contact.ID)
			moveCombatEntity(&entity, contact.Position, speed, .2)
			if combatDistanceM(entity.Position, contact.Position) < 20 {
				entity.BehaviorState = "operational"
			}
		}
		m.combatEntities[contact.ID] = entity
	}
}

func (m *Manager) respawnAndRegenerateLocked() {
	for id, entity := range m.combatEntities {
		if entity.Damage.Sunk || entity.Damage.Disabled {
			if entity.Respawn.DueAtTickMS > 0 && m.simTickMS >= entity.Respawn.DueAtTickMS {
				m.respawnEntityLocked(id, entity)
			}
			continue
		}
		if entity.Damage.Hull >= entity.Profile.HullMaximum || entity.Damage.LastDamageTickMS == 0 || m.simTickMS-entity.Damage.LastDamageTickMS < 60000 {
			continue
		}
		if entity.Damage.LastRegenerationTickMS == 0 {
			entity.Damage.LastRegenerationTickMS = entity.Damage.LastDamageTickMS
		}
		for m.simTickMS-entity.Damage.LastRegenerationTickMS >= 60000 && entity.Damage.Hull < entity.Profile.HullMaximum {
			entity.Damage.Hull = math.Min(entity.Profile.HullMaximum, entity.Damage.Hull+entity.Profile.HullMaximum*.01)
			entity.Damage.LastRegenerationTickMS += 60000
		}
		entity.Damage.IntegrityPercent = entity.Damage.Hull / entity.Profile.HullMaximum * 100
		entity.StateVersion++
		m.combatEntities[id] = entity
	}
}

func (m *Manager) respawnEntityLocked(id string, entity domain.CombatEntityStateV1) {
	entity.SpawnGeneration++
	entity.Damage = domain.DamageStateV1{Hull: entity.Profile.HullMaximum, HullMaximum: entity.Profile.HullMaximum, IntegrityPercent: 100, PropulsionPercent: 100, SensorsPercent: 100, WeaponsPercent: 100}
	entity.Respawn = domain.RespawnStateV1{Status: "active"}
	entity.WeaponReadyAtTickMS, entity.RepairReadyAtTickMS, entity.LastDecision = map[string]int64{}, 0, nil
	entity.CurrentTargetID, entity.ActiveEngagementID, entity.BehaviorState = "", "", "operational"
	if id == blackwakeID {
		for index := range entity.Profile.Weapons {
			if entity.Profile.Weapons[index].ID == "limited-rockets" {
				entity.Profile.Weapons[index].Ammunition = 8
			}
		}
		point := blackwakePatrol[(int(entity.SpawnGeneration)*3)%len(blackwakePatrol)]
		entity.Position, entity.Profile.RecoveryPoint, entity.BehaviorState = point, point, "roam"
	} else if entity.Profile.Controlled {
		entity.Position, entity.BehaviorState = entity.Profile.RecoveryPoint, "safe_hold"
		if vessel, ok := m.vessels[id]; ok {
			vessel.Available = true
			vessel.Telemetry.Position, vessel.Telemetry.SpeedMPS, vessel.Telemetry.Mode, vessel.Telemetry.MissionID, vessel.Telemetry.Route = entity.Position, 0, "recovered · safe hold", "", nil
			vessel.Telemetry.Health = "nominal"
			m.vessels[id] = vessel
		}
	} else {
		entity.Position, entity.BehaviorState, entity.SpeedMPS = entity.Profile.RecoveryPoint, "rejoining_route", 0
	}
	entity.UpdatedAt = time.Now().UTC()
	entity.StateVersion++
	m.combatEntities[id] = entity
	m.combatVersion++
	m.recordCombatEventLocked("entity_respawned", id, "", map[string]any{"spawn_generation": entity.SpawnGeneration})
}

func (m *Manager) advanceBlackwakeLocked(deltaMS int64) {
	raider := m.combatEntities[blackwakeID]
	if raider.Damage.Sunk {
		return
	}
	if raider.Damage.IntegrityPercent < 25 || m.raiderOvermatchedLocked(raider) {
		raider.BehaviorState, raider.CurrentTargetID = "withdraw_repair", ""
		raider.LastDecision = &domain.RaiderDecisionV1{ID: fmt.Sprintf("raider-%d", m.simTickMS), State: "withdraw_repair", Destination: blackwakePatrol[0], Reason: "hull below withdrawal threshold", WorldTickMS: m.simTickMS, Seed: raider.RandomSeed}
	}
	if raider.BehaviorState == "withdraw_repair" && combatDistanceM(raider.Position, blackwakePatrol[0]) < 20 && raider.Damage.IntegrityPercent >= 40 {
		raider.BehaviorState = "roam"
		raider.LastDecision = nil
	}
	if raider.CurrentTargetID == "" && raider.BehaviorState != "withdraw_repair" {
		if target := m.selectRaiderTargetLocked(raider); target != "" {
			raider.CurrentTargetID, raider.BehaviorState = target, "detect"
			targetEntity := m.combatEntities[target]
			raider.LastDecision = &domain.RaiderDecisionV1{ID: fmt.Sprintf("raider-%d", m.simTickMS), State: "detect", TargetID: target, Destination: targetEntity.Position, Reason: "deterministic vulnerable-target score", WorldTickMS: m.simTickMS, Seed: raider.RandomSeed}
			m.recordCombatEventLocked("raider_target_selected", blackwakeID, target, nil)
		}
	}
	if raider.BehaviorState == "detect" && raider.LastDecision != nil && m.simTickMS-raider.LastDecision.WorldTickMS >= 2000 {
		raider.BehaviorState = "stalk"
	}
	destination := m.nextBlackwakeRoamDestinationLocked(&raider)
	if raider.LastDecision != nil {
		destination = raider.LastDecision.Destination
	}
	if target, ok := m.combatEntities[raider.CurrentTargetID]; ok && !target.Damage.Sunk && combatPositionInBounds(target.Position) && (raider.LastDecision == nil || m.simTickMS-raider.LastDecision.WorldTickMS <= 10*60*1000) {
		destination = target.Position
		distance := combatDistanceM(raider.Position, target.Position)
		if raider.BehaviorState == "stalk" && distance > maxWeaponRange(raider.Profile.Weapons) {
			raider.BehaviorState = "intercept"
		}
		if distance <= maxWeaponRange(raider.Profile.Weapons) {
			raider.BehaviorState = "engage"
			m.fireBestWeaponLocked(&raider, &target, "raider-"+fmt.Sprint(raider.SpawnGeneration))
		}
	} else if raider.CurrentTargetID != "" {
		raider.CurrentTargetID, raider.BehaviorState = "", "roam"
		raider.LastDecision = nil
	}
	destination = clampCombatPosition(destination)
	moveCombatEntity(&raider, destination, blackwakeMaximumSpeedMPS*math.Max(.35, raider.Damage.PropulsionPercent/100), float64(deltaMS)/1000)
	raider.StateVersion++
	raider.UpdatedAt = time.Now().UTC()
	m.combatEntities[blackwakeID] = raider
}

func (m *Manager) nextBlackwakeRoamDestinationLocked(raider *domain.CombatEntityStateV1) domain.GeoPointV2 {
	if raider.LastDecision != nil && raider.LastDecision.TargetID == "" && combatDistanceM(raider.Position, raider.LastDecision.Destination) >= 20 {
		return raider.LastDecision.Destination
	}
	cutoff := m.simTickMS - 1800000
	for len(raider.RouteHistoryTicks) > 0 && raider.RouteHistoryTicks[0] < cutoff {
		raider.RouteHistoryTicks = raider.RouteHistoryTicks[1:]
		if len(raider.RouteHistory) > 0 {
			raider.RouteHistory = raider.RouteHistory[1:]
		}
	}
	used := map[string]bool{}
	for _, route := range raider.RouteHistory {
		if len(route) > 1 {
			used[fmt.Sprintf("%.4f,%.4f", route[1][0], route[1][1])] = true
		}
	}
	start := int((uint64(raider.RandomSeed) + uint64(m.simTickMS/60000) + uint64(raider.SpawnGeneration)) % uint64(len(blackwakePatrol)-1))
	destination := blackwakePatrol[start]
	for offset := 0; offset < len(blackwakePatrol)-1; offset++ {
		candidate := blackwakePatrol[(start+offset)%(len(blackwakePatrol)-1)]
		if !used[fmt.Sprintf("%.4f,%.4f", candidate[0], candidate[1])] && combatDistanceM(raider.Position, candidate) > 500 {
			destination = candidate
			break
		}
	}
	raider.RouteHistory = append(raider.RouteHistory, []domain.GeoPointV2{raider.Position, destination})
	raider.RouteHistoryTicks = append(raider.RouteHistoryTicks, m.simTickMS)
	raider.LastDecision = &domain.RaiderDecisionV1{ID: fmt.Sprintf("raider-roam-%d", m.simTickMS), State: "roam", Destination: destination, Reason: "deterministic water-safe varied patrol segment", WorldTickMS: m.simTickMS, Seed: raider.RandomSeed}
	m.recordCombatEventLocked("raider_route_selected", blackwakeID, "", map[string]any{"destination": destination})
	return destination
}

func (m *Manager) raiderOvermatchedLocked(raider domain.CombatEntityStateV1) bool {
	armedNearby := 0
	for _, entity := range m.combatEntities {
		if !entity.Profile.Controlled || len(entity.Profile.Weapons) == 0 || entity.Damage.Sunk || entity.Damage.Disabled {
			continue
		}
		if combatDistanceM(entity.Position, raider.Position) <= 2500 && (entity.CurrentTargetID == blackwakeID || entity.ActiveEngagementID != "") {
			armedNearby++
		}
	}
	return armedNearby >= 3
}

func (m *Manager) advanceCommercialEscapeLocked(deltaMS int64) {
	raider := m.combatEntities[blackwakeID]
	at := time.UnixMilli(m.simulationEpochMS + m.simTickMS).UTC()
	baseline := map[string]domain.SurfaceContactV2{}
	for _, contact := range surfaceContactsAt(at) {
		baseline[contact.ID] = contact
	}
	for id, entity := range m.combatEntities {
		if entity.Profile.Controlled {
			if raider.CurrentTargetID == id && entity.ActiveEngagementID == "" && !entity.Damage.Sunk && combatDistanceM(entity.Position, raider.Position) <= 3000 {
				vessel := m.vessels[id]
				destination := pointAtBearing(entity.Position, combatBearingDeg(raider.Position, entity.Position), 4000)
				moveCombatEntity(&entity, destination, vessel.Class.MaxSpeedMPS*math.Max(.35, entity.Damage.PropulsionPercent/100), float64(deltaMS)/1000)
				entity.BehaviorState = "defensive_evasion"
				vessel.Telemetry.Position, vessel.Telemetry.HeadingDeg, vessel.Telemetry.SpeedMPS, vessel.Telemetry.Mode = entity.Position, entity.HeadingDeg, entity.SpeedMPS, "bounded defensive evasion"
				m.vessels[id], m.combatEntities[id] = vessel, entity
			}
			continue
		}
		if entity.Profile.Hostility != "neutral" || len(entity.Profile.Weapons) > 0 || entity.Damage.Sunk {
			continue
		}
		contact, knownContact := baseline[id]
		if !combatPositionInBounds(entity.Position) {
			// A stale pre-M15 checkpoint may contain an unbounded pursuit. Rejoin
			// at the contact's deterministic water-safe route projection instead
			// of spending simulated days crossing back through unknown water.
			if knownContact {
				entity.Position, entity.HeadingDeg, entity.SpeedMPS = contact.Position, contact.HeadingDeg, contact.SpeedMPS
				entity.BehaviorState = "operational"
				entity.StateVersion++
				m.combatEntities[id] = entity
			}
			continue
		}
		if raider.CurrentTargetID != id {
			if entity.BehaviorState == "escape" {
				entity.BehaviorState = "rejoining_route"
				m.combatEntities[id] = entity
			}
			continue
		}
		if knownContact && combatDistanceM(entity.Position, contact.Position) >= 2500 {
			entity.BehaviorState = "rejoining_route"
			m.combatEntities[id] = entity
			continue
		}
		if combatDistanceM(entity.Position, raider.Position) > 5000 {
			continue
		}
		entity.BehaviorState = "escape"
		away := combatBearingDeg(raider.Position, entity.Position)
		destination := clampCombatPosition(pointAtBearing(entity.Position, away, 2500))
		moveCombatEntity(&entity, destination, math.Min(2.8, combatContactSpeed(id)*1.15), float64(deltaMS)/1000)
		entity.StateVersion++
		m.combatEntities[id] = entity
	}
}

func (m *Manager) selectRaiderTargetLocked(raider domain.CombatEntityStateV1) string {
	bestID, bestScore := "", math.MaxFloat64
	for id, target := range m.combatEntities {
		if id == blackwakeID || target.Profile.Hostility == "hostile" || target.Damage.Sunk || target.Damage.Disabled || !combatPositionInBounds(target.Position) {
			continue
		}
		distance := combatDistanceM(raider.Position, target.Position)
		if distance > 26000 {
			continue
		}
		score := distance + target.Profile.HullMaximum*18
		if len(target.Profile.Weapons) == 0 {
			score -= 9000
		} else {
			score += 8000
		}
		if target.Profile.Controlled {
			score += 5000
		}
		if score < bestScore || (score == bestScore && id < bestID) {
			bestID, bestScore = id, score
		}
	}
	return bestID
}

func (m *Manager) advanceEngagementsLocked() {
	for id, program := range m.combatEngagements {
		if program.Status != "active" {
			continue
		}
		if m.simTickMS >= program.ExpiresTickMS || program.EffectsApplied >= program.MaximumEffects {
			program.Status = "completed"
			m.combatEngagements[id] = program
			m.releaseEngagementParticipantsLocked(program)
			continue
		}
		target := m.combatEntities[program.TargetID]
		if target.Damage.Sunk {
			nextID := m.nextEligibleEngagementTargetLocked(program)
			if nextID == "" {
				program.Status = "completed"
				m.combatEngagements[id] = program
				m.releaseEngagementParticipantsLocked(program)
				continue
			}
			program.TargetID, target = nextID, m.combatEntities[nextID]
			for _, participantID := range program.ParticipantIDs {
				participant := m.combatEntities[participantID]
				participant.CurrentTargetID = nextID
				m.combatEntities[participantID] = participant
			}
			m.recordCombatEventLocked("engagement_target_advanced", id, nextID, nil)
		}
		for _, participantID := range program.ParticipantIDs {
			participant := m.combatEntities[participantID]
			vessel := m.vessels[participantID]
			if !participant.Armed || participant.Damage.Sunk || participant.Damage.Disabled || participant.Damage.IntegrityPercent <= program.DisengageHullPercent || vessel.Telemetry.Reserve <= program.MinimumReserve {
				continue
			}
			if program.RequireTargetInArea && len(program.OperatingAreas) > 0 && !combatPointInAreas(target.Position, program.OperatingAreas) {
				continue
			}
			if combatDistanceM(participant.Position, target.Position) > maxWeaponRange(participant.Profile.Weapons) {
				speed := vessel.Class.MaxSpeedMPS * math.Max(.35, participant.Damage.PropulsionPercent/100)
				moveCombatEntity(&participant, target.Position, speed, .2)
				vessel.Telemetry.Position, vessel.Telemetry.HeadingDeg, vessel.Telemetry.SpeedMPS = participant.Position, participant.HeadingDeg, participant.SpeedMPS
				vessel.Telemetry.Mode = "combat intercept · approved envelope"
				m.vessels[participantID] = vessel
			}
			before := len(m.combatEffects)
			if combatDistanceM(participant.Position, target.Position) <= program.MaximumRangeM {
				m.fireBestWeaponLocked(&participant, &target, id)
			}
			if len(m.combatEffects) > before {
				program.EffectsApplied++
			}
			m.combatEntities[participantID] = participant
			target = m.combatEntities[program.TargetID]
		}
		m.combatEngagements[id] = program
	}
}

func (m *Manager) nextEligibleEngagementTargetLocked(program domain.EngagementProgramV1) string {
	bestID, bestDistance := "", math.MaxFloat64
	for _, targetID := range program.EligibleTargetIDs {
		target, ok := m.combatEntities[targetID]
		if !ok || target.Damage.Sunk || target.Profile.Controlled || targetID == program.TargetID {
			continue
		}
		if program.RequireTargetInArea && len(program.OperatingAreas) > 0 && !combatPointInAreas(target.Position, program.OperatingAreas) {
			continue
		}
		for _, participantID := range program.ParticipantIDs {
			participant, ok := m.combatEntities[participantID]
			if !ok {
				continue
			}
			if distance := combatDistanceM(participant.Position, target.Position); distance < bestDistance {
				bestID, bestDistance = targetID, distance
			}
		}
	}
	return bestID
}

func (m *Manager) releaseEngagementParticipantsLocked(program domain.EngagementProgramV1) {
	autoArmed := map[string]bool{}
	for _, id := range program.AutoArmedIDs {
		autoArmed[id] = true
	}
	for _, participantID := range program.ParticipantIDs {
		entity := m.combatEntities[participantID]
		if entity.ActiveEngagementID != program.ID {
			continue
		}
		entity.ActiveEngagementID, entity.CurrentTargetID, entity.BehaviorState, entity.SpeedMPS = "", "", "safe_hold", 0
		if autoArmed[participantID] && entity.ArmStateSource != "operator" {
			entity.Armed, entity.ArmStateSource, entity.ArmedByMissionID = false, "weapons_safe", ""
		}
		entity.StateVersion++
		m.combatEntities[participantID] = entity
		if vessel, ok := m.vessels[participantID]; ok {
			vessel.Telemetry.SpeedMPS, vessel.Telemetry.Mode = 0, "station_keep · engagement complete"
			m.vessels[participantID] = vessel
		}
	}
	m.combatVersion++
}

func (m *Manager) stopMissionEngagementLocked(missionID string) {
	for id, program := range m.combatEngagements {
		if program.MissionID != missionID || (program.Status != "active" && program.Status != "pending_approval") {
			continue
		}
		program.Status = "stopped"
		m.combatEngagements[id] = program
		m.releaseEngagementParticipantsLocked(program)
		m.recordCombatEventLocked("engagement_stopped", id, program.TargetID, map[string]any{"reason": "mission authority ended"})
	}
}

func (m *Manager) activateMissionEngagementLocked(mission domain.MissionWorkspaceV2) error {
	policy := normalizeEngagementPolicy(mission.EngagementPolicy, mission.FollowContactID)
	if !policy.Enabled {
		return nil
	}
	targetID := m.resolveMissionEngagementTargetLocked(mission.TargetIDs, policy)
	if targetID == "" {
		return &Error{"COMBAT_ENTITY_NOT_FOUND", "No contact satisfies the mission engagement policy."}
	}
	target := m.combatEntities[targetID]
	weapons, maximumRange := []string{}, 0.0
	participants, autoArmed := []string{}, []string{}
	for _, vesselID := range uniqueStrings(mission.TargetIDs) {
		entity, ok := m.combatEntities[vesselID]
		if !ok || !entity.Profile.Controlled || entity.Damage.Sunk || entity.Damage.Disabled || len(entity.Profile.Weapons) == 0 {
			continue
		}
		participants = append(participants, vesselID)
		if policy.AutoArm && !entity.Armed {
			autoArmed = append(autoArmed, vesselID)
			entity.Armed, entity.ArmStateSource, entity.ArmedByMissionID = true, "mission", mission.ID
		}
		entity.ActiveEngagementID = "mission-engagement-" + mission.ID
		entity.CurrentTargetID, entity.BehaviorState = targetID, "engage"
		entity.StateVersion++
		m.combatEntities[vesselID] = entity
		for _, weapon := range entity.Profile.Weapons {
			weapons = append(weapons, vesselID+":"+weapon.ID)
			maximumRange = math.Max(maximumRange, weapon.EffectiveRangeM)
		}
	}
	if len(participants) == 0 {
		return &Error{"WEAPON_UNAVAILABLE", "The engagement mission has no operational armed-capable Fleet vessels."}
	}
	if policy.MaximumRangeM > 0 {
		maximumRange = math.Min(maximumRange, policy.MaximumRangeM)
	}
	id := "mission-engagement-" + mission.ID
	program := domain.EngagementProgramV1{
		SchemaVersion: 1, ID: id, RequestID: "mission-start-" + mission.ID,
		IdempotencyKey: "mission-engagement-" + mission.ID + "-v" + fmt.Sprint(mission.Version),
		TargetID:       target.EntityID, EligibleTargetIDs: m.eligibleMissionEngagementTargetsLocked(policy), ParticipantIDs: participants, AllowedWeaponIDs: weapons,
		MaximumRangeM: maximumRange, MaximumEffects: policy.MaximumEffects,
		DurationSeconds: policy.DurationSeconds, IssuedTickMS: m.simTickMS,
		ExpiresTickMS:        m.simTickMS + policy.DurationSeconds*1000,
		DisengageHullPercent: policy.DisengageHullPercent, MinimumReserve: mission.Constraints.MinimumReserve,
		Status: "active", MissionID: mission.ID, TargetScope: policy.TargetScope,
		ReturnFire: policy.ReturnFire, RequireTargetInArea: policy.RequireTargetInMissionArea,
		OperatingAreas: mission.Geometry.IncludedAreas, AutoArmedIDs: autoArmed, CreatedAt: time.Now().UTC(),
	}
	program.ContentHash = engagementContentHash(program)
	m.combatEngagements[id] = program
	m.recordCombatEventLocked("mission_engagement_activated", id, targetID, map[string]any{"hash": program.ContentHash, "scope": policy.TargetScope, "participants": participants})
	m.combatVersion++
	return nil
}

func (m *Manager) resolveMissionEngagementTargetLocked(participants []string, policy domain.EngagementPolicyV1) string {
	allowed := func(entity domain.CombatEntityStateV1) bool {
		if entity.Profile.Controlled || entity.Damage.Sunk || entity.Damage.Disabled {
			return false
		}
		return policy.TargetScope == "any_contact" || (policy.TargetScope == "hostile_contacts" && entity.Profile.Hostility == "hostile")
	}
	for _, id := range policy.DesignatedTargetIDs {
		if entity, ok := m.combatEntities[id]; ok && !entity.Profile.Controlled && !entity.Damage.Sunk && !entity.Damage.Disabled {
			return id
		}
	}
	bestID, bestDistance := "", math.MaxFloat64
	for id, entity := range m.combatEntities {
		if !allowed(entity) {
			continue
		}
		for _, participantID := range participants {
			if participant, ok := m.combatEntities[participantID]; ok {
				if distance := combatDistanceM(participant.Position, entity.Position); distance < bestDistance {
					bestID, bestDistance = id, distance
				}
			}
		}
	}
	return bestID
}

func (m *Manager) eligibleMissionEngagementTargetsLocked(policy domain.EngagementPolicyV1) []string {
	eligible := []string{}
	for id, entity := range m.combatEntities {
		if entity.Profile.Controlled {
			continue
		}
		allowed := policy.TargetScope == "any_contact" || (policy.TargetScope == "hostile_contacts" && entity.Profile.Hostility == "hostile")
		if policy.TargetScope == "designated" {
			allowed = false
			for _, designated := range policy.DesignatedTargetIDs {
				if designated == id {
					allowed = true
					break
				}
			}
		}
		if allowed {
			eligible = append(eligible, id)
		}
	}
	sort.Strings(eligible)
	return eligible
}

func (m *Manager) advanceMilitaryDefenseLocked() {
	raider := m.combatEntities[blackwakeID]
	if raider.Damage.Sunk {
		return
	}
	for id, entity := range m.combatEntities {
		if entity.Profile.Controlled || entity.Profile.Hostility != "neutral" || len(entity.Profile.Weapons) == 0 || entity.Damage.Sunk {
			continue
		}
		if combatDistanceM(entity.Position, raider.Position) <= maxWeaponRange(entity.Profile.Weapons) && (raider.CurrentTargetID == id || combatDistanceM(entity.Position, raider.Position) < 700) {
			m.fireBestWeaponLocked(&entity, &raider, "self-defense-"+id)
			m.combatEntities[id], raider = entity, m.combatEntities[blackwakeID]
		}
	}
}

func (m *Manager) advanceControlledSelfDefenseLocked() {
	for id, entity := range m.combatEntities {
		if !entity.Profile.Controlled || !entity.Armed || entity.Damage.Sunk || entity.Damage.Disabled || entity.LastAttackerID == "" {
			continue
		}
		// Arming grants bounded return fire, not pursuit. The authorization is
		// short-lived and only applies to the exact entity that fired first.
		if m.simTickMS-entity.LastAttackedTickMS > 5*60*1000 {
			continue
		}
		attacker, ok := m.combatEntities[entity.LastAttackerID]
		if !ok || attacker.Damage.Sunk || combatDistanceM(entity.Position, attacker.Position) > maxWeaponRange(entity.Profile.Weapons) {
			continue
		}
		m.fireBestWeaponLocked(&entity, &attacker, "return-fire-"+id+"-"+attacker.EntityID)
		m.combatEntities[id], m.combatEntities[attacker.EntityID] = entity, attacker
	}
}

func (m *Manager) fireBestWeaponLocked(source, target *domain.CombatEntityStateV1, engagementID string) {
	rangeM := combatDistanceM(source.Position, target.Position)
	for index := len(source.Profile.Weapons) - 1; index >= 0; index-- {
		weapon := source.Profile.Weapons[index]
		if weapon.Ammunition == 0 || rangeM > weapon.EffectiveRangeM || source.WeaponReadyAtTickMS[weapon.ID] > m.simTickMS || source.Damage.WeaponsPercent <= 0 {
			continue
		}
		m.combatSequence++
		hit, damage, component := resolveCombatHit(engagementID, m.combatSequence, source, target, weapon, rangeM)
		source.WeaponReadyAtTickMS[weapon.ID] = m.simTickMS + weapon.ReloadSeconds*1000
		if weapon.Ammunition > 0 {
			source.Profile.Weapons[index].Ammunition--
		}
		effect := domain.CombatEffectV1{SchemaVersion: 1, ID: fmt.Sprintf("effect-%d-%012d", m.simTickMS, m.combatSequence), EngagementID: engagementID, ShotSequence: m.combatSequence, SourceID: source.EntityID, TargetID: target.EntityID, WeaponID: weapon.ID, RangeM: rangeM, Hit: hit, Damage: damage, Component: component, WorldTickMS: m.simTickMS, CreatedAt: time.Now().UTC()}
		target.LastAttackerID, target.LastAttackedTickMS = source.EntityID, m.simTickMS
		if hit {
			m.applyCombatDamageLocked(target, damage, component)
			effect.TargetHullRemaining = target.Damage.Hull
		} else {
			effect.TargetHullRemaining = target.Damage.Hull
		}
		effect.EffectHash = combatHash(effect)
		m.combatEffects = appendBoundedEffect(m.combatEffects, effect, 400)
		projectile := domain.ProjectileEventV1{EventID: "projectile-" + fmt.Sprint(m.combatSequence), EffectID: effect.ID, SourceID: source.EntityID, TargetID: target.EntityID, Kind: weapon.Kind, Start: source.Position, End: target.Position, Hit: hit, Damage: damage, WorldTickMS: m.simTickMS, DisplayTimeMS: 900, CreatedAt: time.Now().UTC()}
		m.appendVisibleProjectileLocked(projectile)
		m.recordCombatEventLocked("weapon_effect", source.EntityID, target.EntityID, map[string]any{"effect_id": effect.ID, "hit": hit, "damage": damage, "weapon": weapon.ID})
		source.StateVersion++
		m.combatEntities[source.EntityID], m.combatEntities[target.EntityID] = *source, *target
		return
	}
}

func (m *Manager) applyCombatDamageLocked(target *domain.CombatEntityStateV1, damage float64, component string) {
	target.Damage.Hull = math.Max(0, target.Damage.Hull-damage)
	target.Damage.IntegrityPercent = target.Damage.Hull / target.Profile.HullMaximum * 100
	target.Damage.LastDamageTickMS, target.Damage.LastRegenerationTickMS = m.simTickMS, 0
	switch component {
	case "propulsion":
		target.Damage.PropulsionPercent = math.Max(0, target.Damage.PropulsionPercent-damage*2)
	case "sensors":
		target.Damage.SensorsPercent = math.Max(0, target.Damage.SensorsPercent-damage*2)
	case "weapons":
		target.Damage.WeaponsPercent = math.Max(0, target.Damage.WeaponsPercent-damage*2)
	}
	if target.Damage.Hull <= 0 {
		target.Damage.Sunk = true
		target.Damage.Disabled = target.Profile.Controlled
		target.SpeedMPS = 0
		delay := target.Profile.RecoveryDelayS * 1000
		wreckDelay := delay
		if wreckDelay > 45000 {
			wreckDelay = 45000
		}
		target.Respawn = domain.RespawnStateV1{Status: "pending", SunkAtTickMS: m.simTickMS, DueAtTickMS: m.simTickMS + delay, WreckUntilMS: m.simTickMS + wreckDelay}
		target.BehaviorState = "sunk"
		m.recordCombatEventLocked("entity_sunk", target.EntityID, "", map[string]any{"respawn_at_tick_ms": target.Respawn.DueAtTickMS})
		if target.Profile.Controlled {
			if vessel, ok := m.vessels[target.EntityID]; ok {
				vessel.Available = false
				vessel.Telemetry.Health, vessel.Telemetry.Mode, vessel.Telemetry.SpeedMPS, vessel.Telemetry.MissionID, vessel.Telemetry.Route = "disabled", "disabled · recovery pending", 0, "", nil
				m.vessels[target.EntityID] = vessel
			}
			for missionID, program := range m.programs {
				delete(program.Cursors, target.EntityID)
				m.programs[missionID] = program
			}
		}
	}
	target.StateVersion++
	target.UpdatedAt = time.Now().UTC()
	m.combatVersion++
}

func resolveCombatHit(engagementID string, sequence int64, source, target *domain.CombatEntityStateV1, weapon domain.WeaponSystemV1, rangeM float64) (bool, float64, string) {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s|%s|%s", engagementID, sequence, source.EntityID, target.EntityID, weapon.ID)))
	roll := float64(uint16(digest[0])<<8|uint16(digest[1])) / 65535
	componentRoll := int(digest[2]) % 10
	falloff := math.Max(.35, 1-rangeM/weapon.EffectiveRangeM*.35)
	sizeBonus := math.Min(.12, target.Profile.HullMaximum/2500)
	accuracy := weapon.Accuracy*falloff*(source.Damage.WeaponsPercent/100) + sizeBonus
	if roll > accuracy {
		return false, 0, ""
	}
	armor := map[string]float64{"light": .10, "medium": .25, "heavy": .40}[target.Profile.Armor]
	variance := .88 + float64(digest[3]%25)/100
	damage := math.Round(weapon.BaseDamage*(1-armor)*falloff*variance*10) / 10
	component := "hull"
	if componentRoll == 0 {
		component = "propulsion"
	} else if componentRoll == 1 {
		component = "sensors"
	} else if componentRoll == 2 {
		component = "weapons"
	}
	return true, damage, component
}

func clearOneComponent(damage *domain.DamageStateV1) string {
	if damage.PropulsionPercent < 100 {
		damage.PropulsionPercent = math.Min(100, damage.PropulsionPercent+35)
		return "propulsion"
	}
	if damage.SensorsPercent < 100 {
		damage.SensorsPercent = math.Min(100, damage.SensorsPercent+35)
		return "sensors"
	}
	if damage.WeaponsPercent < 100 {
		damage.WeaponsPercent = math.Min(100, damage.WeaponsPercent+35)
		return "weapons"
	}
	return ""
}
func moveCombatEntity(entity *domain.CombatEntityStateV1, destination domain.GeoPointV2, speed, seconds float64) {
	distance := combatDistanceM(entity.Position, destination)
	if distance <= 1 {
		entity.SpeedMPS = 0
		return
	}
	step := math.Min(distance, speed*seconds)
	dx, dy := destination[0]-entity.Position[0], destination[1]-entity.Position[1]
	entity.Position[0] += dx * (step / distance)
	entity.Position[1] += dy * (step / distance)
	entity.HeadingDeg = math.Mod(math.Atan2(dx*math.Cos(entity.Position[1]*math.Pi/180), dy)*180/math.Pi+360, 360)
	entity.SpeedMPS = speed
}
func combatDistanceM(a, b domain.GeoPointV2) float64 {
	latitude := (a[1] + b[1]) / 2 * math.Pi / 180
	return math.Hypot((b[0]-a[0])*111000*math.Cos(latitude), (b[1]-a[1])*111000)
}
func combatPositionInBounds(point domain.GeoPointV2) bool {
	return point[0] >= -72.08 && point[0] <= -70.57 && point[1] >= 40.76 && point[1] <= 42.03
}

func combatPointInAreas(point domain.GeoPointV2, areas [][][]float64) bool {
	for _, polygon := range areas {
		if len(polygon) < 3 || !combatPointInRing(point, polygon) {
			continue
		}
		return true
	}
	return false
}

func combatPointInRing(point domain.GeoPointV2, ring [][]float64) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		if len(ring[i]) < 2 || len(ring[j]) < 2 {
			continue
		}
		xi, yi, xj, yj := ring[i][0], ring[i][1], ring[j][0], ring[j][1]
		if (yi > point[1]) != (yj > point[1]) && point[0] < (xj-xi)*(point[1]-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}
func clampCombatPosition(point domain.GeoPointV2) domain.GeoPointV2 {
	return domain.GeoPointV2{
		math.Max(-72.08, math.Min(-70.57, point[0])),
		math.Max(40.76, math.Min(42.03, point[1])),
	}
}
func combatBearingDeg(a, b domain.GeoPointV2) float64 {
	return math.Mod(math.Atan2((b[0]-a[0])*math.Cos((a[1]+b[1])/2*math.Pi/180), b[1]-a[1])*180/math.Pi+360, 360)
}
func combatContactSpeed(id string) float64 {
	for _, spec := range surfaceTraffic {
		if spec.ID == id {
			return spec.SpeedMPS
		}
	}
	return 1.5
}
func maxWeaponRange(weapons []domain.WeaponSystemV1) float64 {
	value := 0.0
	for _, weapon := range weapons {
		value = math.Max(value, weapon.EffectiveRangeM)
	}
	return value
}
func stableSeed(value string) uint64 {
	digest := sha256.Sum256([]byte(value))
	return uint64(digest[0])<<56 | uint64(digest[1])<<48 | uint64(digest[2])<<40 | uint64(digest[3])<<32 | uint64(digest[4])<<24 | uint64(digest[5])<<16 | uint64(digest[6])<<8 | uint64(digest[7])
}
func combatHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
func engagementContentHash(program domain.EngagementProgramV1) string {
	return combatHash(struct {
		TargetID             string        `json:"target_id"`
		EligibleTargetIDs    []string      `json:"eligible_target_ids"`
		ParticipantIDs       []string      `json:"participant_ids"`
		AllowedWeaponIDs     []string      `json:"allowed_weapon_ids"`
		MaximumRangeM        float64       `json:"maximum_range_m"`
		MaximumEffects       int           `json:"maximum_effects"`
		DurationSeconds      int64         `json:"duration_seconds"`
		DisengageHullPercent float64       `json:"disengage_hull_percent"`
		MinimumReserve       float64       `json:"minimum_reserve"`
		MissionID            string        `json:"mission_id,omitempty"`
		TargetScope          string        `json:"target_scope"`
		ReturnFire           bool          `json:"return_fire"`
		RequireTargetInArea  bool          `json:"require_target_in_mission_area"`
		OperatingAreas       [][][]float64 `json:"operating_areas,omitempty"`
	}{program.TargetID, program.EligibleTargetIDs, program.ParticipantIDs, program.AllowedWeaponIDs, program.MaximumRangeM, program.MaximumEffects, program.DurationSeconds, program.DisengageHullPercent, program.MinimumReserve, program.MissionID, program.TargetScope, program.ReturnFire, program.RequireTargetInArea, program.OperatingAreas})
}
func appendBoundedEffect(values []domain.CombatEffectV1, value domain.CombatEffectV1, maximum int) []domain.CombatEffectV1 {
	values = append(values, value)
	if len(values) > maximum {
		values = append([]domain.CombatEffectV1(nil), values[len(values)-maximum:]...)
	}
	return values
}
func (m *Manager) appendVisibleProjectileLocked(value domain.ProjectileEventV1) {
	cutoff, count := value.CreatedAt.Add(-time.Second), 0
	for index := len(m.combatProjectiles) - 1; index >= 0; index-- {
		event := m.combatProjectiles[index]
		if event.CreatedAt.Before(cutoff) {
			break
		}
		if event.SourceID == value.SourceID {
			count++
		}
	}
	if count < 4 {
		m.combatProjectiles = append(m.combatProjectiles, value)
	}
}
func (m *Manager) recordCombatEventLocked(kind, entityID, targetID string, payload map[string]any) {
	m.combatSequence++
	event := domain.CombatEventV1{ID: fmt.Sprintf("combat-event-%d-%012d", m.simTickMS, m.combatSequence), Kind: kind, EntityID: entityID, TargetID: targetID, WorldTickMS: m.simTickMS, Payload: payload, CreatedAt: time.Now().UTC()}
	m.combatEvents = append(m.combatEvents, event)
	if len(m.combatEvents) > 10000 {
		m.combatEvents = append([]domain.CombatEventV1(nil), m.combatEvents[len(m.combatEvents)-10000:]...)
	}
}
