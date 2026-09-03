package agent

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestCommandSceneUsesTrustedOrderedSurfaceAndReplacement(t *testing.T) {
	manager := NewManager(Config{OpenAIKeyFile: t.TempDir() + "/missing"}, slog.Default())
	fleet := domain.FleetSnapshotV2{FleetVersion: 7, Vessels: []domain.VesselProfileV2{{ID: "vessel-1", Callsign: "Gannet", DisplayName: "Gannet (KM-214)", Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.3, 41.4}, Reserve: .82, Mode: "hold"}}}}
	request := domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "request-1", IdempotencyKey: "key-1", Text: "Show me Gannet status", SelectedIDs: []string{"vessel-1"}}, ActorIdentity: "demo-operator", WorkspaceVersion: 7}
	first, err := manager.CreateAssistantTurn(context.Background(), request, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if first.Scene.CatalogID != domain.KeelMeshOperationsCatalogV1 || len(first.Scene.PrimarySurface.Messages) != 3 || len(first.Scene.Bindings) != 1 {
		t.Fatalf("unexpected composed scene: %#v", first.Scene)
	}
	if _, ok := first.Scene.PrimarySurface.Messages[0]["createSurface"]; !ok {
		t.Fatal("surface did not begin with createSurface")
	}
	request.RequestID, request.IdempotencyKey, request.Text = "request-2", "key-2", "How many boats do I have?"
	second, err := manager.CreateAssistantTurn(context.Background(), request, fleet)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := manager.Scene(first.Scene.ID)
	if old.State != "replaced" || second.Scene.ID == first.Scene.ID {
		t.Fatalf("unpinned replacement failed: old=%s new=%s", old.State, second.Scene.ID)
	}
	events := manager.SceneEvents(second.ID, 0)
	foundDelete := false
	for _, event := range events {
		if message, ok := event.Payload.(map[string]any); ok {
			_, foundDelete = message["deleteSurface"]
		}
		if foundDelete {
			break
		}
	}
	if !foundDelete {
		t.Fatal("replacement did not emit an ordered A2UI deleteSurface message")
	}
}

func TestPinnedSceneSurvivesAndStaleMutationFails(t *testing.T) {
	manager := NewManager(Config{OpenAIKeyFile: t.TempDir() + "/missing"}, slog.Default())
	fleet := domain.FleetSnapshotV2{FleetVersion: 9}
	turn, err := manager.CreateAssistantTurn(context.Background(), domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "r", IdempotencyKey: "k", Text: "show fleet status"}, ActorIdentity: "demo-operator", WorkspaceVersion: 9}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := manager.MutateScene(turn.Scene.ID, "pin", domain.SceneMutationV1{ActorIdentity: "demo-operator", WorkspaceVersion: 9})
	if err != nil || !pinned.Pinned {
		t.Fatalf("pin failed: %#v %v", pinned, err)
	}
	if _, err := manager.MutateScene(turn.Scene.ID, "dismiss", domain.SceneMutationV1{ActorIdentity: "demo-operator", WorkspaceVersion: 8}); err == nil {
		t.Fatal("stale mutation was accepted")
	}
}

func TestResetCommandScenesClearsTransientAndRetainsPinned(t *testing.T) {
	manager := NewManager(Config{OpenAIKeyFile: t.TempDir() + "/missing"}, slog.Default())
	fleet := domain.FleetSnapshotV2{FleetVersion: 11}
	first, err := manager.CreateAssistantTurn(context.Background(), domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "r1", IdempotencyKey: "k1", Text: "show fleet status"}, ActorIdentity: "demo-operator", WorkspaceVersion: 11}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.MutateScene(first.Scene.ID, "pin", domain.SceneMutationV1{ActorIdentity: "demo-operator", WorkspaceVersion: 11}); err != nil {
		t.Fatal(err)
	}
	second, err := manager.CreateAssistantTurn(context.Background(), domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "r2", IdempotencyKey: "k2", Text: "show Gannet"}, ActorIdentity: "demo-operator", WorkspaceVersion: 11}, fleet)
	if err != nil {
		t.Fatal(err)
	}

	manager.ResetCommandScenes("demo-operator", false)
	if _, err := manager.Scene(second.Scene.ID); err == nil {
		t.Fatal("transient scene survived explicit scenario reset")
	}
	if pinned, err := manager.Scene(first.Scene.ID); err != nil || !pinned.Pinned {
		t.Fatalf("pinned scene was not retained: %#v %v", pinned, err)
	}
}

func TestCommandScenesAreIsolatedByOperatorSession(t *testing.T) {
	manager := NewManager(Config{OpenAIKeyFile: t.TempDir() + "/missing"}, slog.Default())
	fleet := domain.FleetSnapshotV2{FleetVersion: 13}
	first, err := manager.CreateAssistantTurn(context.Background(), domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "session-r1", IdempotencyKey: "session-k1", Text: "show fleet status"}, ActorIdentity: "demo-operator", SessionID: "browser-a", WorkspaceVersion: 13}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.CreateAssistantTurn(context.Background(), domain.AssistantTurnRequestV2{WorkspaceAssistantRequestV1: domain.WorkspaceAssistantRequestV1{RequestID: "session-r2", IdempotencyKey: "session-k2", Text: "show fleet status"}, ActorIdentity: "demo-operator", SessionID: "browser-b", WorkspaceVersion: 13}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	firstStored, _ := manager.Scene(first.Scene.ID)
	if firstStored.State != "active" || second.Scene.State != "active" {
		t.Fatal("one browser session replaced another session's active scene")
	}
	if got := manager.Scenes("demo-operator", "browser-a"); len(got) != 1 || got[0].ID != first.Scene.ID {
		t.Fatalf("browser-a saw the wrong scenes: %#v", got)
	}
	if _, err := manager.MutateScene(first.Scene.ID, "dismiss", domain.SceneMutationV1{ActorIdentity: "demo-operator", SessionID: "browser-b", WorkspaceVersion: 13}); err == nil {
		t.Fatal("cross-session scene mutation was accepted")
	}
}
