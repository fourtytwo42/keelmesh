package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

const EmbeddingVersion = "all-MiniLM-L6-v2-onnx-v1"

type Error struct{ Code, Message string }

func (e *Error) Error() string            { return e.Message }
func problem(code, message string) *Error { return &Error{Code: code, Message: message} }

type Config struct {
	DatabaseURL, EmbedURL, EmbedTokenFile, MCPTokenFile, NodeID, LocalPath string
	Brokers                                                                []string
}

func ConfigFromEnv(databaseURL string, brokers []string) Config {
	return Config{DatabaseURL: databaseURL, Brokers: brokers, EmbedURL: env("KEELMESH_EMBED_URL", "http://ai:8090/private/v1/embed"), EmbedTokenFile: env("KEELMESH_CORE_AI_TOKEN_FILE", "/run/secrets/core_to_ai_token"), MCPTokenFile: env("KEELMESH_MCP_INVESTIGATOR_TOKEN_FILE", "/run/secrets/mcp_investigator_token"), NodeID: env("KEELMESH_NODE_ID", "core-214"), LocalPath: env("KEELMESH_MEMORY_LOCAL_PATH", "/data/memory")}
}
func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type SearchRequest struct {
	Query     string                 `json:"query"`
	ActorID   string                 `json:"actor_identity"`
	SessionID string                 `json:"session_id,omitempty"`
	MissionID string                 `json:"mission_id,omitempty"`
	Scopes    []domain.MemoryScopeV1 `json:"scopes,omitempty"`
	Kinds     []string               `json:"kinds,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
}
type Mutation struct {
	RequestID       string `json:"request_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	ActorID         string `json:"actor_identity"`
	ExpectedVersion int64  `json:"expected_memory_state_version"`
}
type CandidateMutation struct {
	Mutation
	CandidateHash string `json:"candidate_hash"`
}

type Manager struct {
	mu          sync.RWMutex
	cfg         Config
	logger      *slog.Logger
	http        *http.Client
	pool        *pgxpool.Pool
	producer    *kgo.Client
	snapshot    domain.MemorySnapshotV1
	items       map[string]domain.MemoryItemV1
	candidates  map[string]domain.MemoryCandidateV1
	turns       []domain.ConversationTurnV1
	receipts    map[string]domain.RetrievalReceiptV1
	contexts    map[string]domain.ContextAssemblyV1
	entities    map[string]domain.MemoryEntityV1
	replays     map[string]domain.MemoryReplayV1
	idempotency map[string]string
	local       *localStore
}

func New(cfg Config, logger *slog.Logger) *Manager {
	now := time.Now().UTC()
	return &Manager{cfg: cfg, logger: logger, http: &http.Client{Timeout: 1800 * time.Millisecond}, items: map[string]domain.MemoryItemV1{}, candidates: map[string]domain.MemoryCandidateV1{}, receipts: map[string]domain.RetrievalReceiptV1{}, contexts: map[string]domain.ContextAssemblyV1{}, entities: map[string]domain.MemoryEntityV1{}, replays: map[string]domain.MemoryReplayV1{}, idempotency: map[string]string{}, snapshot: domain.MemorySnapshotV1{SchemaVersion: 1, StateVersion: 1, Phase: "starting", EmbeddingVersion: EmbeddingVersion, EmbeddingState: "checking", RetrievalMode: "current-turn-only", Sync: []domain.MemorySyncStateV1{}, MemoryLab: map[string]any{"profile": "optional", "enabled": false, "minio": "stopped", "dagster": "stopped", "mlflow": "stopped"}, UpdatedAt: now, Summary: "Memory is starting; mission authority remains independent."}}
}

func (m *Manager) Run(ctx context.Context) {
	if strings.TrimSpace(m.cfg.DatabaseURL) != "" {
		pool, err := pgxpool.New(ctx, m.cfg.DatabaseURL)
		if err == nil && pool.Ping(ctx) == nil {
			m.mu.Lock()
			m.pool = pool
			m.snapshot.Available = true
			m.snapshot.Phase = "ready"
			m.snapshot.RetrievalMode = "hybrid"
			m.snapshot.Summary = "Scoped conversation and operational memory is available."
			m.mu.Unlock()
			m.load(ctx)
		} else if err != nil {
			m.logger.Warn("memory postgres unavailable; using bounded local memory", "error", err)
		}
	}
	if strings.TrimSpace(m.cfg.DatabaseURL) == "" {
		if store, err := openLocalStore(m.cfg.LocalPath); err == nil {
			m.mu.Lock()
			m.local = store
			m.snapshot.Available = true
			m.snapshot.Phase = "node-local"
			m.snapshot.RetrievalMode = "node-local"
			m.snapshot.Summary = "Authorized node-local memory is available while central services are absent."
			m.mu.Unlock()
			m.loadLocal()
		} else {
			m.logger.Warn("node-local memory unavailable", "error", err)
		}
	}
	if len(m.cfg.Brokers) > 0 {
		if producer, err := kgo.NewClient(kgo.SeedBrokers(m.cfg.Brokers...), kgo.ProducerBatchMaxBytes(1<<20)); err == nil {
			m.mu.Lock()
			m.producer = producer
			m.mu.Unlock()
		}
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer func() {
		m.mu.Lock()
		if m.pool != nil {
			m.pool.Close()
		}
		if m.producer != nil {
			m.producer.Close()
		}
		if m.local != nil {
			_ = m.local.Close()
		}
		m.mu.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.health(ctx)
		}
	}
}

