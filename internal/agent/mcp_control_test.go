package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/arena"
	"github.com/fourtytwo42/keelmesh/internal/fleetops"
)

func TestControlMCPReadsFleetAndPreservesHumanApprovalBoundary(t *testing.T) {
	fleet := fleetops.New("", slog.Default())
	arenaManager := arena.New()
	result, err := callControlTool(context.Background(), fleet, arenaManager, "fleet.get_snapshot", json.RawMessage(`{}`))
	if err != nil || len(result.Content) != 1 {
		t.Fatalf("fleet snapshot tool failed: %v", err)
	}
	approval, err := callControlTool(context.Background(), fleet, arenaManager, "effect.request_approval", json.RawMessage(`{"effect_kind":"mission_start","proposal_hash":"abc123","summary":"bounded patrol"}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(approval)
	if !strings.Contains(string(encoded), "HUMAN_APPROVAL_REQUIRED") {
		t.Fatalf("effect tool crossed approval boundary: %s", encoded)
	}
}

func TestControlMCPMissionDraftUsesVersionAndIdempotency(t *testing.T) {
	fleet := fleetops.New("", slog.Default())
	arenaManager := arena.New()
	snapshot := fleet.Snapshot()
	args, _ := json.Marshal(map[string]any{
		"request_id": "mcp-create-1", "idempotency_key": "mcp-create-1",
		"expected_version": snapshot.FleetVersion, "objective": "patrol shoreline",
		"target_ids": []string{snapshot.Vessels[0].ID},
	})
	result, err := callControlTool(context.Background(), fleet, arenaManager, "mission.create_draft", args)
	if err != nil || len(result.Content) != 1 {
		t.Fatalf("mission draft tool failed: %v", err)
	}
	if len(fleet.Snapshot().Missions) != 1 {
		t.Fatal("mission draft was not created through the canonical manager")
	}
}
