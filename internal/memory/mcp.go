package memory

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (m *Manager) MCPHandler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "keelmesh-memory-tools", Version: "v1"}, nil)
	tools := []struct{ name, description, schema string }{
		{"memory.search", "Search authorized semantic, procedural, and episodic memory with a retrieval receipt.", `{"type":"object","additionalProperties":false,"properties":{"query":{"type":"string","minLength":1,"maxLength":2000},"actor_identity":{"type":"string","minLength":1},"session_id":{"type":"string"},"mission_id":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":20}},"required":["query","actor_identity"]}`},
		{"memory.get", "Read one authorized memory item by immutable ID.", `{"type":"object","additionalProperties":false,"properties":{"id":{"type":"string","minLength":1},"actor_identity":{"type":"string","minLength":1}},"required":["id","actor_identity"]}`},
		{"memory.list_entities", "List memory graph entities visible to one actor.", `{"type":"object","additionalProperties":false,"properties":{"actor_identity":{"type":"string","minLength":1}},"required":["actor_identity"]}`},
		{"memory.get_context_receipt", "Read one actor-owned context assembly and its retrieval receipt.", `{"type":"object","additionalProperties":false,"properties":{"turn_id":{"type":"string","minLength":1},"actor_identity":{"type":"string","minLength":1}},"required":["turn_id","actor_identity"]}`},
		{"memory.draft_candidate", "Draft but never commit a scoped memory candidate.", `{"type":"object","additionalProperties":false,"properties":{"actor_identity":{"type":"string","minLength":1},"scope_kind":{"type":"string","enum":["operator","mission","vessel","group","faction","approved_global"]},"scope_id":{"type":"string","minLength":1},"kind":{"type":"string","minLength":1},"content":{"type":"string","minLength":1,"maxLength":24000},"source_id":{"type":"string","minLength":1},"confidence":{"type":"number","minimum":0,"maximum":1}},"required":["actor_identity","scope_kind","scope_id","kind","content","source_id","confidence"]}`},
	}
	for _, definition := range tools {
		definition := definition
		server.AddTool(&mcp.Tool{Name: definition.name, Description: definition.description, InputSchema: json.RawMessage(definition.schema)}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ID         string  `json:"id"`
				TurnID     string  `json:"turn_id"`
				Query      string  `json:"query"`
				Actor      string  `json:"actor_identity"`
				Session    string  `json:"session_id"`
				Mission    string  `json:"mission_id"`
				ScopeKind  string  `json:"scope_kind"`
				ScopeID    string  `json:"scope_id"`
				Kind       string  `json:"kind"`
				Content    string  `json:"content"`
				SourceID   string  `json:"source_id"`
				Confidence float64 `json:"confidence"`
				Limit      int     `json:"limit"`
			}
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return nil, err
			}
			// The incident-investigator token is bound server-side to the demo
			// operator. A model-supplied actor field cannot widen its scope.
			if args.Actor != "demo-operator" {
				return nil, problem("MEMORY_SCOPE_DENIED", "MCP token is not authorized for the requested actor scope.")
			}
			var value any
			var err error
			switch definition.name {
			case "memory.search":
				var hits []domain.RetrievalHitV2
				var receipt domain.RetrievalReceiptV1
				hits, receipt, err = m.Search(ctx, SearchRequest{Query: args.Query, ActorID: args.Actor, SessionID: args.Session, MissionID: args.Mission, Limit: args.Limit})
				value = map[string]any{"hits": hits, "receipt": receipt}
			case "memory.get":
				value, err = m.Item(args.ID, args.Actor)
			case "memory.list_entities":
				value = m.Entities(args.Actor)
			case "memory.get_context_receipt":
				value, err = m.Context(args.TurnID, args.Actor)
			case "memory.draft_candidate":
				value, err = m.DraftCandidate(domain.MemoryScopeV1{Kind: args.ScopeKind, ID: args.ScopeID}, args.Kind, args.Content, args.Actor, args.SourceID, args.Confidence)
			}
			if err != nil {
				return nil, err
			}
			raw, _ := json.Marshal(value)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil
		})
	}
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 64 << 10})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		expected := readSecret(m.cfg.MCPTokenFile)
		if expected == "" || len(token) != len(expected) || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="keelmesh-memory-mcp"`)
			http.Error(w, "MCP_UNAUTHORIZED", http.StatusUnauthorized)
			return
		}
		streamable.ServeHTTP(w, r)
	})
}
