package fleetops

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestSeededFleetHasStableClassMix(t *testing.T) {
	m := New("", slog.Default())
	s := m.Snapshot()
	if len(s.Vessels) != 48 || len(s.Groups) != 8 {
		t.Fatalf("seed = %d vessels, %d groups", len(s.Vessels), len(s.Groups))
	}
	for _, g := range s.Groups {
		counts := map[string]int{}
		for _, id := range g.MemberIDs {
			counts[m.vessels[id].Class.ID]++
		}
		if counts["kestrel"] != 3 || counts["mariner"] != 2 || counts["atlas"] != 1 {
			t.Fatalf("group %s class mix: %#v", g.ID, counts)
		}
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

func TestPatchGroupCannotOrphanPrimaryMembers(t *testing.T) {
	m := New("", slog.Default())
	snapshot := m.Snapshot()
	group := snapshot.Groups[0]
	remaining := cloneStrings(group.MemberIDs[1:])
	_, err := m.PatchGroup(group.ID, PatchGroupRequest{
		Mutation:  Mutation{RequestID: "group-patch", IdempotencyKey: "group-patch", ExpectedVersion: group.Revision},
		MemberIDs: &remaining,
	})
	if typed, ok := err.(*Error); !ok || typed.Code != "PRIMARY_GROUP_REQUIRED" {
		t.Fatalf("expected primary-group rejection, got %v", err)
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
