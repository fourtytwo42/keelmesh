package trajectory

import (
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestLongProgramUsesRollingHotTapeWithoutMissionLengthLimit(t *testing.T) {
	mission := domain.MissionWorkspaceV2{ID: "mission-long", Constraints: domain.ConstraintSetV2{MaximumSpeedMPS: 2, MinimumReserve: .2, MinimumObjectSeparationM: 50, MinimumVesselSeparationM: 35, MaximumPNTUncertaintyM: 25}}
	plan := domain.FleetPlanV2{ID: "plan-long", ContentHash: "sha256:plan", Assignments: []domain.FleetAssignmentV2{{VesselID: "vessel-1", SpeedMPS: 1, Route: []domain.GeoPointV2{{-71.4, 41.3}, {-71.4, 41.32}}}}}
	lease := domain.FleetLeaseV2{ID: "lease-long"}
	key := []byte("test-authority")
	revision := BuildRevision(mission, plan, lease, 1, 0, 0, key)
	if !ValidateRevision(revision, key) {
		t.Fatal("generated long revision failed validation")
	}
	if len(revision.Segments["vessel-1"]) <= 6 || revision.DurationS <= 60 {
		t.Fatalf("program was still capped at one minute: duration=%d segments=%d", revision.DurationS, len(revision.Segments["vessel-1"]))
	}
	program := NewProgram(mission.ID, revision, 60)
	view := View(program)
	if view.Summary.DurationS != revision.DurationS || view.Summary.HotTapeHorizonS != 60 || len(view.HotTape["vessel-1"]) != 6 {
		t.Fatalf("full program/hot tape separation failed: %#v hot=%d", view.Summary, len(view.HotTape["vessel-1"]))
	}
}

func TestPendingRevisionActivatesOnlyAtFutureBoundary(t *testing.T) {
	mission := domain.MissionWorkspaceV2{ID: "mission-revise", Constraints: domain.ConstraintSetV2{MaximumSpeedMPS: 2, MinimumReserve: .2, MinimumObjectSeparationM: 50, MinimumVesselSeparationM: 35, MaximumPNTUncertaintyM: 25}}
	plan := domain.FleetPlanV2{ID: "plan-one", ContentHash: "sha256:one", Assignments: []domain.FleetAssignmentV2{{VesselID: "vessel-1", SpeedMPS: 1, Route: []domain.GeoPointV2{{-71.4, 41.3}, {-71.4, 41.31}}}}}
	key := []byte("test-authority")
	program := NewProgram(mission.ID, BuildRevision(mission, plan, domain.FleetLeaseV2{ID: "lease-one"}, 1, 0, 0, key), 60)
	plan.ID, plan.ContentHash = "plan-two", "sha256:two"
	pending := BuildRevision(mission, plan, domain.FleetLeaseV2{ID: "lease-two"}, 2, 7, 30, key)
	if pending.CreatedTick != 7 || pending.ActivationTick != 30 {
		t.Fatalf("revision creation/activation ticks = %d/%d, want 7/30", pending.CreatedTick, pending.ActivationTick)
	}
	AddPending(&program, pending)
	hot := View(program).HotTape["vessel-1"]
	if len(hot) != 6 || hot[2].Revision != 1 || hot[3].Revision != 2 {
		t.Fatalf("hot tape did not cross the pending activation boundary: %#v", hot)
	}
	Advance(&program, 29_000)
	if program.ActiveRevision != 1 || program.PendingRevision != 2 {
		t.Fatalf("revision activated early: %#v", program)
	}
	Advance(&program, 1_000)
	if program.ActiveRevision != 2 || program.PendingRevision != 0 {
		t.Fatalf("revision did not activate atomically at boundary: %#v", program)
	}
}
