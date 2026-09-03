package agent

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/arena"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/fleetops"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPControlHandler is the semantic control plane for external operator agents.
// It deliberately exposes no arbitrary HTTP, shell, filesystem, credential,
// mission-authorize, mission-start, or effect-application tool. Those effects
// pause at the same hash-bound human approval boundary used by the UI.
func (m *Manager) MCPControlHandler(fleet *fleetops.Manager, arenaManager *arena.Manager) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "keelmesh-operator-control", Version: "v1"}, nil)
	tools := []struct{ name, description, schema string }{
		{"system.get_capabilities", "List semantic tools and immutable approval boundaries.", emptySchema},
		{"fleet.get_snapshot", "Read the complete operator-visible fleet, groups, collections, missions, and environment.", emptySchema},
		{"fleet.get_vessel", "Read one operator-visible vessel profile.", idSchema("vessel_id")},
		{"fleet.get_reachability", "Read direct, relayed, and unreachable peers for one vessel.", idSchema("vessel_id")},
		{"mission.get", "Read one mission workspace.", idSchema("mission_id")},
		{"mission.get_trajectory", "Read hot tape, lifecycle, bounded adaptations, and execution cursor.", idSchema("mission_id")},
		{"mission.create_draft", "Create a persistent mission draft from exact vessel IDs; no movement authority is issued.", `{"type":"object","additionalProperties":false,"properties":{"request_id":{"type":"string","minLength":1},"idempotency_key":{"type":"string","minLength":1},"expected_version":{"type":"integer","minimum":1},"name":{"type":"string","maxLength":80},"objective":{"type":"string","maxLength":2000},"target_ids":{"type":"array","minItems":1,"maxItems":48,"uniqueItems":true,"items":{"type":"string"}}},"required":["request_id","idempotency_key","expected_version","objective","target_ids"]}`},
		{"mission.compile_intent", "Compile bounded natural-language intent into an immutable command draft.", `{"type":"object","additionalProperties":false,"properties":{"mission_id":{"type":"string"},"request_id":{"type":"string"},"idempotency_key":{"type":"string"},"expected_version":{"type":"integer","minimum":1},"text":{"type":"string","minLength":1,"maxLength":4000},"target_ids":{"type":"array","maxItems":48,"items":{"type":"string"}},"guidance_kind":{"type":"string"},"formation":{"type":"string"},"waypoints":{"type":"array","maxItems":64,"items":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"number"}}}},"required":["mission_id","request_id","idempotency_key","expected_version","text"]}`},
		{"mission.generate_plans", "Generate deterministic policy-checked candidates from an immutable draft.", `{"type":"object","additionalProperties":false,"properties":{"mission_id":{"type":"string"},"request_id":{"type":"string"},"idempotency_key":{"type":"string"},"expected_version":{"type":"integer","minimum":1},"draft_id":{"type":"string"}},"required":["mission_id","request_id","idempotency_key","expected_version","draft_id"]}`},
		{"mission.preview_plan", "Preview exact candidate trajectories without moving vessels.", `{"type":"object","additionalProperties":false,"properties":{"mission_id":{"type":"string"},"plan_id":{"type":"string"},"plan_hash":{"type":"string"},"request_id":{"type":"string"},"idempotency_key":{"type":"string"},"expected_version":{"type":"integer","minimum":1}},"required":["mission_id","plan_id","plan_hash","request_id","idempotency_key","expected_version"]}`},
		{"arena.get_player_state", "Read only one faction's server-filtered knowledge projection.", factionSchema},
		{"arena.get_infrastructure", "Read referee infrastructure state; deployment should reserve this tool for trusted diagnostic identities.", emptySchema},
		{"workspace.apply", "Apply a presentation-only workspace action such as selection, framing, windows, map view, or annotation.", workspaceSchema},
		{"effect.request_approval", "Describe the immutable human approval boundary for a proposed movement or simulated effect.", `{"type":"object","additionalProperties":false,"properties":{"effect_kind":{"type":"string"},"proposal_hash":{"type":"string"},"summary":{"type":"string","maxLength":1000}},"required":["effect_kind","proposal_hash","summary"]}`},
	}
	for _, item := range tools {
		item := item
		server.AddTool(&mcp.Tool{Name: item.name, Description: item.description, InputSchema: json.RawMessage(item.schema)}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return callControlTool(callCtx, fleet, arenaManager, item.name, req.Params.Arguments)
		})
	}
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 32 << 10})
	return bearerHandler(m.cfg.ControlTokenFile, streamable, m.recordSecurityDenial)
}

