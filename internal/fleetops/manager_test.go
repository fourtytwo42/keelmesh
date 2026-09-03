package fleetops

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/trajectory"
)

func TestMissionLoopDefaultsOffAndRestartsFromFinalPose(t *testing.T) {
	for _, test := range []struct {
		name string
		loop bool
	}{
		{name: "hold at end", loop: false},
		{name: "loop to first marker", loop: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := New("", slog.Default())
			vesselID := m.groups["group-01"].MemberIDs[0]
			start := m.vessels[vesselID].Telemetry.Position
			marker := domain.GeoPointV2{start[0] + .000001, start[1]}
			mission := domain.MissionWorkspaceV2{
				SchemaVersion: 2,
				ID:            "mission-loop-test",
				Name:          "Loop test",
				Status:        "executing",
				TargetIDs:     []string{vesselID},
				Constraints:   defaultConstraints(),
				Formation:     "column",
				Loop:          test.loop,
				Version:       1,
			}
			plan := domain.FleetPlanV2{
				ID:          "plan-loop-test",
				MissionID:   mission.ID,
				ContentHash: "sha256:approved-loop-plan",
				Assignments: []domain.FleetAssignmentV2{{VesselID: vesselID, Route: []domain.GeoPointV2{start, marker}, SpeedMPS: 1}},
			}
			lease := domain.FleetLeaseV2{ID: "lease-loop-test"}
			revision := trajectory.BuildRevision(mission, plan, lease, 1, 0, 0, m.secret)
			program := trajectory.NewProgram(mission.ID, revision, 60)
			program.MissionTickMS = 9_800
			trajectory.UpdateCursors(&program)
			vessel := m.vessels[vesselID]
			vessel.Telemetry.MissionID = mission.ID
			vessel.Telemetry.Position = marker
			m.vessels[vesselID] = vessel
			m.missions[mission.ID] = mission
			m.plans[plan.ID] = plan
			m.programs[mission.ID] = program

			m.tickStepLocked()
			updated := m.missions[mission.ID]
			if test.loop {
				looped := m.programs[mission.ID]
				if updated.Status != "executing" || looped.ActiveRevision != 2 {
					t.Fatalf("loop did not restart: mission=%#v program=%#v", updated, looped)
				}
				if got := looped.Revisions[2].Segments[vesselID][0].Start; got != marker {
					t.Fatalf("loop revision jumped away from actual final pose: got=%v want=%v", got, marker)
				}
				if m.vessels[vesselID].Telemetry.Mode != "mission · loop" {
					t.Fatalf("loop vessel mode = %q", m.vessels[vesselID].Telemetry.Mode)
				}
			} else {
				if updated.Loop || updated.Status != "completed" {
					t.Fatalf("non-loop mission did not complete: %#v", updated)
				}
				if vessel := m.vessels[vesselID]; !strings.HasPrefix(vessel.Telemetry.Mode, "station_keep") || vessel.Telemetry.SpeedMPS != 0 || vessel.Telemetry.Position != marker {
					t.Fatalf("non-loop vessel did not hold at final marker: %#v", vessel.Telemetry)
				}
			}
		})
	}
}

