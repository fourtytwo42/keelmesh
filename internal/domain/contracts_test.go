package domain

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMissionIntentV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/mission-intent-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var intent MissionIntentV1
	if err := json.Unmarshal(data, &intent); err != nil {
		t.Fatal(err)
	}
	if intent.SchemaVersion != SchemaVersion || intent.RequestedAssetCount != 6 || intent.Area.Type != "Polygon" {
		t.Fatalf("fixture does not satisfy MissionIntentV1: %+v", intent)
	}
}

func TestResilienceSnapshotV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/resilience-snapshot-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot ResilienceSnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != SchemaVersion || snapshot.IncidentNodeID != "vessel-04" || snapshot.NextAction != "fail_starlink" {
		t.Fatalf("fixture does not satisfy ResilienceSnapshotV1: %+v", snapshot)
	}
}

func TestPlatformSnapshotV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/platform-snapshot-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot PlatformSnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != SchemaVersion || snapshot.ActiveRun == nil || snapshot.ActiveRun.VesselCount != 1000 || len(snapshot.Workers) != 1 || snapshot.Topics[0].Partitions != 12 {
		t.Fatalf("fixture does not satisfy PlatformSnapshotV1: %+v", snapshot)
	}
}

func TestAgentSnapshotV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/agent-snapshot-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot AgentSnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != SchemaVersion || len(snapshot.Incidents) != 1 || snapshot.Incidents[0].ScenarioSeed != 42042 || snapshot.Provider.Models[len(snapshot.Provider.Models)-1] != "openrouter/free" {
		t.Fatalf("fixture does not satisfy AgentSnapshotV1: %+v", snapshot)
	}
}

func TestQuietFleetSnapshotV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/quiet-fleet-snapshot-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot QuietFleetSnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Contract.Quorum != 3 || snapshot.Metrics.QuorumCount != 3 || snapshot.Metrics.AffectedArmed != 3 || snapshot.Decisions[0].ReasonCode != "SPEED_ENVELOPE_EXCEEDED" {
		t.Fatalf("fixture does not satisfy QuietFleetSnapshotV1: %+v", snapshot)
	}
}

func TestMemorySnapshotV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/memory-snapshot-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot MemorySnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != SchemaVersion || snapshot.EmbeddingVersion != "all-MiniLM-L6-v2-onnx-v1" || snapshot.Sync[0].CentralWatermark != 42 {
		t.Fatalf("fixture does not satisfy MemorySnapshotV1: %+v", snapshot)
	}
}

func TestCoordinationV1Fixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/coordination-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Manifest      CoordinationCellManifestV1 `json:"manifest"`
		Identity      NodeIdentityV2             `json:"identity"`
		Command       ReplicatedCommandV1        `json:"command"`
		Receipt       AppliedCommandReceiptV1    `json:"receipt"`
		Proof         QuorumCommitProofV1        `json:"proof"`
		Advertisement CoordinatorAdvertisementV1 `json:"advertisement"`
		Snapshot      CoordinationCellSnapshotV1 `json:"snapshot"`
		CrossCell     CrossCellOperationV1       `json:"cross_cell"`
		PeerTLS       PeerTLSStateV1             `json:"peer_tls"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Manifest.Quorum != 4 || fixture.Identity.VMID != 220 || fixture.Command.Kind != "mission.start" || fixture.Receipt.LogIndex != 42 || fixture.Proof.Required != 4 || fixture.Advertisement.State != "ready" || fixture.Snapshot.LeaderNodeID != "node-a-01" || fixture.CrossCell.State != "preparing" || !fixture.PeerTLS.Trusted {
		t.Fatalf("fixture does not satisfy M12 coordination contracts: %+v", fixture)
	}
}

func TestPlatformProofContractFixtures(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/platform-proof-summary-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof PlatformProofSummaryV1
	if err := json.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	if len(proof.Planes) != 4 || len(proof.SLOs) < 2 || proof.SLOs[1].Measured {
		t.Fatalf("fixture does not satisfy platform proof contracts: %+v", proof)
	}
	data, err = os.ReadFile("../../contracts/fixtures/platform-drill-receipt-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var drill PlatformDrillReceiptV1
	if err := json.Unmarshal(data, &drill); err != nil {
		t.Fatal(err)
	}
	if drill.Outcome != "passed" || len(drill.ExpectedInvariants) != 4 {
		t.Fatalf("fixture does not satisfy drill receipt contract: %+v", drill)
	}
}

func TestExecutionV7ContractFixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/execution-v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Program        TrajectoryProgramV2     `json:"program"`
		Authority      ExecutionAuthorityV1    `json:"authority"`
		InstallReceipt ProgramInstallReceiptV1 `json:"install_receipt"`
		Decision       GroupDecisionStateV1    `json:"decision"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Program.AuthorizationExpiryTick != 7200 || !fixture.Authority.CompleteProgramOnboard || fixture.InstallReceipt.State != "installed" || fixture.Decision.DecisionNodeID != "node-a-01" {
		t.Fatalf("fixture does not satisfy M14 execution contracts: %+v", fixture)
	}
}

func TestCombatV8ContractFixture(t *testing.T) {
	data, err := os.ReadFile("../../contracts/fixtures/combat-v8.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Snapshot CombatSnapshotV1 `json:"snapshot"`
		Repair   RepairReceiptV1  `json:"repair"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Snapshot.Entities) != 1 || fixture.Snapshot.Entities[0].Name != "Blackwake" || fixture.Snapshot.Engagements[0].Status != "active" || fixture.Repair.RestoredHull != 22 {
		t.Fatalf("fixture does not satisfy M15 combat contracts: %+v", fixture)
	}
}