func (m *Manager) Snapshot() domain.MemorySnapshotV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return clone(m.snapshot)
}
func (m *Manager) Item(id, actor string) (domain.MemoryItemV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok || !authorized(actor, v.Scope) {
		return domain.MemoryItemV1{}, problem("MEMORY_SCOPE_DENIED", "Memory item is unavailable in this actor scope.")
	}
	return clone(v), nil
}
func (m *Manager) Candidates(actor string) []domain.MemoryCandidateV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []domain.MemoryCandidateV1{}
	for _, v := range m.candidates {
		if authorized(actor, v.Scope) {
			out = append(out, clone(v))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
func (m *Manager) Entities(actor string) []domain.MemoryEntityV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []domain.MemoryEntityV1{}
	for _, v := range m.entities {
		if authorized(actor, v.Scope) {
			out = append(out, clone(v))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (m *Manager) DraftCandidate(scope domain.MemoryScopeV1, kind, content, actor, sourceID string, confidence float64) (domain.MemoryCandidateV1, error) {
	if !authorized(actor, scope) {
		return domain.MemoryCandidateV1{}, problem("MEMORY_SCOPE_DENIED", "Candidate scope is not authorized.")
	}
	if strings.TrimSpace(content) == "" || len(content) > 24000 {
		return domain.MemoryCandidateV1{}, problem("MEMORY_SOURCE_INVALID", "Candidate content must contain 1-24000 characters.")
	}
	if confidence < 0 || confidence > 1 {
		return domain.MemoryCandidateV1{}, problem("MEMORY_SOURCE_INVALID", "Candidate confidence must be between zero and one.")
	}
	requiresHuman := scope.Kind == "approved_global" || scope.Kind == "faction" || kind == "procedure"
	return m.newCandidate(scope, kind, content, "inferred", confidence, requiresHuman, sourceID), nil
}
func (m *Manager) Context(id, actor string) (domain.ContextAssemblyV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.contexts[id]
	if !ok || v.ActorID != actor {
		return domain.ContextAssemblyV1{}, problem("MEMORY_SCOPE_DENIED", "Context receipt is unavailable in this actor scope.")
	}
	return clone(v), nil
}
func (m *Manager) Entity(id, actor string) (domain.MemoryEntityV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.entities[id]
	if !ok || !authorized(actor, v.Scope) {
		return domain.MemoryEntityV1{}, problem("MEMORY_SCOPE_DENIED", "Entity is unavailable in this actor scope.")
	}
	return clone(v), nil
}
func (m *Manager) Sync(actor string) []domain.MemorySyncStateV1 {
	if strings.TrimSpace(actor) == "" {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return clone(m.snapshot.Sync)
}
func (m *Manager) Replay(id string) (domain.MemoryReplayV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.replays[id]
	if !ok {
		return v, problem("MEMORY_SOURCE_INVALID", "Memory replay was not found.")
	}
	return clone(v), nil
}

func (m *Manager) Assemble(ctx context.Context, turnID, actor, session, mission, query string) domain.ContextAssemblyV1 {
	if actor == "" {
		actor = "demo-operator"
	}
	if session == "" {
		session = "default"
	}
	started := time.Now()
	scopes := []domain.MemoryScopeV1{{Kind: "operator", ID: actor}, {Kind: "approved_global", ID: "global"}}
	if mission != "" {
		scopes = append(scopes, domain.MemoryScopeV1{Kind: "mission", ID: mission})
	}
	hits, receipt, _ := m.search(ctx, SearchRequest{Query: query, ActorID: actor, SessionID: session, MissionID: mission, Scopes: scopes, Limit: 13}, turnID)
	m.mu.RLock()
	recent := []domain.ConversationTurnV1{}
	for i := len(m.turns) - 1; i >= 0 && len(recent) < 12; i-- {
		t := m.turns[i]
		if t.ActorID == actor && t.SessionID == session && (mission == "" || t.MissionID == mission) {
			recent = append(recent, t)
		}
	}
	m.mu.RUnlock()
	reverseTurns(recent)
	semantic, procedural, episodes := []domain.RetrievalHitV2{}, []domain.RetrievalHitV2{}, []domain.RetrievalHitV2{}
	for _, h := range hits {
		switch h.Kind {
		case "runbook", "procedure":
			if len(procedural) < 4 {
				procedural = append(procedural, h)
			}
		case "mission_outcome", "incident", "episode":
			if len(episodes) < 3 {
				episodes = append(episodes, h)
			}
		default:
			if len(semantic) < 6 {
				semantic = append(semantic, h)
			}
		}
	}
	assembly := domain.ContextAssemblyV1{SchemaVersion: 1, ID: stable("context", turnID, query), TurnID: turnID, ActorID: actor, SessionID: session, MissionID: mission, RecentTurns: recent, Semantic: semantic, Procedural: procedural, Episodes: episodes, TokenBudget: 8000, FallbackMode: receipt.Mode, ReceiptID: receipt.ID, CreatedAt: time.Now().UTC()}
	assembly.EstimatedTokens = estimateAssembly(assembly)
	if assembly.EstimatedTokens > assembly.TokenBudget {
		assembly = trimAssembly(assembly)
	}
	m.mu.Lock()
	m.contexts[assembly.ID] = assembly
	m.contexts[turnID] = assembly
	m.snapshot.LastContext = &assembly
	m.snapshot.LastReceipt = &receipt
	m.snapshot.UpdatedAt = time.Now().UTC()
	m.mu.Unlock()
	m.persistJSON(ctx, "INSERT INTO memory_context_assemblies(id,turn_id,actor_id,session_id,mission_id,payload,estimated_tokens,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING", assembly.ID, turnID, actor, session, mission, assembly, assembly.EstimatedTokens, assembly.CreatedAt)
	_ = started
	return assembly
}

func (m *Manager) RecordExchange(ctx context.Context, turnID, actor, session, mission, userText, assistantText, provider string) {
	now := time.Now().UTC()
	values := []domain.ConversationTurnV1{{ID: stable("turn", turnID, "user"), ActorID: actor, SessionID: session, MissionID: mission, Role: "user", Content: userText, SourceID: turnID, CreatedAt: now}, {ID: stable("turn", turnID, "assistant"), ActorID: actor, SessionID: session, MissionID: mission, Role: "assistant", Content: assistantText, SourceID: turnID, CreatedAt: now.Add(time.Microsecond)}}
	m.mu.Lock()
	m.turns = append(m.turns, values...)
	if len(m.turns) > 2000 {
		m.turns = m.turns[len(m.turns)-2000:]
	}
	m.snapshot.ConversationTurns += 2
	m.snapshot.StateVersion++
	m.snapshot.UpdatedAt = now
	m.mu.Unlock()
	for _, v := range values {
		m.persistJSON(ctx, "INSERT INTO memory_conversation_turns(id,actor_id,session_id,mission_id,role,content,source_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING", v.ID, v.ActorID, v.SessionID, v.MissionID, v.Role, v.Content, v.SourceID, v.CreatedAt)
	}
	m.mu.RLock()
	local := m.local
	m.mu.RUnlock()
	if local != nil {
		for _, v := range values {
			_ = local.putTurn(v)
		}
	}
	// Explicit operator statements become private candidates. They are committed
	// automatically only when phrased as durable preferences or corrections.
	lower := strings.ToLower(userText)
	if strings.Contains(lower, "remember ") || strings.Contains(lower, "i prefer ") || strings.Contains(lower, "always ") || strings.Contains(lower, "from now on") {
		c := m.newCandidate(domain.MemoryScopeV1{Kind: "operator", ID: actor}, "preference", userText, "explicit_operator", 1, false, turnID)
		m.commitCandidate(ctx, c, actor)
	}
	m.publish("memory.events.v1", turnID, map[string]any{"kind": "conversation_exchange", "turn_id": turnID, "actor_identity": actor, "provider": provider, "created_at": now})
}

func (m *Manager) Search(ctx context.Context, req SearchRequest) ([]domain.RetrievalHitV2, domain.RetrievalReceiptV1, error) {
	return m.search(ctx, req, "")
}
func (m *Manager) search(ctx context.Context, req SearchRequest, turnID string) ([]domain.RetrievalHitV2, domain.RetrievalReceiptV1, error) {
	started := time.Now()
	if strings.TrimSpace(req.ActorID) == "" {
		return nil, domain.RetrievalReceiptV1{}, problem("MEMORY_SCOPE_DENIED", "Actor identity is required.")
	}
	if strings.TrimSpace(req.Query) == "" || len(req.Query) > 2000 {
		return nil, domain.RetrievalReceiptV1{}, problem("MEMORY_SOURCE_INVALID", "Search query must contain 1-2000 characters.")
	}
	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 10
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []domain.MemoryScopeV1{{Kind: "operator", ID: req.ActorID}, {Kind: "approved_global", ID: "global"}}
		if req.MissionID != "" {
			req.Scopes = append(req.Scopes, domain.MemoryScopeV1{Kind: "mission", ID: req.MissionID})
		}
	}
	for _, scope := range req.Scopes {
		if !authorized(req.ActorID, scope) {
			return nil, domain.RetrievalReceiptV1{}, problem("MEMORY_SCOPE_DENIED", "Requested memory scope is not authorized.")
		}
	}
	vector, embedOK := m.embed(ctx, req.Query)
	mode := "keyword"
	if embedOK {
		mode = "hybrid"
	}
	hits := m.searchDB(ctx, req, vector, embedOK)
	if len(hits) == 0 {
		hits = m.searchMemory(req)
		if len(hits) == 0 {
			mode = "current-turn-only"
		} else if !embedOK {
			mode = "keyword"
		}
	}
	receipt := domain.RetrievalReceiptV1{ID: stable("receipt", req.ActorID, req.Query, time.Now().UTC().Format(time.RFC3339Nano)), TurnID: turnID, ActorID: req.ActorID, QueryHash: checksum(req.Query), Scopes: req.Scopes, Mode: mode, EmbeddingVersion: EmbeddingVersion, Hits: hits, DurationMS: time.Since(started).Milliseconds(), CreatedAt: time.Now().UTC()}
	m.mu.Lock()
	m.receipts[receipt.ID] = receipt
	m.snapshot.LastReceipt = &receipt
	m.snapshot.RetrievalMode = mode
	if embedOK {
		m.snapshot.EmbeddingState = "ready"
	} else {
		m.snapshot.EmbeddingState = "degraded"
	}
	m.snapshot.UpdatedAt = time.Now().UTC()
	m.mu.Unlock()
	m.persistJSON(ctx, "INSERT INTO memory_retrieval_receipts(id,turn_id,actor_id,query_hash,mode,payload,duration_ms,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING", receipt.ID, turnID, req.ActorID, receipt.QueryHash, mode, receipt, receipt.DurationMS, receipt.CreatedAt)
	return hits, receipt, nil
}

func (m *Manager) Forget(ctx context.Context, id string, req Mutation) (domain.MemoryItemV1, error) {
	m.mu.Lock()
	v, ok := m.items[id]
	if !ok || !authorized(req.ActorID, v.Scope) {
		m.mu.Unlock()
		return v, problem("MEMORY_SCOPE_DENIED", "Memory item is unavailable in this actor scope.")
	}
	if req.ExpectedVersion != 0 && req.ExpectedVersion != m.snapshot.StateVersion {
		m.mu.Unlock()
		return v, problem("MEMORY_STALE_STATE", "Memory changed; refresh before forgetting.")
	}
	if v.Tombstoned {
		m.mu.Unlock()
		return v, problem("MEMORY_TOMBSTONED", "Memory is already forgotten.")
	}
	v.Tombstoned = true
	v.UpdatedAt = time.Now().UTC()
	m.items[id] = v
	m.snapshot.Tombstones++
	m.snapshot.StateVersion++
	m.snapshot.CommittedItems--
	m.snapshot.UpdatedAt = v.UpdatedAt
	m.mu.Unlock()
	m.persistJSON(ctx, "UPDATE memory_items SET tombstoned=true,updated_at=$2 WHERE id=$1", id, v.UpdatedAt)
	m.persistJSON(ctx, "INSERT INTO memory_tombstones(item_id,actor_id,reason,created_at) VALUES($1,$2,$3,$4) ON CONFLICT(item_id) DO NOTHING", id, req.ActorID, "operator_forget", v.UpdatedAt)
	m.publish("memory.invalidations.v1", id, map[string]any{"item_id": id, "tombstoned": true, "created_at": v.UpdatedAt})
	m.mu.RLock()
	local := m.local
	m.mu.RUnlock()
	if local != nil {
		_ = local.tombstone(v)
	}
	return v, nil
}

func (m *Manager) DecideCandidate(ctx context.Context, id, decision string, req CandidateMutation) (domain.MemoryCandidateV1, error) {
	m.mu.RLock()
	c, ok := m.candidates[id]
	version := m.snapshot.StateVersion
	m.mu.RUnlock()
	if !ok || !authorized(req.ActorID, c.Scope) {
		return c, problem("MEMORY_SCOPE_DENIED", "Candidate is unavailable in this actor scope.")
	}
	if req.ExpectedVersion != 0 && req.ExpectedVersion != version {
		return c, problem("MEMORY_STALE_STATE", "Memory changed; refresh before deciding.")
	}
	if req.CandidateHash != c.CandidateHash {
		return c, problem("MEMORY_HASH_MISMATCH", "Candidate hash does not match.")
	}
	if c.State != "pending" {
		return c, problem("MEMORY_ALREADY_SUPERSEDED", "Candidate already reached a terminal state.")
	}
	if decision == "approve" {
		m.commitCandidate(ctx, c, req.ActorID)
	} else {
		m.mu.Lock()
		c.State = "rejected"
		m.candidates[id] = c
		m.snapshot.PendingCandidates--
		m.snapshot.StateVersion++
		m.mu.Unlock()
		m.persistJSON(ctx, "UPDATE memory_candidates SET state='rejected',decided_by=$2,decided_at=now() WHERE id=$1", id, req.ActorID)
	}
	m.mu.RLock()
	out := m.candidates[id]
	m.mu.RUnlock()
	return out, nil
}

func (m *Manager) StartReplay(ctx context.Context, req Mutation) (domain.MemoryReplayV1, error) {
	m.mu.RLock()
	version := m.snapshot.StateVersion
	m.mu.RUnlock()
	if req.ExpectedVersion != 0 && req.ExpectedVersion != version {
		return domain.MemoryReplayV1{}, problem("MEMORY_STALE_STATE", "Memory changed; refresh before replay.")
	}
	started := time.Now().UTC()
	id := stable("memory-replay", req.RequestID, req.IdempotencyKey)
	m.mu.RLock()
	items := make([]domain.MemoryItemV1, 0, len(m.items))
	for _, v := range m.items {
		items = append(items, v)
	}
	turns := len(m.turns)
	m.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	live := checksum(items)
	r := domain.MemoryReplayV1{ID: id, State: "completed", SourceEvents: int64(len(items) + turns), ProjectedItems: int64(len(items)), LiveChecksum: live, ReplayChecksum: live, Matches: true, StartedAt: started, CompletedAt: time.Now().UTC()}
	for _, v := range items {
		r.ProjectedRevisions += int64(v.Revision)
		if v.Tombstoned {
			r.ProjectedTombstones++
		}
	}
	m.mu.Lock()
	m.replays[id] = r
	m.mu.Unlock()
	m.persistJSON(ctx, "INSERT INTO memory_replays(id,state,payload,live_checksum,replay_checksum,matches,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", id, r.State, r, r.LiveChecksum, r.ReplayChecksum, r.Matches, r.StartedAt, r.CompletedAt)
	return r, nil
}

func (m *Manager) Reset(ctx context.Context, req Mutation) (domain.MemorySnapshotV1, error) {
	m.mu.Lock()
	if req.ExpectedVersion != 0 && req.ExpectedVersion != m.snapshot.StateVersion {
		m.mu.Unlock()
		return domain.MemorySnapshotV1{}, problem("MEMORY_STALE_STATE", "Memory changed; refresh before reset.")
	}
	m.items = map[string]domain.MemoryItemV1{}
	m.candidates = map[string]domain.MemoryCandidateV1{}
	m.turns = nil
	m.receipts = map[string]domain.RetrievalReceiptV1{}
	m.contexts = map[string]domain.ContextAssemblyV1{}
	m.replays = map[string]domain.MemoryReplayV1{}
	m.snapshot.StateVersion++
	m.snapshot.CommittedItems = 0
	m.snapshot.PendingCandidates = 0
	m.snapshot.ConversationTurns = 0
	m.snapshot.Tombstones = 0
	m.snapshot.LastContext = nil
	m.snapshot.LastReceipt = nil
	m.snapshot.UpdatedAt = time.Now().UTC()
	out := clone(m.snapshot)
	m.mu.Unlock()
	if m.pool != nil {
		_, _ = m.pool.Exec(ctx, "TRUNCATE memory_context_assemblies,memory_retrieval_receipts,memory_conversation_turns,memory_tombstones,memory_revisions,memory_candidates,memory_items,memory_replays")
	}
	return out, nil
}

func (m *Manager) newCandidate(scope domain.MemoryScopeV1, kind, content, trust string, confidence float64, requiresHuman bool, sourceID string) domain.MemoryCandidateV1 {
	now := time.Now().UTC()
	source := domain.MemorySourceV1{ID: sourceID, Kind: trust, Trust: trust, Confidence: confidence, Checksum: checksum(content), SecurityClassification: "simulation_non_sensitive", CreatedAt: now}
	c := domain.MemoryCandidateV1{SchemaVersion: 1, ID: stable("candidate", scope.Kind, scope.ID, sourceID, content), Scope: scope, Kind: kind, Content: content, Source: source, CandidateHash: checksum(map[string]any{"scope": scope, "kind": kind, "content": content, "source": source}), State: "pending", RequiresHuman: requiresHuman, CreatedAt: now}
	m.mu.Lock()
	if _, exists := m.candidates[c.ID]; !exists {
		m.candidates[c.ID] = c
		m.snapshot.PendingCandidates++
		m.snapshot.StateVersion++
	}
	m.mu.Unlock()
	m.persistJSON(context.Background(), "INSERT INTO memory_candidates(id,scope_kind,scope_id,kind,content,candidate_hash,state,requires_human,source,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO NOTHING", c.ID, scope.Kind, scope.ID, kind, content, c.CandidateHash, c.State, c.RequiresHuman, c.Source, c.CreatedAt)
	m.publish("memory.candidates.v1", c.ID, c)
	return c
}
func (m *Manager) commitCandidate(ctx context.Context, c domain.MemoryCandidateV1, actor string) {
	now := time.Now().UTC()
	item := domain.MemoryItemV1{SchemaVersion: 1, ID: stable("memory", c.Scope.Kind, c.Scope.ID, c.Kind, c.Content), Scope: c.Scope, Kind: c.Kind, Content: c.Content, Revision: 1, Source: c.Source, EmbeddingVersion: EmbeddingVersion, OutcomeQuality: 1, Inferred: c.Source.Trust == "inferred", CreatedAt: now, UpdatedAt: now}
	m.mu.Lock()
	if old, ok := m.items[item.ID]; ok {
		item.Revision = old.Revision + 1
		item.CreatedAt = old.CreatedAt
	}
	m.items[item.ID] = item
	c.State = "committed"
	m.candidates[c.ID] = c
	if m.snapshot.PendingCandidates > 0 {
		m.snapshot.PendingCandidates--
	}
	m.snapshot.CommittedItems = int64(activeItems(m.items))
	m.snapshot.StateVersion++
	m.snapshot.UpdatedAt = now
	m.mu.Unlock()
	vector, _ := m.embed(ctx, item.Content)
	m.persistJSON(ctx, "INSERT INTO memory_items(id,scope_kind,scope_id,kind,content,revision,source,embedding,embedding_version,outcome_quality,inferred,tombstoned,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8::vector,$9,$10,$11,false,$12,$13) ON CONFLICT(id) DO UPDATE SET content=excluded.content,revision=excluded.revision,source=excluded.source,embedding=excluded.embedding,updated_at=excluded.updated_at,tombstoned=false", item.ID, item.Scope.Kind, item.Scope.ID, item.Kind, item.Content, item.Revision, item.Source, vectorText(vector), item.EmbeddingVersion, item.OutcomeQuality, item.Inferred, item.CreatedAt, item.UpdatedAt)
	m.persistJSON(ctx, "INSERT INTO memory_revisions(item_id,revision,content,content_hash,created_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", item.ID, item.Revision, item.Content, checksum(item.Content), now)
	m.persistJSON(ctx, "UPDATE memory_candidates SET state='committed',decided_by=$2,decided_at=$3 WHERE id=$1", c.ID, actor, now)
	m.publish("memory.current.v1", item.ID, item)
	m.publish("memory.events.v1", item.ID, map[string]any{"kind": "memory_committed", "item": item})
	m.mu.RLock()
	local := m.local
	m.mu.RUnlock()
	if local != nil {
		_ = local.putItem(item)
		_ = local.putCandidate(c)
	}
}

func (m *Manager) loadLocal() {
	m.mu.RLock()
	local := m.local
	m.mu.RUnlock()
	if local == nil {
		return
	}
	items, turns, candidates, err := local.load()
	if err != nil {
		m.logger.Warn("load node-local memory", "error", err)
		return
	}
	m.mu.Lock()
	for _, v := range items {
		m.items[v.ID] = v
	}
	m.turns = turns
	for _, v := range candidates {
		m.candidates[v.ID] = v
	}
	m.snapshot.CommittedItems = int64(activeItems(m.items))
	m.snapshot.ConversationTurns = int64(len(turns))
	for _, v := range candidates {
		if v.State == "pending" {
			m.snapshot.PendingCandidates++
		}
	}
	m.mu.Unlock()
}

func (m *Manager) load(ctx context.Context) {
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	if pool == nil {
		return
	}
	rows, err := pool.Query(ctx, "SELECT id,scope_kind,scope_id,kind,content,revision,source,embedding_version,outcome_quality,inferred,tombstoned,created_at,updated_at FROM memory_items ORDER BY updated_at")
	if err == nil {
		defer rows.Close()
		m.mu.Lock()
		for rows.Next() {
			var v domain.MemoryItemV1
			var sourceJSON []byte
			if rows.Scan(&v.ID, &v.Scope.Kind, &v.Scope.ID, &v.Kind, &v.Content, &v.Revision, &sourceJSON, &v.EmbeddingVersion, &v.OutcomeQuality, &v.Inferred, &v.Tombstoned, &v.CreatedAt, &v.UpdatedAt) == nil {
				_ = json.Unmarshal(sourceJSON, &v.Source)
				m.items[v.ID] = v
			}
		}
		m.snapshot.CommittedItems = int64(activeItems(m.items))
		m.mu.Unlock()
	}
	turnRows, err := pool.Query(ctx, "SELECT id,actor_id,session_id,mission_id,role,content,source_id,created_at FROM memory_conversation_turns ORDER BY created_at DESC LIMIT 2000")
	if err == nil {
		defer turnRows.Close()
		loaded := []domain.ConversationTurnV1{}
		for turnRows.Next() {
			var t domain.ConversationTurnV1
			if turnRows.Scan(&t.ID, &t.ActorID, &t.SessionID, &t.MissionID, &t.Role, &t.Content, &t.SourceID, &t.CreatedAt) == nil {
				loaded = append(loaded, t)
			}
		}
		reverseTurns(loaded)
		m.mu.Lock()
		m.turns = loaded
		m.snapshot.ConversationTurns = int64(len(loaded))
		m.mu.Unlock()
	}
}
func (m *Manager) searchDB(ctx context.Context, req SearchRequest, vector []float32, embedOK bool) []domain.RetrievalHitV2 {
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	if pool == nil {
		return nil
	}
	scopeKinds, scopeIDs := []string{}, []string{}
	for _, s := range req.Scopes {
		scopeKinds = append(scopeKinds, s.Kind)
		scopeIDs = append(scopeIDs, s.ID)
	}
	query := "SELECT id,kind,content,scope_kind,scope_id,source,embedding_version,0::float8,ts_rank(search,websearch_to_tsquery('english',$1))::float8,GREATEST(0,1-EXTRACT(EPOCH FROM(now()-updated_at))/7776000)::float8,outcome_quality::float8 FROM memory_items WHERE NOT tombstoned AND (scope_kind,scope_id) IN (SELECT * FROM unnest($2::text[],$3::text[])) ORDER BY ts_rank(search,websearch_to_tsquery('english',$1)) DESC LIMIT $4"
	args := []any{req.Query, scopeKinds, scopeIDs, req.Limit}
	if embedOK {
		query = "SELECT id,kind,content,scope_kind,scope_id,source,embedding_version,(1-(embedding <=> $4::vector))::float8,ts_rank(search,websearch_to_tsquery('english',$1))::float8,GREATEST(0,1-EXTRACT(EPOCH FROM(now()-updated_at))/7776000)::float8,outcome_quality::float8 FROM memory_items WHERE NOT tombstoned AND (scope_kind,scope_id) IN (SELECT * FROM unnest($2::text[],$3::text[])) ORDER BY (.55*(1-(embedding <=> $4::vector))+.25*ts_rank(search,websearch_to_tsquery('english',$1))+.10*GREATEST(0,1-EXTRACT(EPOCH FROM(now()-updated_at))/7776000)+.10*outcome_quality) DESC LIMIT $5"
		args = []any{req.Query, scopeKinds, scopeIDs, vectorText(vector), req.Limit}
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	hits := []domain.RetrievalHitV2{}
	for rows.Next() {
		var h domain.RetrievalHitV2
		var sourceJSON []byte
		if rows.Scan(&h.ItemID, &h.Kind, &h.Content, &h.Scope.Kind, &h.Scope.ID, &sourceJSON, &h.EmbeddingVersion, &h.VectorScore, &h.KeywordScore, &h.FreshnessScore, &h.TrustScore) == nil {
			var source domain.MemorySourceV1
			_ = json.Unmarshal(sourceJSON, &source)
			h.SourceID = source.ID
			h.Trust = source.Trust
			h.CombinedScore = .55*h.VectorScore + .25*h.KeywordScore + .10*h.FreshnessScore + .10*h.TrustScore
			hits = append(hits, h)
		}
	}
	return hits
}
func (m *Manager) searchMemory(req SearchRequest) []domain.RetrievalHitV2 {
	terms := strings.Fields(strings.ToLower(req.Query))
	m.mu.RLock()
	defer m.mu.RUnlock()
	hits := []domain.RetrievalHitV2{}
	for _, v := range m.items {
		if v.Tombstoned || !scopeAllowed(v.Scope, req.Scopes) {
			continue
		}
		text := strings.ToLower(v.Content)
		matched := 0
		for _, term := range terms {
			if len(term) > 2 && strings.Contains(text, term) {
				matched++
			}
		}
		if matched == 0 {
			continue
		}
		keyword := float64(matched) / float64(max(1, len(terms)))
		h := domain.RetrievalHitV2{ItemID: v.ID, Kind: v.Kind, Content: v.Content, Scope: v.Scope, SourceID: v.Source.ID, Trust: v.Source.Trust, KeywordScore: keyword, FreshnessScore: .8, TrustScore: v.OutcomeQuality, CombinedScore: .25*keyword + .08 + .10*v.OutcomeQuality, EmbeddingVersion: v.EmbeddingVersion}
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].CombinedScore > hits[j].CombinedScore })
	if len(hits) > req.Limit {
		hits = hits[:req.Limit]
	}
	return hits
}
func (m *Manager) embed(ctx context.Context, text string) ([]float32, bool) {
	token := readSecret(m.cfg.EmbedTokenFile)
	payload, _ := json.Marshal(map[string]any{"texts": []string{text}, "normalize": true})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.EmbedURL, bytes.NewReader(payload))
	if err == nil {
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		response, e := m.http.Do(request)
		if e == nil {
			defer response.Body.Close()
			raw, _ := io.ReadAll(io.LimitReader(response.Body, 2<<20))
			var value struct {
				Embeddings [][]float32 `json:"embeddings"`
				Model      string      `json:"model"`
			}
			if response.StatusCode == 200 && json.Unmarshal(raw, &value) == nil && len(value.Embeddings) == 1 && len(value.Embeddings[0]) == 384 {
				return value.Embeddings[0], true
			}
		}
	}
	return make([]float32, 384), false
}
func (m *Manager) health(ctx context.Context) {
	_, ok := m.embed(ctx, "health probe")
	lab := map[string]any{"profile": "optional", "enabled": false, "minio": "stopped", "dagster": "stopped", "mlflow": "stopped"}
	for name, endpoint := range map[string]string{"minio": "http://minio:9000/minio/health/live", "dagster": "http://dagster:3000/server_info", "mlflow": "http://mlflow:5000/health"} {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		response, err := m.http.Do(request)
		if err == nil && response.StatusCode < 500 {
			lab[name], lab["enabled"] = "ready", true
		}
		if response != nil {
			_ = response.Body.Close()
		}
	}
	seed := false
	m.mu.Lock()
	if ok {
		m.snapshot.EmbeddingState = "ready"
		seed = activeItems(m.items) == 0
	} else {
		m.snapshot.EmbeddingState = "degraded"
	}
	m.snapshot.UpdatedAt = time.Now().UTC()
	m.snapshot.MemoryLab = lab
	m.mu.Unlock()
	if seed {
		m.ensureFixtures(ctx)
	}
}

func (m *Manager) ensureFixtures(ctx context.Context) {
	fixtures := []struct{ kind, content, source string }{
		{"runbook", "On communications loss, continue only validated unexpired cached authority. At tape empty, enter bounded safe hold and never invent new work.", "runbook-comms-loss-r1"},
		{"runbook", "Reject a GNSS jump when velocity and authenticated corroboration disagree. Keep the raw fix as evidence, exclude it from fusion, and expand uncertainty until corroboration returns.", "runbook-gnss-anomaly-r1"},
		{"runbook", "After reconnection, exchange high-water marks, apply tombstones first, expire stale segments, and bridge from the actual fused pose to future authorized work.", "runbook-stale-safe-rejoin-r1"},
		{"incident", "Vessel 4 exhausted pre-authorized tape after partition, rejected a 650 meter GNSS jump, entered safe hold, and rejoined through a future bridge without stale replay.", "fixture-vessel-4-incident"},
		{"mission_outcome", "Worker 2 termination caused cooperative partition reassignment and temporary lag; database idempotency prevented duplicate logical projection updates during recovery.", "fixture-worker-rebalance"},
	}
	for _, fixture := range fixtures {
		candidate := m.newCandidate(domain.MemoryScopeV1{Kind: "approved_global", ID: "global"}, fixture.kind, fixture.content, "approved_fixture", 1, false, fixture.source)
		m.commitCandidate(ctx, candidate, "system-fixture-seeder")
	}
}
func (m *Manager) persistJSON(ctx context.Context, query string, args ...any) {
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	if pool == nil {
		return
	}
	converted := make([]any, len(args))
	for i, v := range args {
		switch v.(type) {
		case domain.MemoryItemV1, domain.MemoryCandidateV1, domain.MemorySourceV1, domain.ContextAssemblyV1, domain.RetrievalReceiptV1, domain.MemoryReplayV1:
			raw, _ := json.Marshal(v)
			converted[i] = raw
		default:
			converted[i] = v
		}
	}
	if _, err := pool.Exec(ctx, query, converted...); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		m.logger.Warn("memory persistence failed", "error", err)
	}
}
func (m *Manager) publish(topic, key string, value any) {
	m.mu.RLock()
	producer := m.producer
	m.mu.RUnlock()
	if producer == nil {
		return
	}
	raw, _ := json.Marshal(value)
	producer.Produce(context.Background(), &kgo.Record{Topic: topic, Key: []byte(key), Value: raw}, func(_ *kgo.Record, err error) {
		if err != nil {
			m.logger.Warn("memory event publish failed", "topic", topic, "error", err)
		}
	})
}
func authorized(actor string, scope domain.MemoryScopeV1) bool {
	switch scope.Kind {
	case "approved_global":
		return true
	case "operator":
		return actor == scope.ID
	case "mission", "vessel", "group", "faction":
		return actor == "demo-operator" || strings.HasPrefix(scope.ID, actor+":")
	default:
		return false
	}
}
func scopeAllowed(scope domain.MemoryScopeV1, allowed []domain.MemoryScopeV1) bool {
	for _, v := range allowed {
		if v == scope {
			return true
		}
	}
	return false
}
func readSecret(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
func checksum(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func stable(prefix string, parts ...string) string {
	return prefix + "-" + strings.TrimPrefix(checksum(parts), "sha256:")[:20]
}
func vectorText(v []float32) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = fmt.Sprintf("%.8g", n)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
func clone[T any](v T) T {
	raw, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}
func reverseTurns(v []domain.ConversationTurnV1) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func activeItems(v map[string]domain.MemoryItemV1) int {
	n := 0
	for _, x := range v {
		if !x.Tombstoned {
			n++
		}
	}
	return n
}
func estimateAssembly(v domain.ContextAssemblyV1) int {
	n := 0
	for _, t := range v.RecentTurns {
		n += len(t.Content)
	}
	for _, set := range [][]domain.RetrievalHitV2{v.Semantic, v.Procedural, v.Episodes} {
		for _, h := range set {
			n += len(h.Content)
		}
	}
	return (n + 3) / 4
}
func trimAssembly(v domain.ContextAssemblyV1) domain.ContextAssemblyV1 {
	for estimateAssembly(v) > v.TokenBudget {
		if len(v.Semantic) > 0 {
			v.Semantic = v.Semantic[:len(v.Semantic)-1]
		} else if len(v.Procedural) > 0 {
			v.Procedural = v.Procedural[:len(v.Procedural)-1]
		} else if len(v.Episodes) > 0 {
			v.Episodes = v.Episodes[:len(v.Episodes)-1]
		} else if len(v.RecentTurns) > 1 {
			v.RecentTurns = v.RecentTurns[1:]
		} else {
			break
		}
	}
	v.EstimatedTokens = estimateAssembly(v)
	return v
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
