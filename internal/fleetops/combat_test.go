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
	if raider.Profile.Weapons[0].EffectiveRangeM != 750 || raider.Profile.Weapons[1].EffectiveRangeM != 1400 || raider.SpeedMPS > 2.6 {
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