func TestMissionMembershipOrLoopChangeInvalidatesActiveAuthority(t *testing.T) {
	m := New("", slog.Default())
	first := m.groups["group-01"].MemberIDs[0]
	second := m.groups["group-02"].MemberIDs[0]
	mission := domain.MissionWorkspaceV2{SchemaVersion: 2, ID: "mission-replan", Name: "Replan", Status: "executing", TargetIDs: []string{first}, TargetSnapshotHash: hashAny([]string{first}), PlanIDs: []string{"plan-replan"}, AuthorizedPlanID: "plan-replan", Constraints: defaultConstraints(), Version: 4}
	m.missions[mission.ID] = mission
	m.plans["plan-replan"] = domain.FleetPlanV2{ID: "plan-replan", MissionID: mission.ID}
	m.leases["lease-replan"] = domain.FleetLeaseV2{ID: "lease-replan", MissionID: mission.ID}
	m.programs[mission.ID] = domain.TrajectoryProgramV1{MissionID: mission.ID}
	vessel := m.vessels[first]
	vessel.Telemetry.MissionID = mission.ID
	vessel.Telemetry.Mode = "mission"
	m.vessels[first] = vessel
	loop := true
	updated, err := m.PatchMission(mission.ID, PatchMissionRequest{
		Mutation:  Mutation{RequestID: "change-authority", IdempotencyKey: "change-authority", ExpectedVersion: mission.Version},
		TargetIDs: &[]string{second},
		Loop:      &loop,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "draft" || !updated.Loop || len(updated.TargetIDs) != 1 || updated.TargetIDs[0] != second || len(updated.PlanIDs) != 0 || updated.AuthorizedPlanID != "" || updated.Trajectory != nil {
		t.Fatalf("authority-changing patch did not produce a clean draft: %#v", updated)
	}
	if _, ok := m.programs[mission.ID]; ok {
		t.Fatal("stale trajectory program survived membership change")
	}
	if vessel := m.vessels[first]; vessel.Telemetry.MissionID != "" || vessel.Telemetry.Mode != "station_keep" || vessel.Telemetry.SpeedMPS != 0 {
		t.Fatalf("previously controlled vessel retained stale authority: %#v", vessel.Telemetry)
	}
}

func TestSeededFleetHasStableClassMix(t *testing.T) {
	m := New("", slog.Default())
	s := m.Snapshot()
	if len(s.Vessels) != 48 || len(s.Groups) != 8 {
		t.Fatalf("seed = %d vessels, %d groups", len(s.Vessels), len(s.Groups))
	}
	for _, g := range s.Groups {
		if g.DecisionPolicy != "lowest_reachable_capable_id" || g.DecisionNodeID != g.MemberIDs[0] || g.DecisionEpoch < 1 {
			t.Fatalf("group %s decision election = policy %q node %q epoch %d", g.ID, g.DecisionPolicy, g.DecisionNodeID, g.DecisionEpoch)
		}
		counts := map[string]int{}
		for _, id := range g.MemberIDs {
			counts[m.vessels[id].Class.ID]++
		}
		if counts["kestrel"] != 3 || counts["mariner"] != 2 || counts["atlas"] != 1 {
			t.Fatalf("group %s class mix: %#v", g.ID, counts)
		}
	}
}

func TestSimulationRateIsBoundedAndControlsAuthoritativeClock(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	if snapshot.SimulationRate != 20 || snapshot.SimulationTick != 0 {
		t.Fatalf("unexpected default simulation clock: rate=%d tick=%d", snapshot.SimulationRate, snapshot.SimulationTick)
	}
	paused, err := m.SetSimulationRate(SimulationRateRequest{Mutation: Mutation{RequestID: "pause", IdempotencyKey: "pause", ExpectedVersion: snapshot.FleetVersion}, Rate: 0})
	if err != nil || paused.SimulationRate != 0 {
		t.Fatalf("pause failed: snapshot=%#v err=%v", paused, err)
	}
	m.tick()
	pausedAfterTick := m.Snapshot()
	if pausedAfterTick.SimulationTick != 0 {
		t.Fatal("paused simulation advanced")
	}
	if pausedAfterTick.SurfaceContacts[0].Position != paused.SurfaceContacts[0].Position {
		t.Fatal("paused surface traffic advanced")
	}
	five, err := m.SetSimulationRate(SimulationRateRequest{Mutation: Mutation{RequestID: "five", IdempotencyKey: "five", ExpectedVersion: paused.FleetVersion}, Rate: 5})
	if err != nil || five.SimulationRate != 5 {
		t.Fatalf("5x rate failed: snapshot=%#v err=%v", five, err)
	}
	m.tick()
	fiveAfterTick := m.Snapshot()
	if fiveAfterTick.SimulationTick != 1000 {
		t.Fatalf("5x tick = %d, want 1000", fiveAfterTick.SimulationTick)
	}
	if fiveAfterTick.SurfaceContacts[0].Position == five.SurfaceContacts[0].Position {
		t.Fatal("surface traffic did not follow the authoritative clock")
	}
	_, err = m.SetSimulationRate(SimulationRateRequest{Mutation: Mutation{RequestID: "bad-rate", IdempotencyKey: "bad-rate", ExpectedVersion: five.FleetVersion}, Rate: 50})
	if typed, ok := err.(*Error); !ok || typed.Code != "INVALID_SIMULATION_RATE" {
		t.Fatalf("expected bounded-rate rejection, got %v", err)
	}
}

func TestManualCompileAndExplicitContactAreStructured(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	targets := snapshot.Groups[0].MemberIDs
	mission, err := m.CreateMission(CreateMissionRequest{Mutation: Mutation{RequestID: "manual-mission", IdempotencyKey: "manual-mission", ExpectedVersion: snapshot.FleetVersion}, Name: "Manual Watch", TargetIDs: targets})
	if err != nil {
		t.Fatal(err)
	}
	mission, err = m.SetGeometry(mission.ID, GeometryRequest{Mutation: Mutation{RequestID: "manual-geometry", IdempotencyKey: "manual-geometry", ExpectedVersion: mission.Version}, Waypoints: []domain.GeoPointV2{{-71.40, 41.36}, {-71.38, 41.34}}})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{Mutation: Mutation{RequestID: "manual-compile", IdempotencyKey: "manual-compile", ExpectedVersion: mission.Version}, Text: "Manual transit route", PlanningMode: "manual", GuidanceKind: "transit", Formation: "column"})
	if err != nil {
		t.Fatal(err)
	}
	if draft.PlanningMode != "manual" || draft.GuidanceKind != "transit" || len(draft.Waypoints) != 2 {
		t.Fatalf("unexpected manual draft: %#v", draft)
	}

	current := m.missions[mission.ID]
	follow, err := m.Compile(mission.ID, CompileRequest{Mutation: Mutation{RequestID: "follow-compile", IdempotencyKey: "follow-compile", ExpectedVersion: current.Version}, Text: "Follow selected contact", PlanningMode: "manual", FollowContactID: "surface-01", GuidanceKind: "follow_contact"})
	if err != nil {
		t.Fatal(err)
	}
	if follow.FollowContactID != "surface-01" || len(follow.Waypoints) != 12 || follow.GeometrySource != "intent:surface-contact:surface-01" {
		t.Fatalf("explicit contact was not resolved: %#v", follow)
	}
}

func TestLocalAdjustmentStopsAndEscalatesOutsideGuardrails(t *testing.T) {
	m := New("", slog.Default())
	vessel := m.vessels[m.groups["group-01"].MemberIDs[1]]
	vessel.Telemetry.PNTIntegrity = "unsafe"
	vessel.Telemetry.UncertaintyM = 80
	segment := domain.TrajectorySegmentV2{MaximumUncertaintyM: 25, MinimumReserve: .2, TargetSpeedMPS: 1, MaximumSpeedMPS: 2}
	adjustment := m.localAdjustmentLocked(vessel, segment)
	if adjustment.InsideEnvelope || adjustment.SpeedFactor != 0 || adjustment.Escalation != "instruction_requested" || adjustment.Contingency != "safe_hold" {
		t.Fatalf("unsafe adjustment did not fail closed: %#v", adjustment)
	}
	if adjustment.DecisionScope != "group" || adjustment.DecisionNodeID != m.groups["group-01"].MemberIDs[0] {
		t.Fatalf("unexpected group decision authority: %#v", adjustment)
	}
}

func TestSeededGroupsUseReadableRealWorldSpacing(t *testing.T) {
	m := New("", slog.Default())
	for _, group := range m.Snapshot().Groups {
		for i, leftID := range group.MemberIDs {
			left := m.vessels[leftID].Telemetry.Position
			for _, rightID := range group.MemberIDs[i+1:] {
				right := m.vessels[rightID].Telemetry.Position
				delta := math.Hypot(left[0]-right[0], left[1]-right[1])
				if delta < .021 {
					t.Fatalf("group %s vessels %s and %s overlap at regional zoom: %.4f degrees", group.ID, leftID, rightID, delta)
				}
			}
		}
	}
}

func TestReachabilityIgnoresEmptyExternalGroups(t *testing.T) {
	m := New("", slog.Default())
	m.groups["empty"] = domain.OperationalGroupV2{ID: "empty", Code: "E", Name: "Empty", MemberIDs: []string{}}
	if _, err := m.Reachability(m.groups["group-01"].MemberIDs[0]); err != nil {
		t.Fatal(err)
	}
}

func TestSeededVesselsAreInWaterWithShorelineMargin(t *testing.T) {
	type geometry struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	type feature struct {
		Properties map[string]any `json:"properties"`
		Geometry   geometry       `json:"geometry"`
	}
	var chart struct {
		Features []feature `json:"features"`
	}
	path := filepath.Join("..", "..", "web", "public", "assets", "maps", "narragansett.geojson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &chart); err != nil {
		t.Fatal(err)
	}
	land := make([][][]domain.GeoPointV2, 0)
	for _, f := range chart.Features {
		if f.Properties["kind"] != "land" {
			continue
		}
		switch f.Geometry.Type {
		case "Polygon":
			var polygon [][]domain.GeoPointV2
			if err := json.Unmarshal(f.Geometry.Coordinates, &polygon); err != nil {
				t.Fatal(err)
			}
			land = append(land, polygon)
		case "MultiPolygon":
			var polygons [][][]domain.GeoPointV2
			if err := json.Unmarshal(f.Geometry.Coordinates, &polygons); err != nil {
				t.Fatal(err)
			}
			land = append(land, polygons...)
		}
	}
	m := New("", slog.Default())
	for _, vessel := range m.Snapshot().Vessels {
		if pointOnLand(vessel.Telemetry.Position, land) {
			t.Fatalf("%s spawned on land at %v", vessel.DisplayName, vessel.Telemetry.Position)
		}
		if distanceToShore(vessel.Telemetry.Position, land) < .004 {
			t.Fatalf("%s spawned too close to the rendered shoreline at %v", vessel.DisplayName, vessel.Telemetry.Position)
		}
	}
}

func TestKnownLegacySpawnIsMigratedWithoutMovingAnOperatedVessel(t *testing.T) {
	m := New("", slog.Default())
	seeded := m.vessels["00000000-0000-6000-8000-000000000002"]
	legacy := seeded
	legacy.Telemetry.Position = spawnPoint(legacySpawnCenters, 0, 1, .008, .007)
	migrated := migrateLegacySpawn(legacy, seeded)
	if migrated.Telemetry.Position != seeded.Telemetry.Position {
		t.Fatalf("legacy position was not migrated: %v", migrated.Telemetry.Position)
	}
	operated := seeded
	operated.Telemetry.Position = domain.GeoPointV2{-71.28, 41.31}
	if got := migrateLegacySpawn(operated, seeded); got.Telemetry.Position != operated.Telemetry.Position {
		t.Fatalf("operated vessel was unexpectedly moved: %v", got.Telemetry.Position)
	}
}

func pointOnLand(point domain.GeoPointV2, polygons [][][]domain.GeoPointV2) bool {
	for _, polygon := range polygons {
		if len(polygon) == 0 || !pointInRing(point, polygon[0]) {
			continue
		}
		inHole := false
		for _, hole := range polygon[1:] {
			if pointInRing(point, hole) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

func pointInRing(point domain.GeoPointV2, ring []domain.GeoPointV2) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > point[1]) != (b[1] > point[1]) && point[0] < (b[0]-a[0])*(point[1]-a[1])/(b[1]-a[1])+a[0] {
			inside = !inside
		}
	}
	return inside
}

func distanceToShore(point domain.GeoPointV2, polygons [][][]domain.GeoPointV2) float64 {
	minimum := math.Inf(1)
	for _, polygon := range polygons {
		for _, ring := range polygon {
			for i := 1; i < len(ring); i++ {
				if distance := pointSegmentDistance(point, ring[i-1], ring[i]); distance < minimum {
					minimum = distance
				}
			}
		}
	}
	return minimum
}

func pointSegmentDistance(point, start, end domain.GeoPointV2) float64 {
	dx, dy := end[0]-start[0], end[1]-start[1]
	if dx == 0 && dy == 0 {
		return math.Hypot(point[0]-start[0], point[1]-start[1])
	}
	t := ((point[0]-start[0])*dx + (point[1]-start[1])*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(point[0]-(start[0]+t*dx), point[1]-(start[1]+t*dy))
}

func TestConcurrentDisjointMissionsAndAuthorityConflict(t *testing.T) {
	m := New("", slog.Default())
	s := m.Snapshot()
	for i := 0; i < 3; i++ {
		ids := s.Groups[i].MemberIDs
		_, err := m.CreateMission(CreateMissionRequest{Mutation: Mutation{RequestID: "r" + string(rune('a'+i)), IdempotencyKey: "k" + string(rune('a'+i)), ExpectedVersion: m.fleetVersion}, Name: "Mission", TargetIDs: ids})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := m.CreateMission(CreateMissionRequest{Mutation: Mutation{RequestID: "rx", IdempotencyKey: "kx", ExpectedVersion: m.fleetVersion}, Name: "Conflict", TargetIDs: []string{s.Groups[0].MemberIDs[0]}})
	if typed, ok := err.(*Error); !ok || typed.Code != "MOVEMENT_AUTHORITY_CONFLICT" {
		t.Fatalf("expected authority conflict, got %v", err)
	}
}

func TestPreviewDoesNotMoveAndExactHashStarts(t *testing.T) {
	m := New("", slog.Default())
	s := m.Snapshot()
	target := s.Groups[0].MemberIDs
	before := m.vessels[target[0]].Telemetry.Position
	mission, err := m.CreateMission(CreateMissionRequest{Mutation: Mutation{RequestID: "mission", IdempotencyKey: "mission", ExpectedVersion: m.fleetVersion}, Name: "Search", TargetIDs: target})
	if err != nil {
		t.Fatal(err)
	}
	mission, err = m.SetGeometry(mission.ID, GeometryRequest{Mutation: Mutation{RequestID: "geometry", IdempotencyKey: "geometry", ExpectedVersion: mission.Version}, Waypoints: []domain.GeoPointV2{{before[0] + .01, before[1] + .01}, {before[0] + .02, before[1]}}})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{Mutation: Mutation{RequestID: "compile", IdempotencyKey: "compile", ExpectedVersion: mission.Version}, Text: "Search in a wedge"})
	if err != nil {
		t.Fatal(err)
	}
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{Mutation: Mutation{RequestID: "plans", IdempotencyKey: "plans", ExpectedVersion: mission.Version}, DraftID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	mission = m.missions[mission.ID]
	preview, err := m.Preview(mission.ID, plans[0].ID, PlanActionRequest{Mutation: Mutation{RequestID: "preview", IdempotencyKey: "preview", ExpectedVersion: mission.Version}})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.NothingSent || m.vessels[target[0]].Telemetry.Position != before {
		t.Fatal("preview changed vessel state")
	}
	_, err = m.Authorize(mission.ID, plans[0].ID, PlanActionRequest{Mutation: Mutation{RequestID: "bad", IdempotencyKey: "bad", ExpectedVersion: mission.Version}, PlanHash: "tampered", OperatorID: "demo-operator"})
	if typed, ok := err.(*Error); !ok || typed.Code != "PLAN_HASH_MISMATCH" {
		t.Fatalf("expected hash rejection: %v", err)
	}
	lease, err := m.Authorize(mission.ID, plans[0].ID, PlanActionRequest{Mutation: Mutation{RequestID: "auth", IdempotencyKey: "auth", ExpectedVersion: mission.Version}, PlanHash: plans[0].ContentHash, OperatorID: "demo-operator"})
	if err != nil {
		t.Fatal(err)
	}
	mission = m.missions[mission.ID]
	mission, err = m.Start(mission.ID, plans[0].ID, PlanActionRequest{Mutation: Mutation{RequestID: "start", IdempotencyKey: "start", ExpectedVersion: mission.Version}, PlanHash: plans[0].ContentHash, LeaseID: lease.ID})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Status != "executing" {
		t.Fatalf("status = %s", mission.Status)
	}
	pausedStatus := "paused"
	mission, err = m.PatchMission(mission.ID, PatchMissionRequest{Mutation: Mutation{RequestID: "pause", IdempotencyKey: "pause", ExpectedVersion: mission.Version}, Status: &pausedStatus})
	if err != nil || mission.Status != "paused" {
		t.Fatalf("pause failed: status=%s err=%v", mission.Status, err)
	}
	pausedAt := m.vessels[target[0]].Telemetry.Position
	m.tick()
	if m.vessels[target[0]].Telemetry.Position != pausedAt {
		t.Fatal("paused mission moved a vessel")
	}
	resumeStatus := "executing"
	mission, err = m.PatchMission(mission.ID, PatchMissionRequest{Mutation: Mutation{RequestID: "resume", IdempotencyKey: "resume", ExpectedVersion: mission.Version}, Status: &resumeStatus})
	if err != nil || mission.Status != "executing" {
		t.Fatalf("resume failed: status=%s err=%v", mission.Status, err)
	}
	if err = m.DeleteMission(mission.ID, Mutation{RequestID: "delete", IdempotencyKey: "delete", ExpectedVersion: mission.Version}); err != nil {
		t.Fatal(err)
	}
	if _, exists := m.missions[mission.ID]; exists || m.vessels[target[0]].Telemetry.MissionID != "" || len(m.vessels[target[0]].Telemetry.Route) != 0 {
		t.Fatal("deleted mission retained mission state or vessel authority")
	}
}

func TestCompileResolvesShorelineIntentWithoutDrawnGeometry(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "shore-mission", IdempotencyKey: "shore-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Shoreline patrol",
		TargetIDs: m.groups["group-01"].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "shore-compile", IdempotencyKey: "shore-compile", ExpectedVersion: mission.Version},
		Text:     "Patrol the shoreline and reserve 20% battery.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Ambiguities) != 0 {
		t.Fatalf("unexpected ambiguities: %v", draft.Ambiguities)
	}
	if draft.GeometrySource != "intent:map-depth-coastal-corridor-01" {
		t.Fatalf("geometry source = %q", draft.GeometrySource)
	}
	if len(draft.Waypoints) != 13 || draft.GuidanceKind != "patrol" {
		t.Fatalf("resolved guidance = %s, waypoints = %d", draft.GuidanceKind, len(draft.Waypoints))
	}
	if draft.Constraints.MinimumReserve != .30 {
		t.Fatalf("unsafe reserve request weakened standing minimum: %.2f", draft.Constraints.MinimumReserve)
	}
	if len(draft.ResolutionNotes) != 2 {
		t.Fatalf("resolution notes = %v", draft.ResolutionNotes)
	}
	current := m.missions[mission.ID]
	if current.Version != mission.Version+1 || current.Geometry.Revision != mission.Geometry.Revision+1 || len(current.Geometry.IncludedAreas) != 1 || len(current.Geometry.Waypoints) != 13 {
		t.Fatalf("mission was not updated with resolved geometry: %#v", current)
	}
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{
		Mutation: Mutation{RequestID: "shore-plans", IdempotencyKey: "shore-plans", ExpectedVersion: current.Version},
		DraftID:  draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 3 {
		t.Fatalf("plans = %d", len(plans))
	}
}

func TestAICompileSelectsTargetsForEmptyMissionBeforeResolvingRelativeRoute(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:   Mutation{RequestID: "empty-ai-mission", IdempotencyKey: "empty-ai-mission", ExpectedVersion: snapshot.FleetVersion},
		NamingMode: "ai",
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation:     Mutation{RequestID: "empty-ai-compile", IdempotencyKey: "empty-ai-compile", ExpectedVersion: mission.Version},
		Text:         "Move a group 1 nm east and hold position.",
		PlanningMode: "ai_assisted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.TargetIDs) != 6 || draft.TargetSelection == nil || draft.TargetSelection.Provider != "deterministic" {
		t.Fatalf("expected a complete fallback group with receipt, got %#v", draft)
	}
	if len(draft.Waypoints) < 1 || draft.GuidanceKind != "hold" || draft.GeometrySource == "" {
		t.Fatalf("relative hold route was not resolved after target selection: %#v", draft)
	}
	updated := m.missions[mission.ID]
	if !sameMembers(updated.TargetIDs, draft.TargetIDs) || updated.Version <= mission.Version {
		t.Fatalf("AI-selected roster was not persisted to the mission: %#v", updated)
	}
}

func TestManualCompileStillRequiresExplicitTargets(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "empty-manual-mission", IdempotencyKey: "empty-manual-mission", ExpectedVersion: snapshot.FleetVersion},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Compile(mission.ID, CompileRequest{
		Mutation:     Mutation{RequestID: "empty-manual-compile", IdempotencyKey: "empty-manual-compile", ExpectedVersion: mission.Version},
		Text:         "Move east.",
		PlanningMode: "manual",
	})
	if typed, ok := err.(*Error); !ok || typed.Code != "TARGETS_REQUIRED" {
		t.Fatalf("expected manual target requirement, got %v", err)
	}
}

