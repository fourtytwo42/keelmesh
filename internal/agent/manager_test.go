package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"guidance_kind\":\"orbit_contact\",\"contact_id\":\"surface-16\",\"contact_behavior\":\"surround\",\"dynamic_target\":true,\"formation\":\"ring\",\"formation_spacing_m\":80,\"standoff_m\":120,\"minimum_reserve\":0,\"maximum_speed_mps\":0,\"hold_at_end\":true,\"summary\":\"Approach and surround the identified tanker.\"}"}]}]}`))
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
	if result.ContactID != "surface-16" || !result.DynamicTarget || result.ContactBehavior != "surround" || result.Formation != "ring" || result.FormationSpacingM != 80 || result.Provider != "openai" {
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
		input, _ := request["input"].([]any)
		inputJSON, _ := json.Marshal(input)
		if len(input) < 2 || !strings.Contains(string(inputJSON), `"role":"assistant"`) || !strings.Contains(string(inputJSON), "Atlantic Beacon") || !strings.Contains(string(inputJSON), "conversation_history") || !strings.Contains(string(inputJSON), "recent_entity_references") || !strings.Contains(string(inputJSON), "verified_spatial_facts") || !strings.Contains(string(inputJSON), "Position") {
			t.Fatalf("workspace context omitted first-class history or chart facts: %s", inputJSON)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"mode\":\"workspace\",\"speech\":\"Opening Gannet's vessel status.\",\"mission_intent\":\"\",\"actions\":[{\"kind\":\"inspect_vessel\",\"target\":\"Gannet\",\"value\":0}]}"}]}]}`))
	}))
	defer provider.Close()
	manager := NewManager(Config{OpenAIKeyFile: keyFile, OpenAIModel: "gpt-5.6-luna", OpenAIURL: provider.URL}, slog.Default())
	result, err := manager.WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: "Select Gannet on the map.", Persona: "navy", MemoryContext: &domain.ContextAssemblyV1{RecentTurns: []domain.ConversationTurnV1{{Role: "user", Content: "Tell me about Atlantic Beacon."}, {Role: "assistant", Content: "Atlantic Beacon is a visible contact."}}}}, domain.FleetSnapshotV2{
		Vessels:         []domain.VesselProfileV2{{ID: "vessel-1", Callsign: "Gannet", Designation: "KM-214", DisplayName: "Gannet (KM-214)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.2, 41.5}}}},
		SurfaceContacts: []domain.SurfaceContactV2{{ID: "surface-1", Name: "MV Atlantic Beacon", Position: domain.GeoPointV2{-71.1, 41.4}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "openai" || result.Mode != "workspace" || len(result.Actions) != 1 || result.Actions[0].Kind != "inspect_vessel" {
		t.Fatalf("unexpected workspace command: %#v", result)
	}
}

func TestWorkspaceCommandOpensReferencedContactFromConversationHistory(t *testing.T) {
	fleet := domain.FleetSnapshotV2{SurfaceContacts: []domain.SurfaceContactV2{{ID: "surface-sea-robin", Name: "FV Sea Robin", Callsign: "SEA ROBIN", BoatID: "NPC-4108"}}}
	request := domain.WorkspaceAssistantRequestV1{
		Text: "open its info window pls",
		MemoryContext: &domain.ContextAssemblyV1{RecentTurns: []domain.ConversationTurnV1{
			{Role: "user", Content: "can you tell me about the sea robin?"},
			{Role: "assistant", Content: "FV Sea Robin is a blue commercial trawler, Boat ID NPC-4108."},
		}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), request, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "deterministic" || result.Mode != "workspace" || len(result.Actions) != 1 || result.Actions[0].Kind != "inspect_contact" || result.Actions[0].Target != "surface-sea-robin" {
		t.Fatalf("conversation reference did not resolve to contact inspector: %#v", result)
	}
}

func TestWorkspaceCommandUsesRecentVoiceHistoryAndVerifiedNearestVessel(t *testing.T) {
	fleet := domain.FleetSnapshotV2{
		Vessels: []domain.VesselProfileV2{
			{ID: "vessel-near", Callsign: "Gannet", Designation: "KM-214", DisplayName: "Gannet (KM-214)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.001, 41}}},
			{ID: "vessel-far", Callsign: "Osprey", Designation: "KM-215", DisplayName: "Osprey (KM-215)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.1, 41}}},
		},
		SurfaceContacts: []domain.SurfaceContactV2{{ID: "surface-1", Name: "MV Atlantic Beacon", Callsign: "ATLANTIC BEACON", Position: domain.GeoPointV2{-71, 41}}},
	}
	request := domain.WorkspaceAssistantRequestV1{
		Text: "Which one of our boats is closest to that one?",
		MemoryContext: &domain.ContextAssemblyV1{RecentTurns: []domain.ConversationTurnV1{
			{Role: "user", Content: "Can you tell me about the Atlantic Beacon?"},
			{Role: "assistant", Content: "MV Atlantic Beacon is a container ship."},
		}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), request, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Speech, "Gannet (KM-214)") || !strings.Contains(result.Speech, "MV Atlantic Beacon") || !strings.Contains(result.Speech, "nautical miles") {
		t.Fatalf("recent reference or verified distance was not used: %#v", result)
	}
}

func TestWorkspacePositionAnswerIncludesRequestedReserve(t *testing.T) {
	fleet := domain.FleetSnapshotV2{Vessels: []domain.VesselProfileV2{{
		ID: "vessel-1", Callsign: "Gannet", Designation: "KM-220", DisplayName: "Gannet (KM-220)",
		Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.5, 41.1}, HeadingDeg: 90, SpeedMPS: 1.2, Reserve: .73},
	}}}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: "What is Gannet's current position and reserve?"}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Speech, "Gannet (KM-220)") || !strings.Contains(result.Speech, "73.0% reserve") {
		t.Fatalf("position answer omitted requested reserve: %#v", result)
	}
}

func TestWorkspaceClosestAnswerReturnsRequestedCountAndReserves(t *testing.T) {
	fleet := domain.FleetSnapshotV2{
		Vessels: []domain.VesselProfileV2{
			{ID: "v1", Callsign: "Near", DisplayName: "Near (KM-1)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.20, 41.20}, Reserve: .8}},
			{ID: "v2", Callsign: "Middle", DisplayName: "Middle (KM-2)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.25, 41.20}, Reserve: .7}},
			{ID: "v3", Callsign: "Far", DisplayName: "Far (KM-3)", Available: true, Telemetry: domain.VesselTelemetryV2{Position: domain.GeoPointV2{-71.30, 41.20}, Reserve: .6}},
		},
		SurfaceContacts: []domain.SurfaceContactV2{{ID: "contact-1", Name: "MV Beacon", Callsign: "BEACON", Position: domain.GeoPointV2{-71.19, 41.20}}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: "Which three boats are closest to MV Beacon and what are their reserves?"}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Near (KM-1)", "Middle (KM-2)", "Far (KM-3)", "80.0% reserve", "70.0% reserve", "60.0% reserve"} {
		if !strings.Contains(result.Speech, expected) {
			t.Fatalf("ranked answer omitted %q: %s", expected, result.Speech)
		}
	}
}

func TestWorkspaceMissionPromptForbidsPreplanningStrategyClaims(t *testing.T) {
	instructions := workspaceCommandInstructions("navy")
	for _, required := range []string{"no route or strategy exists yet", "never invent or name a recommended plan", "If exactly one vessel is named"} {
		if !strings.Contains(instructions, required) {
			t.Fatalf("workspace mission instructions omitted %q", required)
		}
	}
}

func TestHoldPositionOrderIsNotMisclassifiedAsPositionQuestion(t *testing.T) {
	fleet := domain.FleetSnapshotV2{
		Groups:          []domain.OperationalGroupV2{{ID: "group-ws", Code: "WS", Name: "Watch Shoal", ColorName: "yellow", MemberIDs: []string{"vessel-1"}}},
		Vessels:         []domain.VesselProfileV2{{ID: "vessel-1", Callsign: "Gannet", Designation: "KM-214", DisplayName: "Gannet (KM-214)", Available: true}},
		SurfaceContacts: []domain.SurfaceContactV2{{ID: "surface-16", Name: "MT Safe Haven", Callsign: "SAFE HAVEN", BoatID: "NPC-4116", Position: domain.GeoPointV2{-71.275, 41.285}}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: "Have Watch Shoal rendezvous with Safe Haven and hold position."}, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "mission" || result.MissionIntent == "" {
		t.Fatalf("hold-position task was misclassified: %#v", result)
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

func TestWorkspaceCommandConfirmsSingleRecommendedPlanWithoutLetter(t *testing.T) {
	request := domain.WorkspaceAssistantRequestV1{
		Text: "Confirm and execute it.", ActiveMissionID: "mission-1",
		PlanOptions: []domain.WorkspacePlanOptionV1{{Label: "A", PlanID: "plan-a", Name: "Predicted Intercept", ContentHash: "hash-a", PolicyStatus: "approval_required"}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), request, domain.FleetSnapshotV2{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Actions) != 1 || result.Actions[0].Kind != "choose_plan" || result.Actions[0].Target != "A" {
		t.Fatalf("single-plan confirmation was not resolved: %#v", result)
	}
}

func TestWorkspaceCommandDoesNotTreatMissionDeletionAsPlanConfirmation(t *testing.T) {
	fleet := domain.FleetSnapshotV2{Missions: []domain.MissionWorkspaceV2{{ID: "mission-1", Name: "Operation Balanced shoreline loop", Status: "planned"}}}
	request := domain.WorkspaceAssistantRequestV1{
		Text:            "Delete the current planned mission Operation Balanced shoreline loop.",
		ActiveMissionID: "mission-1",
		PlanOptions: []domain.WorkspacePlanOptionV1{{
			Label: "B", PlanID: "plan-b", Name: "Balanced shoreline loop", ContentHash: "hash-b", PolicyStatus: "approval_required",
		}},
	}
	result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), request, fleet)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Actions) != 1 || result.Actions[0].Kind != "delete_mission" || result.Actions[0].Target != "mission-1" {
		t.Fatalf("mission deletion was misclassified as plan confirmation: %#v", result)
	}
}

func TestWorkspaceCommandSupportsMissionAndGroupManagement(t *testing.T) {
	fleet := domain.FleetSnapshotV2{
		Missions: []domain.MissionWorkspaceV2{{ID: "mission-7", Name: "Harbor Lantern", Status: "executing"}},
		Groups:   []domain.OperationalGroupV2{{ID: "group-ws", Code: "WS", Name: "Watch Shoal", ColorName: "amber", MemberIDs: []string{"vessel-2"}}},
		Vessels:  []domain.VesselProfileV2{{ID: "vessel-1", Callsign: "Gannet", Designation: "KM-214", DisplayName: "Gannet (KM-214)"}},
	}
	tests := []struct {
		name, text, kind, target, secondary string
	}{
		{name: "pause mission", text: "Pause Harbor Lantern", kind: "pause_mission", target: "mission-7"},
		{name: "open mission", text: "Open Harbor Lantern", kind: "open_mission", target: "mission-7"},
		{name: "delete group", text: "Delete Watch Shoal group", kind: "delete_group", target: "group-ws"},
		{name: "move vessel", text: "Move Gannet to Watch Shoal", kind: "move_vessel_to_group", target: "vessel-1", secondary: "group-ws"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := NewManager(Config{}, slog.Default()).WorkspaceCommand(context.Background(), domain.WorkspaceAssistantRequestV1{Text: test.text}, fleet)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Actions) != 1 || result.Actions[0].Kind != test.kind || result.Actions[0].Target != test.target || result.Actions[0].SecondaryTarget != test.secondary {
				t.Fatalf("unexpected management action: %#v", result)
			}
		})
	}
}

func TestMissionOptionsReturnsOneStrategyUnlessAlternativesRequested(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"assistant_markdown\":\"Recommended intercept.\",\"geometry_option_id\":\"\",\"strategies\":[{\"id\":\"recommended\",\"name\":\"Predicted intercept\",\"description\":\"Meet the contact on its predicted track\",\"formation\":\"column\",\"speed_factor\":0.8,\"reserve_bias\":0.4,\"maneuvers\":[\"intercept predicted track\",\"match contact motion\"]},{\"id\":\"wide\",\"name\":\"Wide intercept\",\"description\":\"Use a wider intercept\",\"formation\":\"column\",\"speed_factor\":0.6,\"reserve_bias\":0.6,\"maneuvers\":[\"intercept wide\",\"match course\"]},{\"id\":\"reserve\",\"name\":\"Reserve intercept\",\"description\":\"Protect reserve\",\"formation\":\"column\",\"speed_factor\":0.5,\"reserve_bias\":0.8,\"maneuvers\":[\"intercept slowly\",\"match course\"]}]}"}]}]}`))
	}))
	defer provider.Close()
	manager := NewManager(Config{AIURL: "http://127.0.0.1:1", OpenAIKeyFile: keyFile, OpenAIModel: "gpt-5.6-luna", OpenAIURL: provider.URL}, slog.Default())
	result, err := manager.MissionOptions(context.Background(), domain.MissionPlanningContextV2{MissionID: "mission-1", Intent: "follow Safe Haven", TargetCount: 6, StrategyCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Strategies) != 1 || !strings.Contains(strings.ToLower(result.Summary), "confirm") {
		t.Fatalf("expected one confirmation-bound recommendation: %#v", result)
	}
}
