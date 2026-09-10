package fleetops

import (
	"log/slog"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestCombatProfilesAndBlackwakeBalance(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	snapshot := m.CombatSnapshot()
	if len(snapshot.Entities) != 45 { // twelve controlled, thirty-two traffic, one raider
		t.Fatalf("unexpected combat population: %d", len(snapshot.Entities))
	}
	raider, err := m.CombatEntity(blackwakeID)
	if err != nil {
		t.Fatal(err)
	}
	if raider.Profile.HullMaximum != 170 || raider.Profile.Armor != "medium" || raider.Profile.Hostility != "hostile" || len(raider.Profile.Weapons) != 2 {
		t.Fatalf("unexpected Blackwake profile: %#v", raider.Profile)
	}
	if raider.Profile.Weapons[0].EffectiveRangeM != 750 || raider.Profile.Weapons[1].EffectiveRangeM != 1400 || raider.SpeedMPS > blackwakeMaximumSpeedMPS || !raider.Armed {
		t.Fatalf("unexpected Blackwake armament or speed: %#v", raider)
	}
	foundSurfaceProjection := false
	for _, contact := range m.Snapshot().SurfaceContacts {
		if contact.ID == blackwakeID && contact.Route == nil {
			t.Fatal("Blackwake surface projection must serialize an empty route, not null")
		}
		foundSurfaceProjection = foundSurfaceProjection || contact.ID == blackwakeID
	}
	if !foundSurfaceProjection {
		t.Fatal("Blackwake surface projection missing")
	}
}

func TestArmStateAllowsReturnFireButNotInitiation(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	armed, err := m.ArmCombatVessel(vesselID, CombatArmRequest{Mutation: Mutation{RequestID: "arm", IdempotencyKey: "arm-key", ActorIdentity: "operator"}, Armed: true})
	if err != nil || !armed.Armed || armed.ArmStateSource != "operator" {
		t.Fatalf("arm failed: %#v %v", armed, err)
	}
	if _, err := m.SetCombatDefense(vesselID, CombatDefenseRequest{Mutation: Mutation{RequestID: "defense", IdempotencyKey: "defense-key", ActorIdentity: "operator"}, Enabled: true, Response: "retaliate"}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	vessel, raider := m.combatEntities[vesselID], m.combatEntities[blackwakeID]
	vessel.Position, raider.Position = domain.GeoPointV2{-71.48, 40.88}, domain.GeoPointV2{-71.479, 40.88}
	m.combatEntities[vesselID], m.combatEntities[blackwakeID] = vessel, raider
	m.advanceControlledSelfDefenseLocked()
	if len(m.combatEffects) != 0 {
		m.mu.Unlock()
		t.Fatal("armed vessel initiated fire without being attacked")
	}
	vessel = m.combatEntities[vesselID]
	vessel.LastAttackerID, vessel.LastAttackedTickMS = blackwakeID, m.simTickMS
	m.combatEntities[vesselID] = vessel
	m.advanceControlledSelfDefenseLocked()
	effects := append([]domain.CombatEffectV1(nil), m.combatEffects...)
	m.mu.Unlock()
	if len(effects) != 1 || effects[0].SourceID != vesselID || effects[0].TargetID != blackwakeID {
		t.Fatalf("bounded return fire missing: %#v", effects)
	}
}

func TestControlledVesselHoldsUntilDefensePolicyAllowsResponse(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	m.mu.Lock()
	vessel, raider := m.combatEntities[vesselID], m.combatEntities[blackwakeID]
	vessel.Position, raider.Position = domain.GeoPointV2{-71.48, 40.88}, domain.GeoPointV2{-71.479, 40.88}
	vessel.LastAttackerID, vessel.LastAttackedTickMS = blackwakeID, m.simTickMS
	raider.CurrentTargetID = vesselID
	m.combatEntities[vesselID], m.combatEntities[blackwakeID] = vessel, raider
	before := vessel.Position
	m.advanceCommercialEscapeLocked(10000)
	after := m.combatEntities[vesselID]
	m.mu.Unlock()
	if after.Position != before || after.BehaviorState == "defensive_evasion" {
		t.Fatalf("default-disabled auto defense moved an idle vessel: before=%v after=%#v", before, after)
	}
	if after.AutoDefense {
		t.Fatal("controlled auto defense must default off")
	}
	if _, err := m.SetCombatDefense(vesselID, CombatDefenseRequest{Mutation: Mutation{RequestID: "retreat", IdempotencyKey: "retreat-key", ActorIdentity: "operator"}, Enabled: true, Response: "retreat"}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.advanceCommercialEscapeLocked(10000)
	retreating := m.combatEntities[vesselID]
	m.mu.Unlock()
	if retreating.Position == before || retreating.BehaviorState != "defensive_evasion" {
		t.Fatalf("explicit retreat defense did not move after attack: %#v", retreating)
	}
}

func TestExplicitEngagementMayTargetNeutralContact(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	program, err := m.PlanCombatEngagement(CombatEngagementRequest{Mutation: Mutation{RequestID: "neutral", IdempotencyKey: "neutral-key"}, TargetID: "surface-07", ParticipantIDs: []string{vesselID}})
	if err != nil || program.TargetID != "surface-07" || program.ContentHash == "" {
		t.Fatalf("neutral engagement was not staged behind approval: %#v %v", program, err)
	}
}

func TestEngagementHashBindsRulesOfEngagement(t *testing.T) {
	base := domain.EngagementProgramV1{
		TargetID: "surface-07", EligibleTargetIDs: []string{"surface-07"},
		ParticipantIDs: []string{"vessel-1"}, AllowedWeaponIDs: []string{"vessel-1:cannon"},
		MaximumRangeM: 650, MaximumEffects: 8, DurationSeconds: 300,
		DisengageHullPercent: 25, MinimumReserve: .3, TargetScope: "designated",
		ReturnFire: true, RequireTargetInArea: true,
		DefensiveResponse: "retaliate",
		OperatingAreas:    [][][]float64{{{-71.6, 41.0}, {-71.4, 41.0}, {-71.4, 41.2}, {-71.6, 41.0}}},
	}
	baseHash := engagementContentHash(base)
	modified := base
	modified.TargetScope = "any_contact"
	modified.EligibleTargetIDs = []string{"surface-07", "surface-08"}
	if engagementContentHash(modified) == baseHash {
		t.Fatal("target scope and eligible targets must be part of the exact engagement hash")
	}
	modified = base
	modified.OperatingAreas = nil
	if engagementContentHash(modified) == baseHash {
		t.Fatal("engagement geography must be part of the exact engagement hash")
	}
	modified = base
	modified.DefensiveResponse = "retreat"
	if engagementContentHash(modified) == baseHash {
		t.Fatal("defensive response must be part of the exact engagement hash")
	}
}

func TestArmStateMutationIsIdempotent(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	req := CombatArmRequest{Mutation: Mutation{RequestID: "arm", IdempotencyKey: "arm-idempotent", ActorIdentity: "operator"}, Armed: true}
	first, err := m.ArmCombatVessel(vesselID, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.ArmCombatVessel(vesselID, req)
	if err != nil || second.StateVersion != first.StateVersion {
		t.Fatalf("arm replay was not idempotent: first=%#v second=%#v err=%v", first, second, err)
	}
	_, err = m.ArmCombatVessel(vesselID, CombatArmRequest{Mutation: req.Mutation, Armed: false})
	if typed, ok := err.(*Error); !ok || typed.Code != "COMBAT_STATE_STALE" {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestBlackwakeMovementIsContinuousAtTopSpeed(t *testing.T) {
	m := New("", slog.Default())
	m.mu.Lock()
	before := m.combatEntities[blackwakeID].Position
	m.advanceBlackwakeLocked(100000)
	after := m.combatEntities[blackwakeID]
	m.mu.Unlock()
	if distance := combatDistanceM(before, after.Position); distance > blackwakeMaximumSpeedMPS*100+.2 {
		t.Fatalf("Blackwake teleported %.2f m in a 100-second step", distance)
	}
	if after.SpeedMPS > blackwakeMaximumSpeedMPS {
		t.Fatalf("Blackwake exceeded top speed: %.2f", after.SpeedMPS)
	}
}

func TestBlackwakeChoosesReachablePredictedIntercept(t *testing.T) {
	m := New("", slog.Default())
	m.mu.Lock()
	raider := m.combatEntities[blackwakeID]
	raider.Position = domain.GeoPointV2{-71.50, 40.90}
	for id, entity := range m.combatEntities {
		if id != blackwakeID {
			entity.Damage.Sunk = true
			m.combatEntities[id] = entity
		}
	}
	slowPosition := pointAtBearing(raider.Position, 90, 2200)
	fastPosition := pointAtBearing(raider.Position, 90, 1800)
	slow := newCombatEntity("slow-contact", "SLOW", "Catchable", "contact", slowPosition, domain.CombatProfileV1{EntityID: "slow-contact", Class: "trawler", HullMaximum: 40, Hostility: "neutral"}, 1)
	slow.HeadingDeg, slow.SpeedMPS = 90, 1.3
	fast := newCombatEntity("fast-contact", "FAST", "Too Fast", "contact", fastPosition, domain.CombatProfileV1{EntityID: "fast-contact", Class: "patrol", HullMaximum: 40, Hostility: "neutral"}, 2)
	fast.HeadingDeg, fast.SpeedMPS = 90, 3.4
	m.combatEntities[slow.EntityID], m.combatEntities[fast.EntityID] = slow, fast

	if selected := m.selectRaiderTargetLocked(raider); selected != slow.EntityID {
		m.mu.Unlock()
		t.Fatalf("Blackwake selected %q instead of the catchable target", selected)
	}
	intercept, eta, feasible, reason := m.raiderInterceptSolutionLocked(raider, slow)
	_, _, fastFeasible, _ := m.raiderInterceptSolutionLocked(raider, fast)
	m.mu.Unlock()
	if !feasible || eta <= 0 || combatDistanceM(intercept, slow.Position) < 100 {
		t.Fatalf("Blackwake did not lead the catchable target: intercept=%v eta=%.1f feasible=%v reason=%s", intercept, eta, feasible, reason)
	}
	if fastFeasible {
		t.Fatal("Blackwake treated a faster target escaping on the same bearing as catchable")
	}
}

func TestBlackwakeWithdrawsAndRepairsDegradedSystems(t *testing.T) {
	m := New("", slog.Default())
	m.mu.Lock()
	raider := m.combatEntities[blackwakeID]
	raider.Position = blackwakePatrol[0]
	raider.Damage.PropulsionPercent = 50
	raider.Damage.SensorsPercent = 70
	raider.Damage.WeaponsPercent = 80
	m.combatEntities[blackwakeID] = raider
	m.advanceBlackwakeLocked(60000)
	after := m.combatEntities[blackwakeID]
	m.mu.Unlock()
	if after.BehaviorState != "withdraw_repair" || after.CurrentTargetID != "" {
		t.Fatalf("degraded Blackwake did not enter bounded repair withdrawal: %#v", after)
	}
	if after.Damage.PropulsionPercent != 60 || after.Damage.SensorsPercent != 80 || after.Damage.WeaponsPercent != 90 {
		t.Fatalf("Blackwake did not repair systems at its recovery point: %#v", after.Damage)
	}
}

func TestMissionEngagementAutoArmsOnlyAssignedParticipants(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vessels := m.Snapshot().Vessels
	mission := domain.MissionWorkspaceV2{
		ID: "mission-attack", TargetIDs: []string{vessels[0].ID}, FollowContactID: "surface-07",
		Constraints: defaultConstraints(), EngagementPolicy: domain.EngagementPolicyV1{
			Enabled: true, TargetScope: "designated", DesignatedTargetIDs: []string{"surface-07"},
			AutoArm: true, ReturnFire: true, MaximumEffects: 4, DurationSeconds: 120, DisengageHullPercent: 25,
		},
	}
	m.mu.Lock()
	err := m.activateMissionEngagementLocked(mission)
	participant := m.combatEntities[vessels[0].ID]
	unassigned := m.combatEntities[vessels[1].ID]
	program := m.combatEngagements["mission-engagement-"+mission.ID]
	m.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if !participant.Armed || participant.ArmedByMissionID != mission.ID || participant.CurrentTargetID != "surface-07" {
		t.Fatalf("mission participant was not armed and targeted: %#v", participant)
	}
	if unassigned.Armed {
		t.Fatal("mission armed an unassigned vessel")
	}
	if program.Status != "active" || program.MaximumEffects != 4 || len(program.AutoArmedIDs) != 1 {
		t.Fatalf("unexpected mission engagement: %#v", program)
	}
}

func TestControlledEngagementRequiresExactHash(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	program, err := m.PlanCombatEngagement(CombatEngagementRequest{Mutation: Mutation{RequestID: "plan", IdempotencyKey: "plan-key"}, TargetID: blackwakeID, ParticipantIDs: []string{vesselID}})
	if err != nil {
		t.Fatal(err)
	}
	if program.Status != "pending_approval" || program.ContentHash == "" {
		t.Fatalf("unexpected plan: %#v", program)
	}
	if _, err := m.AuthorizeCombatEngagement(program.ID, CombatAuthorizeRequest{Mutation: Mutation{RequestID: "bad", IdempotencyKey: "bad"}, PlanHash: "wrong", OperatorID: "operator"}); err == nil {
		t.Fatal("expected exact-hash rejection")
	}
	authorized, err := m.AuthorizeCombatEngagement(program.ID, CombatAuthorizeRequest{Mutation: Mutation{RequestID: "ok", IdempotencyKey: "ok"}, PlanHash: program.ContentHash, OperatorID: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if authorized.Status != "active" {
		t.Fatalf("engagement not active: %#v", authorized)
	}
}

func TestRepairUsesWorldCooldownAndIsIdempotent(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	m.mu.Lock()
	entity := m.combatEntities[vesselID]
	entity.Damage.Hull = entity.Profile.HullMaximum * .5
	entity.Damage.IntegrityPercent = 50
	entity.Damage.PropulsionPercent = 60
	m.combatEntities[vesselID] = entity
	m.mu.Unlock()
	req := CombatRepairRequest{Mutation: Mutation{RequestID: "repair", IdempotencyKey: "repair-key", ActorIdentity: "operator"}, VesselID: vesselID}
	receipt, err := m.RepairCombatVessel(vesselID, req)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.RestoredHull != entity.Profile.HullMaximum*.2 || receipt.ReadyAtTickMS != 300000 || receipt.ClearedComponent != "propulsion" {
		t.Fatalf("unexpected repair: %#v", receipt)
	}
	replayed, err := m.RepairCombatVessel(vesselID, req)
	if err != nil || replayed.ID != receipt.ID || replayed.HullAfter != receipt.HullAfter {
		t.Fatalf("repair was not idempotent: %#v %v", replayed, err)
	}
	_, err = m.RepairCombatVessel(vesselID, CombatRepairRequest{Mutation: Mutation{RequestID: "early", IdempotencyKey: "early"}, VesselID: vesselID})
	if typed, ok := err.(*Error); !ok || typed.Code != "REPAIR_COOLDOWN_ACTIVE" {
		t.Fatalf("expected cooldown error, got %v", err)
	}
}

func TestRegenerationAndRespawnUseWorldTime(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	contactID := "surface-07"
	m.mu.Lock()
	entity := m.combatEntities[contactID]
	entity.Damage.Hull = entity.Profile.HullMaximum * .5
	entity.Damage.IntegrityPercent = 50
	entity.Damage.LastDamageTickMS = 1000
	m.combatEntities[contactID] = entity
	m.simTickMS = 61999
	m.respawnAndRegenerateLocked()
	regenerated := m.combatEntities[contactID]
	if regenerated.Damage.Hull != entity.Profile.HullMaximum*.51 {
		t.Fatalf("expected exactly one one-percent regeneration step, got %.2f", regenerated.Damage.Hull)
	}
	regenerated.Damage.Hull = 0
	regenerated.Damage.Sunk = true
	regenerated.Respawn = domain.RespawnStateV1{Status: "pending", DueAtTickMS: 180000}
	m.combatEntities[contactID] = regenerated
	m.simTickMS = 180000
	m.respawnAndRegenerateLocked()
	respawned := m.combatEntities[contactID]
	m.mu.Unlock()
	if respawned.Damage.Sunk || respawned.Damage.Hull != respawned.Profile.HullMaximum || respawned.SpawnGeneration != 2 {
		t.Fatalf("unexpected respawn: %#v", respawned)
	}
}

func TestHitResolutionIsDeterministic(t *testing.T) {
	m := New("", slog.Default())
	source := m.combatEntities[blackwakeID]
	target := m.combatEntities["surface-07"]
	weapon := source.Profile.Weapons[0]
	aHit, aDamage, aComponent := resolveCombatHit("fixed-engagement", 17, &source, &target, weapon, 400)
	bHit, bDamage, bComponent := resolveCombatHit("fixed-engagement", 17, &source, &target, weapon, 400)
	if aHit != bHit || aDamage != bDamage || aComponent != bComponent {
		t.Fatal("same seed did not produce the same combat effect")
	}
}

func TestControlledVesselCannotFireBeforeEngagementApproval(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	m.mu.Lock()
	vessel := m.combatEntities[vesselID]
	raider := m.combatEntities[blackwakeID]
	vessel.Position = domain.GeoPointV2{-71.48, 40.88}
	raider.Position = domain.GeoPointV2{-71.479, 40.88}
	m.combatEntities[vesselID], m.combatEntities[blackwakeID] = vessel, raider
	m.advanceEngagementsLocked()
	if len(m.combatEffects) != 0 {
		t.Fatal("controlled vessel fired without an authorized engagement")
	}
	m.mu.Unlock()
	program, err := m.PlanCombatEngagement(CombatEngagementRequest{Mutation: Mutation{RequestID: "bounded-plan", IdempotencyKey: "bounded-plan"}, TargetID: blackwakeID, ParticipantIDs: []string{vesselID}, MaximumEffects: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.AuthorizeCombatEngagement(program.ID, CombatAuthorizeRequest{Mutation: Mutation{RequestID: "bounded-authorize", IdempotencyKey: "bounded-authorize"}, PlanHash: program.ContentHash, OperatorID: "operator"}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.advanceEngagementsLocked()
	effects := append([]domain.CombatEffectV1(nil), m.combatEffects...)
	m.mu.Unlock()
	if len(effects) != 1 || effects[0].SourceID != vesselID || effects[0].TargetID != blackwakeID {
		t.Fatalf("authorized effect did not remain bounded: %#v", effects)
	}
}

func TestBlackwakePatrolAndRespawnRemainOffshoreAndVaried(t *testing.T) {
	land := testLandPolygons(t)
	for segment := 1; segment < len(blackwakePatrol); segment++ {
		start, end := blackwakePatrol[segment-1], blackwakePatrol[segment]
		for sample := 0; sample <= 40; sample++ {
			ratio := float64(sample) / 40
			point := domain.GeoPointV2{start[0] + (end[0]-start[0])*ratio, start[1] + (end[1]-start[1])*ratio}
			if pointOnLand(point, land) || distanceToShore(point, land) < .004 {
				t.Fatalf("Blackwake patrol approaches rendered land on segment %d at %v", segment, point)
			}
		}
	}
	m := New("", slog.Default())
	m.mu.Lock()
	before := m.combatEntities[blackwakeID].Position
	raider := m.combatEntities[blackwakeID]
	raider.Damage.Hull, raider.Damage.Sunk = 0, true
	raider.Respawn = domain.RespawnStateV1{Status: "pending", DueAtTickMS: 100}
	m.combatEntities[blackwakeID] = raider
	m.simTickMS = 100
	m.respawnAndRegenerateLocked()
	after := m.combatEntities[blackwakeID]
	m.mu.Unlock()
	if after.Position == before || after.SpawnGeneration != 2 || after.BehaviorState != "roam" {
		t.Fatalf("Blackwake did not use a different offshore respawn: %#v", after)
	}
}

func TestLongRunningCombatPursuitStaysInsideOperatingChart(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	for step := 0; step < 180; step++ {
		m.mu.Lock()
		m.simTickMS += 100000 // one 500x display tick
		m.advanceCombatLocked(100000)
		for id, entity := range m.combatEntities {
			if !combatPositionInBounds(entity.Position) {
				m.mu.Unlock()
				t.Fatalf("%s escaped the operating chart at step %d: %v", id, step, entity.Position)
			}
		}
		m.mu.Unlock()
	}
}

func TestCommercialEscapeRepairsLegacyOutOfBoundsCheckpoint(t *testing.T) {
	m := New("", slog.Default())
	m.mu.Lock()
	entity := m.combatEntities["surface-27"]
	entity.Position = domain.GeoPointV2{-61.9, 40.2}
	entity.BehaviorState = "escape"
	m.combatEntities[entity.EntityID] = entity
	m.advanceCommercialEscapeLocked(100000)
	repaired := m.combatEntities[entity.EntityID]
	m.mu.Unlock()
	if !combatPositionInBounds(repaired.Position) || repaired.BehaviorState != "operational" {
		t.Fatalf("legacy pursuit checkpoint was not reconciled: %#v", repaired)
	}
}

func TestDisabledControlledVesselRecoversWithoutMissionRejoin(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	m := New("", slog.Default())
	vesselID := m.Snapshot().Vessels[0].ID
	m.mu.Lock()
	entity := m.combatEntities[vesselID]
	entity.Damage.Hull, entity.Damage.Sunk, entity.Damage.Disabled = 0, true, true
	entity.Respawn = domain.RespawnStateV1{Status: "pending", DueAtTickMS: 300000}
	vessel := m.vessels[vesselID]
	vessel.Available, vessel.Telemetry.MissionID = false, "old-mission"
	m.combatEntities[vesselID], m.vessels[vesselID] = entity, vessel
	m.simTickMS = 300000
	m.respawnAndRegenerateLocked()
	recoveredEntity, recoveredVessel := m.combatEntities[vesselID], m.vessels[vesselID]
	m.mu.Unlock()
	if recoveredEntity.Damage.Disabled || recoveredEntity.Damage.Hull != recoveredEntity.Profile.HullMaximum {
		t.Fatalf("controlled recovery did not restore the hull: %#v", recoveredEntity)
	}
	if !recoveredVessel.Available || recoveredVessel.Telemetry.MissionID != "" || recoveredVessel.Telemetry.SpeedMPS != 0 {
		t.Fatalf("controlled vessel automatically rejoined old work: %#v", recoveredVessel.Telemetry)
	}
}