func TestAICompileRejectsPartialGroupSelectionFromProvider(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "partial-ai-mission", IdempotencyKey: "partial-ai-mission", ExpectedVersion: snapshot.FleetVersion},
	})
	if err != nil {
		t.Fatal(err)
	}
	partial := domain.MissionTargetSelectionV2{
		TargetIDs: []string{snapshot.Groups[0].MemberIDs[0]},
		Summary:   "Provider selected only part of a requested group.",
		Provider:  "openai",
		Model:     "test",
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation:        Mutation{RequestID: "partial-ai-compile", IdempotencyKey: "partial-ai-compile", ExpectedVersion: mission.Version},
		Text:            "Move a group 1 nm east and hold position.",
		PlanningMode:    "ai_assisted",
		TargetSelection: &partial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.TargetIDs) != 6 || draft.TargetSelection == nil || draft.TargetSelection.Provider != "deterministic" {
		t.Fatalf("partial provider selection was not replaced safely: %#v", draft.TargetSelection)
	}
}

func TestSingleVesselAdvisorRejectsFleetFormationsAndBuildsIndependentPlans(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	vesselID := snapshot.Groups[0].MemberIDs[0]
	mission, err := m.CreateMission(CreateMissionRequest{Mutation: Mutation{RequestID: "single-mission", IdempotencyKey: "single-mission", ExpectedVersion: snapshot.FleetVersion}, Name: "Solo shoreline patrol", TargetIDs: []string{vesselID}})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{Mutation: Mutation{RequestID: "single-compile", IdempotencyKey: "single-compile", ExpectedVersion: mission.Version}, Text: "patrol the shoreline and preserve reserve"})
	if err != nil {
		t.Fatal(err)
	}
	bad := domain.MissionAdvisorV2{State: "accepted", Provider: "openrouter", Model: "bad", Strategies: []domain.MissionStrategyV2{
		{ID: "solo-a", Name: "Solo patrol", Description: "invalid fleet prose", Formation: "independent", SpeedFactor: .7, ReserveBias: .3, Maneuvers: []string{"patrol", "regroup on completion"}},
		{ID: "solo-b", Name: "Reserve patrol", Description: "invalid other vessel instruction", Formation: "independent", SpeedFactor: .6, ReserveBias: .4, Maneuvers: []string{"hold", "maintain separation from other vessel"}},
	}}
	draft, err = m.ApplyAdvisor(draft.ID, bad)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Advisor.Provider != "deterministic" {
		t.Fatalf("invalid advisor was not replaced: %#v", draft.Advisor)
	}
	mission = m.missions[mission.ID]
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{Mutation: Mutation{RequestID: "single-plans", IdempotencyKey: "single-plans", ExpectedVersion: mission.Version}, DraftID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, plan := range plans {
		if plan.Formation != "independent" || plan.MinimumSeparationM != 0 || plan.Name == "Adaptive Wedge" || plan.Name == "Line Abreast" {
			t.Fatalf("single-vessel plan retained fleet semantics: %#v", plan)
		}
	}
}

