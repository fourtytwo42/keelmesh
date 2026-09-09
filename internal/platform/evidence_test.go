package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestExternalDrillsAcceptOnlyCompletedSelfHashedReceipts(t *testing.T) {
	dir := t.TempDir()
	completed := time.Date(2026, 9, 9, 12, 0, 1, 0, time.UTC)
	receipt := domain.PlatformDrillReceiptV1{
		SchemaVersion: 1, ID: "drill-external", Type: "coordination_leader_recovery",
		TargetID: "node-a-04", ActorIdentity: "test-operator", State: "completed",
		ExpectedInvariants: []string{"management remains reachable"},
		Observations:       []domain.PlatformDrillObservationV1{{At: completed, Kind: "recovery_verified", Summary: "converged"}},
		InitialState:       map[string]string{"term": "21"}, FinalState: map[string]string{"term": "22"},
		RecoveryMS: 900, StartedAt: completed.Add(-time.Second), CompletedAt: &completed, Outcome: "passed",
	}
	unsigned, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var canonical map[string]any
	if err := json.Unmarshal(unsigned, &canonical); err != nil {
		t.Fatal(err)
	}
	canonical["evidence_hash"] = ""
	unsigned, _ = json.Marshal(canonical)
	digest := sha256.Sum256(unsigned)
	receipt.EvidenceHash = "sha256:" + hex.EncodeToString(digest[:])
	encoded, _ := json.Marshal(receipt)
	if err := os.WriteFile(filepath.Join(dir, "valid.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt.ID = "tampered"
	tampered, _ := json.Marshal(receipt)
	if err := os.WriteFile(filepath.Join(dir, "tampered.json"), tampered, 0o600); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(Config{EvidenceDir: dir}, slog.Default())
	values := manager.externalDrills()
	if len(values) != 1 || values[0].ID != "drill-external" {
		t.Fatalf("external drills = %+v", values)
	}
}
