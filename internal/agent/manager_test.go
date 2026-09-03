package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestMissionOptionsFallsBackToDirectOpenAIWithSingleVesselSchema(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header was not sourced from the key file")
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "gpt-5.6-luna" || request["store"] != false {
			t.Fatalf("unexpected Responses request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"assistant_markdown\":\"I mapped three shoreline patrol approaches for Gannet. Confirm Option A, B, or C.\",\"geometry_option_id\":\"\",\"strategies\":[{\"id\":\"balanced\",\"name\":\"Balanced patrol\",\"description\":\"Depth-safe shoreline coverage\",\"formation\":\"independent\",\"speed_factor\":0.8,\"reserve_bias\":0.5,\"maneuvers\":[\"enter corridor\",\"patrol shoreline\"]},{\"id\":\"reserve\",\"name\":\"Reserve patrol\",\"description\":\"Conservative independent coverage\",\"formation\":\"independent\",\"speed_factor\":0.5,\"reserve_bias\":0.9,\"maneuvers\":[\"enter slowly\",\"safe hold\"]},{\"id\":\"current\",\"name\":\"Current-aware patrol\",\"description\":\"Use favorable simulated current\",\"formation\":\"independent\",\"speed_factor\":0.65,\"reserve_bias\":0.65,\"maneuvers\":[\"join current\",\"patrol shoreline\"]}]}"}]}]}`))
	}))
	defer provider.Close()

	manager := NewManager(Config{
		AIURL:         "http://127.0.0.1:1",
		OpenAIKeyFile: keyFile,
		OpenAIModel:   "gpt-5.6-luna",
		OpenAIURL:     provider.URL,
	}, slog.Default())
	result, err := manager.MissionOptions(context.Background(), domain.MissionPlanningContextV2{
		SchemaVersion: 2,
		MissionID:     "mission-1",
		Intent:        "patrol the shoreline",
		GuidanceKind:  "patrol",
		TargetCount:   1,
		Targets: []domain.MissionPlanningVesselV2{{
			ID: "vessel-1", Name: "Gannet", Class: "Kestrel", Reserve: .9,
			MaxSpeedMPS: 1.8, PNTIntegrity: "trusted", UncertaintyM: 4,
		}},
		WaypointCount: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "openai" || result.Model != "gpt-5.6-luna" || len(result.Strategies) != 3 {
		t.Fatalf("unexpected direct provider result: %#v", result)
	}
	if result.MissionName != "Operation Balanced patrol" {
		t.Fatalf("mission name was not derived from the accepted model strategy: %q", result.MissionName)
	}
	if result.Summary == "" || result.Summary[:8] != "I mapped" {
		t.Fatalf("model-written conversational reply was not retained: %q", result.Summary)
	}
	for _, strategy := range result.Strategies {
		if strategy.Formation != "independent" || strategy.GuidanceKind != "patrol" {
			t.Fatalf("single-vessel semantics were not retained: %#v", strategy)
		}
	}
}

func TestSelectMissionTargetsUsesBoundedAvailableFleet(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "gpt-5.6-luna" || request["store"] != false {
			t.Fatalf("unexpected target selection request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"target_ids\":[\"vessel-1\",\"vessel-2\"],\"explanation\":\"Amber group has the strongest reserve margin for the eastbound hold.\"}"}]}]}`))
	}))
	defer provider.Close()
	manager := NewManager(Config{OpenAIKeyFile: keyFile, OpenAIModel: "gpt-5.6-luna", OpenAIURL: provider.URL}, slog.Default())
	selection, err := manager.SelectMissionTargets(context.Background(), domain.MissionTargetSelectionContextV2{
		SchemaVersion: 2,
		MissionID:     "mission-1",
		Intent:        "Move a group 1 nm east and hold position.",
		Groups:        []domain.MissionTargetGroupCandidateV2{{ID: "group-1", Code: "AG", Name: "Amber Guard", ColorName: "amber", MemberIDs: []string{"vessel-1", "vessel-2"}, Available: true}},
		Vessels:       []domain.MissionTargetVesselCandidateV2{{ID: "vessel-1", Available: true}, {ID: "vessel-2", Available: true}, {ID: "vessel-locked", Available: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Provider != "openai" || len(selection.TargetIDs) != 2 || selection.TargetIDs[0] != "vessel-1" || len(selection.Attempts) != 1 {
		t.Fatalf("unexpected target selection: %#v", selection)
	}
}

func TestInterpretMissionCommandBindsDynamicSurfaceContact(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"guidance_kind\":\"orbit_contact\",\"contact_id\":\"surface-16\",\"contact_behavior\":\"surround\",\"dynamic_target\":true,\"formation\":\"ring\",\"standoff_m\":120,\"minimum_reserve\":0,\"maximum_speed_mps\":0,\"hold_at_end\":true,\"summary\":\"Approach and surround the identified tanker.\"}"}]}]}`))
	}))
	defer provider.Close()
	manager := NewManager(Config{AIURL: "http://127.0.0.1:1", OpenAIKeyFile: keyFile, OpenAIModel: "gpt-5.6-luna", OpenAIURL: provider.URL}, slog.Default())
	result, err := manager.InterpretMissionCommand(context.Background(), domain.MissionCommandInterpretationContextV2{
		SchemaVersion: 2, MissionID: "mission-1", Intent: "approach and surround Safe Haven",
		TargetIDs: []string{"vessel-1", "vessel-2"}, CurrentFormation: "column",
		SurfaceContacts: []domain.SurfaceContactV2{{ID: "surface-16", Name: "MT Safe Haven", Callsign: "SAFE HAVEN", BoatID: "NPC-4116"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContactID != "surface-16" || !result.DynamicTarget || result.ContactBehavior != "surround" || result.Formation != "ring" || result.Provider != "openai" {
		t.Fatalf("unexpected semantic interpretation: %#v", result)
	}
}

func TestWorkspaceCommandUsesModelForBoundedPresentationAction(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "gpt-5.6-luna" || request["store"] != false {
			t.Fatalf("unexpected workspace request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"mode\":\"workspace\",\"speech\":\"Opening Gannet's vessel status.\",\"mission_intent\":\"\",\"actions\":[{\"kind\":\"inspect_vessel\",\"target\":\"Gannet\",\"value\":0}]}"}]}]}`))
	}))
	defer provider.Close()
	manager := NewManager(Config{OpenAIKeyFile: keyFile, OpenAIModel: "gpt-5.6-luna", OpenAIURL: provider.URL}, slog.Default())
	result, err := manager.WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: "Show me Gannet's status.", Persona: "navy"}, domain.FleetSnapshotV2{
		Vessels: []domain.VesselProfileV2{{ID: "vessel-1", Callsign: "Gannet", Designation: "KM-214", DisplayName: "Gannet (KM-214)"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "openai" || result.Mode != "workspace" || len(result.Actions) != 1 || result.Actions[0].Kind != "inspect_vessel" {
		t.Fatalf("unexpected workspace command: %#v", result)
	}
}

func TestWorkspaceCommandConfirmsOnlySuppliedPlanChoice(t *testing.T) {
	manager := NewManager(Config{}, slog.Default())
	request := domain.WorkspaceAssistantRequestV1{
		Text:            "Go with option B.",
		Persona:         "navy",
		ActiveMissionID: "mission-1",
		PlanOptions: []domain.WorkspacePlanOptionV1{
			{Label: "A", PlanID: "plan-a", Name: "Fast Transit", ContentHash: "hash-a", PolicyStatus: "valid"},
			{Label: "B", PlanID: "plan-b", Name: "Balanced Transit", ContentHash: "hash-b", PolicyStatus: "valid"},
			{Label: "C", PlanID: "plan-c", Name: "Reserve Transit", ContentHash: "hash-c", PolicyStatus: "valid"},
		},
	}
	result, err := manager.WorkspaceCommand(context.Background(), request, domain.FleetSnapshotV2{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "workspace" || len(result.Actions) != 1 || result.Actions[0].Kind != "choose_plan" || result.Actions[0].Target != "B" {
		t.Fatalf("unexpected plan confirmation: %#v", result)
	}
}