func TestCompileResolvesBeachDepthAndNauticalMileIntent(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "beach-mission", IdempotencyKey: "beach-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Beach patrol",
		TargetIDs: m.groups["group-01"].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "beach-compile", IdempotencyKey: "beach-compile", ExpectedVersion: mission.Version},
		Text:     "patrol the beach, stay within 1nm from the beach as long as ocean depth permits",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Ambiguities) != 0 || draft.GeometrySource != "intent:map-depth-coastal-corridor-01" {
		t.Fatalf("beach intent was not resolved: %#v", draft)
	}
	if draft.Constraints.MaximumShoreDistanceM != 1852 {
		t.Fatalf("maximum shore distance = %.1f", draft.Constraints.MaximumShoreDistanceM)
	}
	if len(draft.Waypoints) != 13 || len(draft.ResolutionNotes) != 2 {
		t.Fatalf("resolved route/notes = %d / %v", len(draft.Waypoints), draft.ResolutionNotes)
	}
	current := m.missions[mission.ID]
	if current.Constraints.MaximumShoreDistanceM != 1852 || len(current.Geometry.IncludedAreas) != 1 {
		t.Fatalf("mission did not retain inferred coastal limits: %#v", current)
	}
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{
		Mutation: Mutation{RequestID: "beach-plans", IdempotencyKey: "beach-plans", ExpectedVersion: current.Version},
		DraftID:  draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plans[0].Recommended || plans[0].PolicyStatus == "prohibited" {
		t.Fatalf("recommended plan must be policy-valid: %#v", plans[0])
	}
}

func TestCompileResolvesRelativeSeawardRoundTripAndBuildsAdvisorStrategies(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	vesselID := snapshot.Groups[0].MemberIDs[0]
	start := m.vessels[vesselID].Telemetry.Position
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "relative-mission", IdempotencyKey: "relative-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:     "Seaward round trip", TargetIDs: []string{vesselID},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "relative-compile", IdempotencyKey: "relative-compile", ExpectedVersion: mission.Version},
		Text:     "Travel 15nm from current location heading out to sea, then return to your current location",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Ambiguities) != 0 || draft.GeometrySource != "intent:relative-seaward:15.0nm" {
		t.Fatalf("relative intent was not resolved: %#v", draft)
	}
	if len(draft.Waypoints) != 2 || draft.Waypoints[0][1] >= start[1] || draft.Waypoints[1] != start {
		t.Fatalf("unexpected out-and-back route: start=%v waypoints=%v", start, draft.Waypoints)
	}
	current := m.missions[mission.ID]
	if len(current.Geometry.IncludedAreas) != 1 || len(current.Geometry.Waypoints) != 2 {
		t.Fatalf("mission did not retain inferred route: %#v", current.Geometry)
	}
	advisor := deterministicAdvisor(1, draft.GuidanceKind, "test")
	draft, err = m.ApplyAdvisor(draft.ID, advisor)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{
		Mutation: Mutation{RequestID: "relative-plans", IdempotencyKey: "relative-plans", ExpectedVersion: current.Version}, DraftID: draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 3 {
		t.Fatalf("expected three strategy cards, got %d", len(plans))
	}
	for _, plan := range plans {
		if plan.Formation != "independent" || len(plan.Assignments) != 1 || len(plan.Assignments[0].Route) != 3 {
			t.Fatalf("invalid solo strategy plan: %#v", plan)
		}
	}
}

