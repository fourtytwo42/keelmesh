package fleetops

import (
	"log/slog"
	"math"
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
