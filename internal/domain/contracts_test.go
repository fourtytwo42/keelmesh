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