func TestCompileResolvesSpokenMultiLegCardinalRouteAndHold(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	targets := snapshot.Groups[3].MemberIDs
	start := m.targetCentroidLocked(targets)
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "multileg-mission", IdempotencyKey: "multileg-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:     "Cardinal hold", TargetIDs: targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "multileg-compile", IdempotencyKey: "multileg-compile", ExpectedVersion: mission.Version},
		Text:     "I want this group to go two nautical miles south then two nautical miles west and then hold position.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Ambiguities) != 0 || draft.GeometrySource != "intent:relative-multileg:2" || draft.GuidanceKind != "hold" {
		t.Fatalf("multi-leg intent was not resolved: %#v", draft)
	}
	if len(draft.Waypoints) != 2 || draft.Waypoints[0][1] >= start[1] || draft.Waypoints[1][0] >= draft.Waypoints[0][0] {
		t.Fatalf("ordered south/west route is wrong: start=%v route=%v", start, draft.Waypoints)
	}
	current := m.missions[mission.ID]
	advisor := deterministicAdvisor(len(targets), draft.GuidanceKind, "test")
	draft, err = m.ApplyAdvisor(draft.ID, advisor)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{Mutation: Mutation{RequestID: "multileg-plans", IdempotencyKey: "multileg-plans", ExpectedVersion: current.Version}, DraftID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) < 2 || len(plans) > 4 {
		t.Fatalf("expected multiple executable strategies, got %d", len(plans))
	}
	for _, plan := range plans {
		if len(plan.Assignments) != len(targets) || len(plan.Assignments[0].Route) != 3 {
			t.Fatalf("multi-leg strategy lost route or targets: %#v", plan)
		}
	}
}

func TestAdvisorSelectsDepthValidatedMissionGeometry(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	vesselID := snapshot.Groups[0].MemberIDs[0]
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "geometry-mission", IdempotencyKey: "geometry-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "AI geometry test",
		TargetIDs: []string{vesselID},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "geometry-compile", IdempotencyKey: "geometry-compile", ExpectedVersion: mission.Version},
		Text:     "patrol the shoreline near the Sakonnet north reach",
	})
	if err != nil {
		t.Fatal(err)
	}
	context, err := m.PlanningContext(draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.Targets) != 1 || context.Targets[0].GroupName == "" || context.Targets[0].GroupCode == "" || context.Targets[0].GroupColorName == "" {
		t.Fatalf("advisor group identity is incomplete: %#v", context.Targets)
	}
	var selected domain.MissionGeometryOptionV2
	for _, option := range context.GeometryOptions {
		if option.ID == "coastal-corridor-03" {
			selected = option
		}
	}
	if selected.ID == "" || !selected.DepthValidated || len(selected.Waypoints) < 2 {
		t.Fatalf("expected bounded Sakonnet geometry option, got %#v", context.GeometryOptions)
	}
	advisor := deterministicAdvisor(1, draft.GuidanceKind, "test")
	advisor.Provider = "openai"
	advisor.Model = "gpt-5.6-luna"
	advisor.GeometryOptionID = selected.ID
	draft, err = m.ApplyAdvisor(draft.ID, advisor)
	if err != nil {
		t.Fatal(err)
	}
	if draft.GeometrySource != "advisor:openai:coastal-corridor-03" || len(draft.Ambiguities) != 0 {
		t.Fatalf("advisor geometry was not accepted: %#v", draft)
	}
	updated := m.missions[mission.ID]
	if len(updated.Geometry.Waypoints) != len(selected.Waypoints) || updated.Geometry.Waypoints[0] != selected.Waypoints[0] {
		t.Fatalf("mission did not receive the exact validated option: %#v", updated.Geometry)
	}
}

func TestMissionNamesAreUniqueAndPersistedDuplicatesAreRenumbered(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	first, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "name-one", IdempotencyKey: "name-one", ExpectedVersion: snapshot.FleetVersion},
		Name:     "Mission 1", TargetIDs: []string{snapshot.Vessels[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "name-two", IdempotencyKey: "name-two", ExpectedVersion: snapshot.FleetVersion},
		Name:     "Mission 1", TargetIDs: []string{snapshot.Vessels[1].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Mission 1" || second.Name != "Mission 2" {
		t.Fatalf("mission names were not made unique: %q, %q", first.Name, second.Name)
	}
	third := second
	third.ID = "persisted-duplicate"
	third.Name = "Mission 1"
	third.CreatedAt = second.CreatedAt.Add(time.Second)
	m.missions[third.ID] = third
	m.deduplicateMissionNamesLocked()
	if got := m.missions[third.ID].Name; got != "Mission 3" {
		t.Fatalf("persisted duplicate name = %q", got)
	}
}

func TestEmptyMissionDraftAcceptsTargetsAfterCreation(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation: Mutation{RequestID: "empty-mission", IdempotencyKey: "empty-mission", ExpectedVersion: snapshot.FleetVersion},
	})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Status != "draft" || len(mission.TargetIDs) != 0 {
		t.Fatalf("expected an empty persistent draft, got %#v", mission)
	}

	targets := append([]string(nil), snapshot.Groups[0].MemberIDs...)
	updated, err := m.PatchMission(mission.ID, PatchMissionRequest{
		Mutation:  Mutation{RequestID: "assign-after-create", IdempotencyKey: "assign-after-create", ExpectedVersion: mission.Version},
		TargetIDs: &targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.TargetIDs) != len(targets) || updated.AuthorizedPlanID != "" {
		t.Fatalf("targets were not assigned safely: %#v", updated)
	}
}

