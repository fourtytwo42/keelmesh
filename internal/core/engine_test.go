package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	edge "github.com/fourtytwo42/keelmesh/internal/resilience"
)

func goldenPlan(t *testing.T, e *Engine) (int64, string, string) {
	t.Helper()
	boot := e.Bootstrap()
	intent, err := e.Compile(CompileRequest{RequestID: "trace-test", ExpectedStateVersion: boot.Snapshot.StateVersion, Text: "Search with six vessels, keep 30% reserve, finish in 20 minutes", Area: &boot.SuggestedArea.Geometry})
	if err != nil {
		t.Fatal(err)
	}
	plans, err := e.GeneratePlans(PlansRequest{RequestID: "plans-test", ExpectedStateVersion: intent.SourceStateVersion, IntentID: intent.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range plans {
		if p.Recommended {
			return intent.SourceStateVersion, p.ID, p.ContentHash
		}
	}
	t.Fatal("no recommended plan")
	return 0, "", ""
}

func TestResilientEdgeFaultScheduleFailsClosedAndRejoins(t *testing.T) {
	e := New()
	version, planID, hash := goldenPlan(t, e)
	lease, err := e.Authorize(planID, AuthorizeRequest{RequestID: "m2-auth", ExpectedStateVersion: version, PlanHash: hash, OperatorID: "demo-operator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Start(lease.MissionID, StartRequest{RequestID: "m2-start", ExpectedStateVersion: version, LeaseID: lease.ID, PlanHash: hash, IdempotencyKey: "m2-start"}); err != nil {
		t.Fatal(err)
	}
	before := e.Snapshot().Vessels[3].Position
	state := e.Snapshot().StateVersion
	if _, wrongTarget := e.ApplyFault(domain.FaultCommandV1{Kind: edge.FaultFailStarlink, TargetID: "vessel-06", RequestID: "wrong", IdempotencyKey: "wrong", ExpectedStateVersion: state}); errorCode(wrongTarget) != "INVALID_FAULT" {
		t.Fatalf("wrong target error=%v", wrongTarget)
	}
	for index, kind := range []string{edge.FaultFailStarlink, edge.FaultPartition, edge.FaultGNSSSpoof, edge.FaultRestore} {
		state := e.Snapshot().StateVersion
		command := domain.FaultCommandV1{SchemaVersion: 1, Kind: kind, TargetID: "vessel-04", ScenarioTick: e.resilience.Snapshot().MissionTick, RequestID: "m2", IdempotencyKey: "fault-" + kind, ExpectedStateVersion: state}
		result, faultErr := e.ApplyFault(command)
		if faultErr != nil {
			t.Fatalf("fault %d %s: %v", index, kind, faultErr)
		}
		if result.StateVersion <= state {
			t.Fatal("fault did not advance state version")
		}
		if replay, replayErr := e.ApplyFault(command); replayErr != nil || replay.StateVersion != result.StateVersion {
			t.Fatalf("idempotent replay=%#v err=%v", replay, replayErr)
		}
	}
	result, err := e.Resilience()
	if err != nil || result.Phase != "rejoined" || result.Bridge == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	incident := result.Nodes[4]
	if incident.PNT.Position != before {
		t.Fatalf("position jumped: before=%v after=%v", before, incident.PNT.Position)
	}
	if incident.PNT.Integrity != "trusted" || len(incident.PNT.ExcludedSources) == 0 {
		t.Fatalf("PNT not safely recovered: %#v", incident.PNT)
	}
}

func TestPreviewDoesNotMoveVesselsAndExecutionDoes(t *testing.T) {
	e := New()
	before := e.Snapshot().Vessels
	version, planID, hash := goldenPlan(t, e)
	preview, err := e.Preview(planID, PreviewRequest{RequestID: "preview-test", ExpectedStateVersion: version})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Samples) < 2 {
		t.Fatal("preview has no one-second samples")
	}
	if !reflect.DeepEqual(before, e.Snapshot().Vessels) {
		t.Fatal("preview mutated vessel state")
	}

	if _, err := e.Authorize(planID, AuthorizeRequest{ExpectedStateVersion: version, PlanHash: hash + "tampered", OperatorID: "demo-operator"}); errorCode(err) != "PLAN_HASH_MISMATCH" {
		t.Fatalf("tampered hash error=%v", err)
	}
	lease, err := e.Authorize(planID, AuthorizeRequest{RequestID: "auth-test", ExpectedStateVersion: version, PlanHash: hash, OperatorID: "demo-operator"})
	if err != nil {
		t.Fatal(err)
	}
	start := StartRequest{RequestID: "start-test", ExpectedStateVersion: version, LeaseID: lease.ID, PlanHash: hash, IdempotencyKey: "start-once"}
	mission, err := e.Start(lease.MissionID, start)
	if err != nil || mission.Phase != "executing" {
		t.Fatalf("start mission=%+v error=%v", mission, err)
	}
	if _, err := e.Start(lease.MissionID, start); err != nil {
		t.Fatalf("identical retry was not idempotent: %v", err)
	}
	conflict := start
	conflict.PlanHash = hash + "different"
	if _, err := e.Start(lease.MissionID, conflict); errorCode(err) != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflicting reuse error=%v", err)
	}

	e.mu.Lock()
	next := e.runtime.lastTick.Add(time.Second)
	e.mu.Unlock()
	e.tick(next)
	if reflect.DeepEqual(before, e.Snapshot().Vessels) {
		t.Fatal("authorized execution did not move vessels")
	}
}

func TestStaleMutationFailsClosed(t *testing.T) {
	e := New()
	boot := e.Bootstrap()
	_, err := e.Compile(CompileRequest{ExpectedStateVersion: boot.Snapshot.StateVersion - 1, Area: &boot.SuggestedArea.Geometry})
	if errorCode(err) != "STALE_STATE" {
		t.Fatalf("stale mutation error=%v", err)
	}
}

func errorCode(err error) string {
	if problem, ok := err.(*Error); ok {
		return problem.Code
	}
	return ""
}
