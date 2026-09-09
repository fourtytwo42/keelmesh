package edgeexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/trajectory"
)

func TestInstallPersistsCompleteProgramAndRejectsConflicts(t *testing.T) {
	key := []byte("edge-store-test-key")
	mission := domain.MissionWorkspaceV2{ID: "mission-1", Constraints: domain.ConstraintSetV2{MinimumReserve: .2, MaximumSpeedMPS: 2, MinimumObjectSeparationM: 50, MinimumVesselSeparationM: 35, MaximumPNTUncertaintyM: 45}}
	plan := domain.FleetPlanV2{ID: "plan-1", MissionID: "mission-1", ContentHash: "plan-hash", Assignments: []domain.FleetAssignmentV2{{VesselID: "vessel-01", SpeedMPS: 1, Route: []domain.GeoPointV2{{1, 1}, {1.02, 1.02}}}}}
	lease := domain.FleetLeaseV2{ID: "lease-1", MissionID: "mission-1", PlanID: "plan-1", PlanHash: "plan-hash"}
	revision := trajectory.BuildRevision(mission, plan, lease, 1, 0, 0, key)
	program := trajectory.NewFullProgram(mission, plan, lease, revision, 0, key)
	store, err := Open(filepath.Join(t.TempDir(), "execution.bbolt"), "node-a-01")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if got, err := store.Install(program); err != nil || got.State != "installed" {
		t.Fatalf("install = %#v, %v", got, err)
	}
	loaded, ok, err := store.Program(program.ProgramID)
	if err != nil || !ok || loaded.ContentHash != program.ContentHash {
		t.Fatalf("load = %#v, %v, %v", loaded, ok, err)
	}
	plan2 := plan
	plan2.ID = "plan-2"
	plan2.ContentHash = "plan-hash-2"
	lease2 := lease
	lease2.ID = "lease-2"
	lease2.PlanID = plan2.ID
	lease2.PlanHash = plan2.ContentHash
	revision2 := trajectory.BuildRevision(mission, plan2, lease2, 2, 10, 20, key)
	trajectory.AddPendingFullProgram(&program, plan2, lease2, mission, revision2, true, key)
	if got, installErr := store.Install(program); installErr != nil || got.Revision != 2 {
		t.Fatalf("revision install = %#v, %v", got, installErr)
	}
	loaded, ok, err = store.Program(program.ProgramID)
	if err != nil || !ok || loaded.PendingRevision != 2 || loaded.ContentHash != program.ContentHash {
		t.Fatalf("revision load = %#v, %v, %v", loaded, ok, err)
	}
	program.MissionID = "tampered"
	if _, err := store.Install(program); err == nil {
		t.Fatal("expected content mismatch")
	}
}

func TestDecisionElectionAndEnvelopeValidation(t *testing.T) {
	if got := ElectDecisionNode([]string{"vessel-12", "vessel-02", "vessel-07"}); got != "vessel-02" {
		t.Fatalf("leader = %s", got)
	}
}

func TestAdaptationValidationAndReconciliation(t *testing.T) {
	key := []byte("edge-adaptation-test-key")
	mission := domain.MissionWorkspaceV2{ID: "mission-2", Constraints: domain.ConstraintSetV2{MinimumReserve: .2, MaximumSpeedMPS: 2, MinimumObjectSeparationM: 50, MinimumVesselSeparationM: 35, MaximumPNTUncertaintyM: 45}}
	plan := domain.FleetPlanV2{ID: "plan-2", MissionID: mission.ID, ContentHash: "plan-2-hash", Assignments: []domain.FleetAssignmentV2{{VesselID: "vessel-01", SpeedMPS: 1, Route: []domain.GeoPointV2{{1, 1}, {1.02, 1.02}}}, {VesselID: "vessel-02", SpeedMPS: 1, Route: []domain.GeoPointV2{{1, 1.001}, {1.02, 1.021}}}}}
	lease := domain.FleetLeaseV2{ID: "lease-2", MissionID: mission.ID, PlanID: plan.ID, PlanHash: plan.ContentHash}
	revision := trajectory.BuildRevision(mission, plan, lease, 1, 0, 0, key)
	program := trajectory.NewFullProgram(mission, plan, lease, revision, 0, key)
	store, err := Open(filepath.Join(t.TempDir(), "execution.bbolt"), "node-a-01")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.Install(program); err != nil {
		t.Fatal(err)
	}
	adaptation := domain.GroupAdaptationV1{SchemaVersion: 1, AdaptationID: "adaptation-1", ProgramID: program.ProgramID, MissionID: mission.ID, VesselID: "vessel-02", DecisionNodeID: "vessel-01", DecisionScope: "group", DecisionEpoch: 2, Tick: 10, Kind: "collision_avoidance", Reason: "verified traffic conflict", HeadingDelta: 8, SpeedFactor: .7, LateralOffsetM: 35, InsideEnvelope: true}
	adaptation.ContentHash = adaptationHash(adaptation)
	if err = store.Adapt(adaptation); err != nil {
		t.Fatalf("adapt = %v", err)
	}
	if err = store.Adapt(adaptation); err != nil {
		t.Fatalf("idempotent adapt = %v", err)
	}
	tampered := adaptation
	tampered.SpeedFactor = 1.2
	if err = store.Adapt(tampered); err == nil {
		t.Fatal("expected tampered adaptation rejection")
	}
	stale := adaptation
	stale.AdaptationID = "adaptation-stale"
	stale.DecisionEpoch = 1
	stale.ContentHash = adaptationHash(stale)
	if err = store.Adapt(stale); err == nil {
		t.Fatal("expected stale decision epoch rejection")
	}
	outside := adaptation
	outside.AdaptationID = "adaptation-outside"
	outside.HeadingDelta = 55
	outside.ContentHash = adaptationHash(outside)
	if err = store.Adapt(outside); err == nil {
		t.Fatal("expected out-of-envelope rejection")
	}
	if err = store.Reconcile(domain.ExecutionReconciliationV1{SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: mission.ID, NodeID: "node-a-01", ProgramRevision: 0}); err == nil {
		t.Fatal("expected stale reconciliation rejection")
	}
	if err = store.Reconcile(domain.ExecutionReconciliationV1{SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: mission.ID, NodeID: "node-a-01", ProgramRevision: 1, ExecutionWatermark: 3, DecisionEpoch: 2, FusedPosition: domain.GeoPointV2{1.01, 1.01}, State: "reconciled"}); err != nil {
		t.Fatalf("reconcile = %v", err)
	}
}

func adaptationHash(value domain.GroupAdaptationV1) string {
	value.ContentHash = ""
	value.Signature = ""
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