func TestGeneratedAndAINamedMissionsAndEntityRenames(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	operatorMission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "operator-name", IdempotencyKey: "operator-name", ExpectedVersion: snapshot.FleetVersion},
		TargetIDs: []string{snapshot.Vessels[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if operatorMission.Name != "Harbor Lantern" || operatorMission.NameSource != "generated" {
		t.Fatalf("unexpected generated mission name: %#v", operatorMission)
	}

	aiMission, err := m.CreateMission(CreateMissionRequest{
		Mutation:   Mutation{RequestID: "ai-name", IdempotencyKey: "ai-name", ExpectedVersion: snapshot.FleetVersion},
		NamingMode: "ai",
		TargetIDs:  []string{snapshot.Vessels[1].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if aiMission.Name != "Operation Pending" || aiMission.NameSource != "ai" {
		t.Fatalf("unexpected pending AI mission name: %#v", aiMission)
	}
	draft, err := m.Compile(aiMission.ID, CompileRequest{
		Mutation:  Mutation{RequestID: "ai-compile", IdempotencyKey: "ai-compile", ExpectedVersion: aiMission.Version},
		Text:      "patrol the shoreline",
		TargetIDs: aiMission.TargetIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err = m.ApplyAdvisor(draft.ID, domain.MissionAdvisorV2{
		State:       "accepted",
		Provider:    "openai",
		Model:       "gpt-5.6-luna",
		MissionName: "Operation Tideglass",
		Strategies:  deterministicAdvisor(1, draft.GuidanceKind, "test").Strategies,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = draft
	renamedAI := m.missions[aiMission.ID]
	if renamedAI.Name != "Operation Tideglass" || renamedAI.NameSource != "openai" {
		t.Fatalf("advisor name was not applied: %#v", renamedAI)
	}

	manualName := "Operation Night Heron"
	renamedMission, err := m.PatchMission(operatorMission.ID, PatchMissionRequest{
		Mutation: Mutation{RequestID: "rename-mission", IdempotencyKey: "rename-mission", ExpectedVersion: operatorMission.Version},
		Name:     &manualName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if renamedMission.Name != manualName || renamedMission.NameSource != "operator" {
		t.Fatalf("mission rename was not retained: %#v", renamedMission)
	}

	vessel := snapshot.Vessels[2]
	renamedVessel, err := m.PatchVessel(vessel.ID, PatchVesselRequest{
		Mutation: Mutation{RequestID: "rename-vessel", IdempotencyKey: "rename-vessel", ExpectedVersion: m.fleetVersion},
		Callsign: "Nightjar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if renamedVessel.Callsign != "Nightjar" || renamedVessel.DisplayName != "Nightjar ("+vessel.Designation+")" {
		t.Fatalf("vessel rename was not retained: %#v", renamedVessel)
	}

	group := m.groups["group-01"]
	groupName := "Harbor Ward"
	renamedGroup, err := m.PatchGroup(group.ID, PatchGroupRequest{
		Mutation: Mutation{RequestID: "rename-group", IdempotencyKey: "rename-group", ExpectedVersion: group.Revision},
		Name:     &groupName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if renamedGroup.Name != groupName {
		t.Fatalf("group rename was not retained: %#v", renamedGroup)
	}
}

func TestColoredWaypointsPersistRenumberAndCompileByColor(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "color-mission", IdempotencyKey: "color-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Colored route",
		TargetIDs: m.groups["group-01"].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	points := []domain.GeoPointV2{{-71.34, 41.40}, {-71.32, 41.41}, {-71.30, 41.42}}
	mission, err = m.SetGeometry(mission.ID, GeometryRequest{
		Mutation:  Mutation{RequestID: "color-geometry", IdempotencyKey: "color-geometry", ExpectedVersion: mission.Version},
		Waypoints: points,
		WaypointDetails: []domain.MissionWaypointV2{
			{ID: "red-1", Color: "red"},
			{ID: "green-1", Color: "green"},
			{ID: "red-2", Color: "red"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index, waypoint := range mission.Geometry.WaypointDetails {
		if waypoint.Sequence != index+1 || waypoint.Position != points[index] {
			t.Fatalf("waypoint metadata was not normalized: %#v", mission.Geometry.WaypointDetails)
		}
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "color-compile", IdempotencyKey: "color-compile", ExpectedVersion: mission.Version},
		Text:     "Send the selected boats through the red waypoints in order.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.GeometrySource != "intent:waypoint-color:red" || len(draft.Waypoints) != 2 || draft.Waypoints[0] != points[0] || draft.Waypoints[1] != points[2] {
		t.Fatalf("colored route selection = %#v", draft)
	}
}

func TestWaypointMetadataFailsClosed(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "invalid-waypoint-mission", IdempotencyKey: "invalid-waypoint-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Invalid waypoint",
		TargetIDs: snapshot.Groups[0].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.SetGeometry(mission.ID, GeometryRequest{
		Mutation:        Mutation{RequestID: "invalid-waypoint", IdempotencyKey: "invalid-waypoint", ExpectedVersion: mission.Version},
		Waypoints:       []domain.GeoPointV2{{-71.34, 41.40}},
		WaypointDetails: []domain.MissionWaypointV2{{ID: "bad", Color: "orange"}},
	})
	if err == nil || err.(*Error).Code != "INVALID_GEOMETRY" {
		t.Fatalf("invalid waypoint color error = %v", err)
	}
}

func TestCompileStillRequiresGeometryForUnresolvedPlaceIntent(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "vague-mission", IdempotencyKey: "vague-mission", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Vague patrol",
		TargetIDs: snapshot.Groups[0].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "vague-compile", IdempotencyKey: "vague-compile", ExpectedVersion: mission.Version},
		Text:     "Patrol somewhere useful.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Ambiguities) != 1 {
		t.Fatalf("expected unresolved geometry ambiguity, got %v", draft.Ambiguities)
	}
}

func TestSavedCollectionCanOverlapGroupsAndBePatched(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	members := []string{snapshot.Groups[0].MemberIDs[0], snapshot.Groups[1].MemberIDs[0]}
	collection, err := m.CreateCollection(CreateCollectionRequest{
		Mutation:  Mutation{RequestID: "collection-create", IdempotencyKey: "collection-create", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Cross-group watch",
		MemberIDs: members,
	})
	if err != nil {
		t.Fatal(err)
	}
	name := "Renamed watch"
	collection, err = m.PatchCollection(collection.ID, PatchCollectionRequest{
		Mutation: Mutation{RequestID: "collection-patch", IdempotencyKey: "collection-patch", ExpectedVersion: collection.Revision},
		Name:     &name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if collection.Name != name || collection.Revision != 2 || len(collection.MemberIDs) != 2 {
		t.Fatalf("unexpected collection: %#v", collection)
	}
}

func TestCreateGroupMovesMembersWithoutDuplicatePrimaryMembership(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	members := []string{snapshot.Groups[0].MemberIDs[0], snapshot.Groups[1].MemberIDs[0]}
	group, err := m.CreateGroup(CreateGroupRequest{
		Mutation:  Mutation{RequestID: "group-create", IdempotencyKey: "group-create", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Harbor Watch",
		Color:     "#d39b3a",
		Pattern:   "ring",
		MemberIDs: members,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, vesselID := range members {
		vessel := m.vessels[vesselID]
		if vessel.GroupID != group.ID || vessel.GroupCode != group.Code {
			t.Fatalf("vessel %s retained wrong primary group: %#v", vesselID, vessel)
		}
		membershipCount := 0
		for _, candidate := range m.groups {
			if contains(candidate.MemberIDs, vesselID) {
				membershipCount++
			}
		}
		if membershipCount != 1 {
			t.Fatalf("vessel %s appears in %d operational groups", vesselID, membershipCount)
		}
	}
}

func TestGroupMembershipChangeFailsClosedDuringActiveMission(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	activeMembers := snapshot.Groups[0].MemberIDs
	_, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "mission-create", IdempotencyKey: "mission-create", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Active patrol",
		TargetIDs: activeMembers,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.CreateGroup(CreateGroupRequest{
		Mutation:  Mutation{RequestID: "group-create", IdempotencyKey: "group-create", ExpectedVersion: m.fleetVersion},
		Name:      "Unsafe regroup",
		Color:     "#d39b3a",
		Pattern:   "ring",
		MemberIDs: []string{activeMembers[0]},
	})
	if typed, ok := err.(*Error); !ok || typed.Code != "ACTIVE_MISSION_REPLAN_REQUIRED" {
		t.Fatalf("expected active mission rejection, got %v", err)
	}
}

func TestPatchGroupCanLeaveMembersExplicitlyUnassigned(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	group := snapshot.Groups[0]
	remaining := cloneStrings(group.MemberIDs[1:])
	updated, err := m.PatchGroup(group.ID, PatchGroupRequest{
		Mutation:  Mutation{RequestID: "group-patch", IdempotencyKey: "group-patch", ExpectedVersion: group.Revision},
		MemberIDs: &remaining,
	})
	if err != nil {
		t.Fatal(err)
	}
	removed := group.MemberIDs[0]
	if contains(updated.MemberIDs, removed) || m.vessels[removed].GroupID != "" || m.vessels[removed].GroupCode != "" {
		t.Fatalf("removed member was not left explicitly unassigned: group=%v vessel=%#v", updated.MemberIDs, m.vessels[removed])
	}
	reassigned := append(cloneStrings(updated.MemberIDs), removed)
	if _, err := m.PatchGroup(group.ID, PatchGroupRequest{
		Mutation:  Mutation{RequestID: "group-readd", IdempotencyKey: "group-readd", ExpectedVersion: updated.Revision},
		MemberIDs: &reassigned,
	}); err != nil {
		t.Fatal(err)
	}
	if _, exists := m.groups[""]; exists {
		t.Fatal("reassigning an unassigned vessel created a phantom empty-ID group")
	}
}

func TestDeleteGroupUnassignsMembersAndKeepsVessels(t *testing.T) {
	m := New("", slog.Default())
	group := m.groups["group-01"]
	members := cloneStrings(group.MemberIDs)
	if err := m.DeleteGroup(group.ID, Mutation{RequestID: "delete-group", IdempotencyKey: "delete-group", ExpectedVersion: group.Revision}); err != nil {
		t.Fatal(err)
	}
	if _, exists := m.groups[group.ID]; exists {
		t.Fatal("deleted group remained present")
	}
	for _, id := range members {
		vessel, exists := m.vessels[id]
		if !exists || vessel.GroupID != "" || vessel.GroupCode != "" {
			t.Fatalf("vessel %s was deleted or remained assigned: %#v", id, vessel)
		}
	}
}

func TestGroupStationPolicyAndAssemblyPointAreVersioned(t *testing.T) {
	m := New("", slog.Default())
	group := m.groups["group-02"]
	formation, spacing, heading := "ring", 85.0, 275.0
	point := domain.GeoPointV2{-71.31, 41.42}
	updated, err := m.PatchGroup(group.ID, PatchGroupRequest{Mutation: Mutation{RequestID: "station-policy", IdempotencyKey: "station-policy", ExpectedVersion: group.Revision}, Formation: &formation, FormationSpacingM: &spacing, FormationHeadingDeg: &heading, AssemblyPoint: &point})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Formation != formation || updated.FormationSpacingM != spacing || updated.FormationHeadingDeg != heading || updated.AssemblyPoint == nil || *updated.AssemblyPoint != point || updated.AssemblySource != "operator" {
		t.Fatalf("station policy not retained: %#v", updated)
	}
}

func TestIdleFormationUsesOrientedSlotsAndCommonHeading(t *testing.T) {
	m := New("", slog.Default())
	group := m.groups["group-02"]
	formation, spacing, heading := "line_abreast", 100.0, 90.0
	center := *group.AssemblyPoint
	group.Formation = formation
	group.FormationSpacingM = spacing
	group.FormationHeadingDeg = heading
	m.groups[group.ID] = group

	for index, vesselID := range group.MemberIDs {
		offset := orientedFormationOffset(formation, index, len(group.MemberIDs), spacing, center[1], heading)
		vessel := m.vessels[vesselID]
		vessel.Telemetry.Position = domain.GeoPointV2{center[0] + offset[0], center[1] + offset[1]}
		vessel.Telemetry.HeadingDeg = float64(index * 47)
		m.vessels[vesselID] = vessel
	}

	m.tickIdleGroupsLocked()
	for _, vesselID := range group.MemberIDs {
		vessel := m.vessels[vesselID]
		if math.Abs(vessel.Telemetry.HeadingDeg-heading) > .001 || vessel.Telemetry.SpeedMPS != 0 {
			t.Fatalf("vessel did not hold the common formation heading: %#v", vessel.Telemetry)
		}
	}
	first := m.vessels[group.MemberIDs[0]].Telemetry.Position
	second := m.vessels[group.MemberIDs[1]].Telemetry.Position
	if distance := geoDistanceM(first, second); math.Abs(distance-spacing) > .5 {
		t.Fatalf("oriented slot spacing = %.2fm, want %.2fm", distance, spacing)
	}
}

func TestMoveGroupMemberIsAtomicAndUpdatesVesselIdentity(t *testing.T) {
	m := New("", slog.Default())
	source := m.groups["group-01"]
	destination := m.groups["group-02"]
	vesselID := source.MemberIDs[0]
	updated, err := m.MoveGroupMember(destination.ID, MoveGroupMemberRequest{
		Mutation: Mutation{RequestID: "move-member", IdempotencyKey: "move-member", ExpectedVersion: destination.Revision},
		VesselID: vesselID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(updated.MemberIDs, vesselID) || contains(m.groups[source.ID].MemberIDs, vesselID) {
		t.Fatalf("membership was not transferred atomically: source=%v destination=%v", m.groups[source.ID].MemberIDs, updated.MemberIDs)
	}
	vessel := m.vessels[vesselID]
	if vessel.GroupID != destination.ID || vessel.GroupCode != destination.Code || vessel.GroupColor != destination.Color || vessel.GroupPattern != destination.Pattern {
		t.Fatalf("vessel identity did not follow destination group: %#v", vessel)
	}
}

func TestMoveGroupMemberFailsClosedDuringMission(t *testing.T) {
	m := New("", slog.Default())
	source := m.groups["group-01"]
	destination := m.groups["group-02"]
	vesselID := source.MemberIDs[0]
	_, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "move-mission", IdempotencyKey: "move-mission", ExpectedVersion: m.fleetVersion},
		Name:      "Frozen membership",
		TargetIDs: []string{vesselID},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.MoveGroupMember(destination.ID, MoveGroupMemberRequest{
		Mutation: Mutation{RequestID: "blocked-move", IdempotencyKey: "blocked-move", ExpectedVersion: destination.Revision},
		VesselID: vesselID,
	})
	if typed, ok := err.(*Error); !ok || typed.Code != "ACTIVE_MISSION_REPLAN_REQUIRED" {
		t.Fatalf("expected active mission rejection, got %v", err)
	}
	if m.vessels[vesselID].GroupID != source.ID {
		t.Fatal("failed move changed vessel membership")
	}
}

func TestSurfaceTrafficHasStableIdentityAndMovingTracks(t *testing.T) {
	first := surfaceContactsAt(time.Unix(1_800_000_000, 0))
	second := surfaceContactsAt(time.Unix(1_800_000_030, 0))
	if len(first) != 12 || len(second) != 12 {
		t.Fatalf("expected twelve surface contacts, got %d and %d", len(first), len(second))
	}
	seen := map[string]bool{}
	for i := range first {
		if first[i].ID != second[i].ID || first[i].BoatID != second[i].BoatID || seen[first[i].BoatID] {
			t.Fatalf("surface contact identity is not stable/unique: %#v", first[i])
		}
		seen[first[i].BoatID] = true
		if first[i].Position == second[i].Position || len(first[i].Route) < 2 || !first[i].Looping {
			t.Fatalf("surface contact is not moving on a looped track: %#v", first[i])
		}
	}
}

func TestFollowSurfaceContactCompilesPredictedTrack(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "follow-create", IdempotencyKey: "follow-create", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Copper Horizon Watch",
		TargetIDs: snapshot.Groups[0].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation: Mutation{RequestID: "follow-compile", IdempotencyKey: "follow-compile", ExpectedVersion: mission.Version},
		Text:     "Have amber team follow NPC-4101 at a safe distance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.FollowContactID != "surface-01" || draft.GuidanceKind != "follow_contact" || len(draft.Waypoints) != 12 || len(draft.Ambiguities) != 0 {
		t.Fatalf("follow command did not compile to the expected track: %#v", draft)
	}
	planning, err := m.PlanningContext(draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if planning.FollowContact == nil || planning.FollowContact.BoatID != "NPC-4101" || len(planning.SurfaceContacts) != 12 {
		t.Fatalf("advisor context is missing bounded traffic state: %#v", planning)
	}
}

func TestControlledFleetCanOvertakeAndPlanAgainstFastestSurfaceContact(t *testing.T) {
	m := New("", slog.Default())
	fastestContact := 0.0
	fastestID := ""
	for _, contact := range m.Snapshot().SurfaceContacts {
		if contact.SpeedMPS > fastestContact {
			fastestContact, fastestID = contact.SpeedMPS, contact.ID
		}
	}
	for _, slot := range []int{0, 3, 5} {
		class := classFor(slot)
		if class.MaxSpeedMPS <= fastestContact {
			t.Fatalf("%s max speed %.1f cannot overtake fastest contact %.1f", class.Name, class.MaxSpeedMPS, fastestContact)
		}
	}

	snapshot := m.Snapshot()
	mission, err := m.CreateMission(CreateMissionRequest{
		Mutation:  Mutation{RequestID: "overtake-create", IdempotencyKey: "overtake-create", ExpectedVersion: snapshot.FleetVersion},
		Name:      "Fast Contact Watch",
		TargetIDs: snapshot.Groups[0].MemberIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := m.Compile(mission.ID, CompileRequest{
		Mutation:        Mutation{RequestID: "overtake-compile", IdempotencyKey: "overtake-compile", ExpectedVersion: mission.Version},
		Text:            "Intercept and follow the fastest contact",
		PlanningMode:    "manual",
		GuidanceKind:    "follow_contact",
		FollowContactID: fastestID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Constraints.MaximumSpeedMPS <= fastestContact {
		t.Fatalf("follow envelope %.1f does not exceed contact speed %.1f", draft.Constraints.MaximumSpeedMPS, fastestContact)
	}
	current := m.missions[mission.ID]
	plans, err := m.GeneratePlans(mission.ID, PlansRequest{
		Mutation: Mutation{RequestID: "overtake-plans", IdempotencyKey: "overtake-plans", ExpectedVersion: current.Version},
		DraftID:  draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) < 2 {
		t.Fatalf("expected multiple follow strategies, got %d", len(plans))
	}
	for _, plan := range plans {
		for _, assignment := range plan.Assignments {
			vessel := m.vessels[assignment.VesselID]
			if assignment.SpeedMPS <= fastestContact {
				t.Fatalf("plan %s assignment %.2f cannot close on %.2f m/s contact", plan.Name, assignment.SpeedMPS, fastestContact)
			}
			if assignment.SpeedMPS > vessel.Class.MaxSpeedMPS {
				t.Fatalf("plan %s exceeds %s hardware limit: %.2f > %.2f", plan.Name, vessel.Class.Name, assignment.SpeedMPS, vessel.Class.MaxSpeedMPS)
			}
		}
	}
}

func TestGroupAssemblyTranslationAndWaypointRouteState(t *testing.T) {
	m := New("", slog.Default())
	group := m.groups["group-01"]
	anchor := m.vessels[group.MemberIDs[0]].Telemetry.Position
	destination := domain.GeoPointV2{anchor[0] + .01, anchor[1]}
	updated, err := m.PatchGroup(group.ID, PatchGroupRequest{
		Mutation:      Mutation{RequestID: "move-hold", IdempotencyKey: "move-hold", ExpectedVersion: group.Revision},
		AssemblyPoint: &destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := m.vessels[group.MemberIDs[0]].Telemetry.Position
	m.tick()
	after := m.vessels[group.MemberIDs[0]].Telemetry.Position
	if before == after || geoDistanceM(after, destination) >= geoDistanceM(before, destination) {
		t.Fatalf("group did not translate toward moved hold point: before=%v after=%v", before, after)
	}

	waypoints := []domain.MissionWaypointV2{
		{ID: "route-one", Position: destination, Color: updated.ColorName, Sequence: 1},
		{ID: "route-two", Position: domain.GeoPointV2{destination[0], destination[1] + .01}, Color: updated.ColorName, Sequence: 2},
	}
	updated, err = m.CommandGroupRoute(group.ID, GroupRouteCommandRequest{
		Mutation:  Mutation{RequestID: "route-once", IdempotencyKey: "route-once", ExpectedVersion: updated.Revision},
		Action:    "start_once",
		Waypoints: waypoints,
	})
	if err != nil || updated.RouteMode != "once" || len(updated.RouteWaypoints) != 2 {
		t.Fatalf("group route did not arm: group=%#v err=%v", updated, err)
	}
	updated, err = m.CommandGroupRoute(group.ID, GroupRouteCommandRequest{
		Mutation:  Mutation{RequestID: "route-loop", IdempotencyKey: "route-loop", ExpectedVersion: updated.Revision},
		Action:    "enable_loop",
		Waypoints: waypoints,
	})
	if err != nil || updated.RouteMode != "loop" {
		t.Fatalf("group route did not enter loop mode: group=%#v err=%v", updated, err)
	}
	updated, err = m.CommandGroupRoute(group.ID, GroupRouteCommandRequest{
		Mutation: Mutation{RequestID: "route-pause", IdempotencyKey: "route-pause", ExpectedVersion: updated.Revision},
		Action:   "pause_after_leg",
	})
	if err != nil || updated.RouteMode != "pause_pending" {
		t.Fatalf("group route did not request bounded pause: group=%#v err=%v", updated, err)
	}
}

func TestClearGroupRouteHoldsAtLowestVesselID(t *testing.T) {
	m := New("", slog.Default())
	group := m.groups["group-01"]
	ids := cloneStrings(group.MemberIDs)
	sort.Strings(ids)
	want := m.vessels[ids[0]].Telemetry.Position
	updated, err := m.CommandGroupRoute(group.ID, GroupRouteCommandRequest{
		Mutation: Mutation{RequestID: "route-clear", IdempotencyKey: "route-clear", ExpectedVersion: group.Revision},
		Action:   "clear",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AssemblyPoint == nil || *updated.AssemblyPoint != want || updated.RouteMode != "moving_to_hold" || len(updated.RouteWaypoints) != 0 {
		t.Fatalf("clear did not create deterministic lowest-ID hold: %#v", updated)
	}
}

func TestNominalRangeAndDaylightSolarRecharge(t *testing.T) {
	m := New("", slog.Default())
	for slot, wantRangeNM := range map[int]float64{0: 20, 3: 30, 5: 45} {
		vessel := domain.VesselProfileV2{Class: classFor(slot), Telemetry: domain.VesselTelemetryV2{Reserve: 1}}
		used := batteryUseFraction(vessel, wantRangeNM*1.852, nominalCruiseMPS)
		if math.Abs(used-1) > .001 {
			t.Fatalf("%s nominal range used %.4f battery, want 1.0", vessel.Class.Name, used)
		}
	}

	kestrel := domain.VesselProfileV2{Class: classFor(0), Telemetry: domain.VesselTelemetryV2{Reserve: .5}}
	noonTick := int64(4 * 60 * 60) // The demo day begins at 08:00.
	charged := m.advanceEnergy(kestrel, 0, noonTick, 3600)
	if charged < .70 {
		t.Fatalf("full-sun stationary recharge too slow: %.3f", charged)
	}
	cruising := m.advanceEnergy(kestrel, nominalCruiseMPS, noonTick, 3600)
	if cruising <= kestrel.Telemetry.Reserve {
		t.Fatalf("full-sun cruise should extend range, reserve %.3f", cruising)
	}

	full := kestrel
	full.Telemetry.Reserve = 1
	nightTick := int64(16 * 60 * 60)
	travelSeconds := 20 * 1852 / nominalCruiseMPS
	remaining := m.advanceEnergy(full, nominalCruiseMPS, nightTick, travelSeconds)
	if remaining > .001 {
		t.Fatalf("20 nm battery-only calibration left %.4f reserve", remaining)
	}
	constraints := defaultConstraints()
	if constraints.MaximumRouteDistanceKM < 45*1.852 || constraints.MaximumDurationMinutes < 20*1852/nominalCruiseMPS/60 {
		t.Fatalf("default planning envelope does not expose configured endurance: %#v", constraints)
	}
}
