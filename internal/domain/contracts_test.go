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
