package trajectory

import (
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func fullProgramFixture(t *testing.T, loop bool) (domain.TrajectoryProgramV2, []byte) {
	t.Helper()
	mission := domain.MissionWorkspaceV2{ID: "mission-full", Loop: loop, Constraints: domain.ConstraintSetV2{MaximumSpeedMPS: 2, MinimumReserve: .2, MinimumObjectSeparationM: 50, MinimumVesselSeparationM: 35, MaximumPNTUncertaintyM: 25, MaximumDurationMinutes: 120}}
	plan := domain.FleetPlanV2{ID: "plan-full", ContentHash: "sha256:plan", Assignments: []domain.FleetAssignmentV2{{VesselID: "vessel-1", SpeedMPS: 1, Route: []domain.GeoPointV2{{-71.4, 41.3}, {-71.4, 41.32}}}}}
	lease := domain.FleetLeaseV2{ID: "lease-full"}
	key := []byte("test-authority")
	revision := BuildRevision(mission, plan, lease, 1, 0, 0, key)
	return NewFullProgram(mission, plan, lease, revision, 0, key), key
}

func TestFullProgramCarriesAllSegmentsBeyondSixtySeconds(t *testing.T) {
	program, key := fullProgramFixture(t, false)
	if err := ValidateFullProgram(program, key); err != nil {
		t.Fatal(err)
	}
	manifest := FullProgramManifest(program)
	if manifest.SegmentCount <= 6 || program.AuthorizationExpiryTick <= 60 {
		t.Fatalf("complete program was truncated: %#v", manifest)
	}
	AdvanceFullProgram(&program, 61_000)
	segment, ok := FullProgramCurrentSegment(program, "vessel-1")
	if !ok || segment.Sequence < 6 {
		t.Fatalf("program did not continue beyond one minute: ok=%v segment=%#v", ok, segment)
	}
}

func TestLoopingProgramStopsAtExplicitAuthorityExpiry(t *testing.T) {
	program, _ := fullProgramFixture(t, true)
	if program.AuthorizationExpiryTick != 120*60 {
		t.Fatalf("loop expiry=%d want 7200", program.AuthorizationExpiryTick)
	}
	program.MissionTickMS = program.AuthorizationExpiryTick * 1000
	UpdateFullProgramCursors(&program)
	if _, ok := FullProgramCurrentSegment(program, "vessel-1"); ok {
		t.Fatal("expired loop still returned executable work")
	}
	if got := FullProgramAuthority(program, "vessel-1", "vessel-1", "local", 1); got.Status != "expired" || got.AuthorizedTimeRemainingSeconds != 0 {
		t.Fatalf("expired authority=%#v", got)
	}
}

func TestFullProgramRejectsTampering(t *testing.T) {
	program, key := fullProgramFixture(t, false)
	program.AuthorizationExpiryTick++
	if err := ValidateFullProgram(program, key); err == nil {
		t.Fatal("tampered authorization boundary was accepted")
	}
}