func bearerHandler(tokenFile string, next http.Handler, denied func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		expected := readSecret(tokenFile)
		if expected == "" || len(token) != len(expected) || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			denied()
			w.Header().Set("WWW-Authenticate", `Bearer realm="keelmesh-control-mcp"`)
			http.Error(w, "MCP_UNAUTHORIZED", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func callControlTool(ctx context.Context, fleet *fleetops.Manager, arenaManager *arena.Manager, name string, raw json.RawMessage) (*mcp.CallToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var args struct {
		VesselID        string              `json:"vessel_id"`
		MissionID       string              `json:"mission_id"`
		PlanID          string              `json:"plan_id"`
		PlanHash        string              `json:"plan_hash"`
		DraftID         string              `json:"draft_id"`
		RequestID       string              `json:"request_id"`
		IdempotencyKey  string              `json:"idempotency_key"`
		ExpectedVersion int64               `json:"expected_version"`
		Name            string              `json:"name"`
		Objective       string              `json:"objective"`
		Text            string              `json:"text"`
		GuidanceKind    string              `json:"guidance_kind"`
		Formation       string              `json:"formation"`
		TargetIDs       []string            `json:"target_ids"`
		Waypoints       []domain.GeoPointV2 `json:"waypoints"`
		Faction         string              `json:"faction"`
		SessionID       string              `json:"session_id"`
		ActorID         string              `json:"actor_id"`
		Kind            string              `json:"kind"`
		Arguments       map[string]any      `json:"arguments"`
		EffectKind      string              `json:"effect_kind"`
		ProposalHash    string              `json:"proposal_hash"`
		Summary         string              `json:"summary"`
	}
	_ = json.Unmarshal(raw, &args)
	mutation := fleetops.Mutation{RequestID: args.RequestID, IdempotencyKey: args.IdempotencyKey, ExpectedVersion: args.ExpectedVersion}
	var value any
	var err error
	switch name {
	case "system.get_capabilities":
		value = map[string]any{"boundary": "typed semantic tools", "presentation": "automatic", "drafting": "automatic within supplied identity and versions", "effects": "human exact-hash approval required", "forbidden": []string{"shell", "filesystem", "secrets", "arbitrary_url", "mission_authorize", "mission_start", "apply_effect"}}
	case "fleet.get_snapshot":
		value = fleet.Snapshot()
	case "fleet.get_vessel":
		value, err = fleet.Vessel(args.VesselID)
	case "fleet.get_reachability":
		value, err = fleet.Reachability(args.VesselID)
	case "mission.get":
		s := fleet.Snapshot()
		err = problem("MISSION_NOT_FOUND", "Mission not found.")
		for _, mission := range s.Missions {
			if mission.ID == args.MissionID {
				value, err = mission, nil
				break
			}
		}
	case "mission.get_trajectory":
		value, err = fleet.TrajectoryProgram(args.MissionID)
	case "mission.create_draft":
		value, err = fleet.CreateMission(fleetops.CreateMissionRequest{Mutation: mutation, Name: args.Name, NamingMode: "ai", Objective: args.Objective, TargetIDs: args.TargetIDs})
	case "mission.compile_intent":
		value, err = fleet.Compile(args.MissionID, fleetops.CompileRequest{Mutation: mutation, Text: args.Text, TargetIDs: args.TargetIDs, GuidanceKind: args.GuidanceKind, Waypoints: args.Waypoints, Formation: args.Formation})
	case "mission.generate_plans":
		value, err = fleet.GeneratePlans(args.MissionID, fleetops.PlansRequest{Mutation: mutation, DraftID: args.DraftID})
	case "mission.preview_plan":
		value, err = fleet.Preview(args.MissionID, args.PlanID, fleetops.PlanActionRequest{Mutation: mutation, PlanHash: args.PlanHash})
	case "arena.get_player_state":
		value = arenaManager.Snapshot(args.Faction)
	case "arena.get_infrastructure":
		value = arenaManager.InfrastructureSnapshot()
	case "workspace.apply":
		value, err = arenaManager.WorkspaceAction(arena.ActionRequest{ArenaMutationV1: domain.ArenaMutationV1{RequestID: args.RequestID, IdempotencyKey: args.IdempotencyKey, ExpectedVersion: args.ExpectedVersion, ActorID: args.ActorID}, SessionID: args.SessionID, Faction: args.Faction, Kind: args.Kind, Arguments: args.Arguments})
	case "effect.request_approval":
		value = map[string]any{"state": "awaiting_human_approval", "code": "HUMAN_APPROVAL_REQUIRED", "effect_kind": args.EffectKind, "proposal_hash": args.ProposalHash, "summary": args.Summary}
	}
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil
}

const emptySchema = `{"type":"object","additionalProperties":false}`

func idSchema(name string) string {
	return `{"type":"object","additionalProperties":false,"properties":{"` + name + `":{"type":"string","minLength":1}},"required":["` + name + `"]}`
}

const factionSchema = `{"type":"object","additionalProperties":false,"properties":{"faction":{"type":"string","enum":["A","B"]}},"required":["faction"]}`
const workspaceSchema = `{"type":"object","additionalProperties":false,"properties":{"request_id":{"type":"string"},"idempotency_key":{"type":"string"},"expected_version":{"type":"integer","minimum":1},"actor_id":{"type":"string"},"session_id":{"type":"string"},"faction":{"type":"string","enum":["A","B"]},"kind":{"type":"string","enum":["select","open_window","close_window","minimize_window","move_window","set_map_view","annotate_map","frame_group"]},"arguments":{"type":"object"}},"required":["request_id","idempotency_key","expected_version","actor_id","session_id","faction","kind","arguments"]}`
