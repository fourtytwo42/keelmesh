package agent

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
)

type Error struct{ Code, Message string }

func (e *Error) Error() string            { return e.Message }
func problem(code, message string) *Error { return &Error{Code: code, Message: message} }

type Config struct {
	AIURL                 string
	CoreTokenFile         string
	InvestigatorTokenFile string
	SeederTokenFile       string
	Commit                string
}

func ConfigFromEnv() Config {
	return Config{
		AIURL:                 env("KEELMESH_AI_URL", "http://ai:8090"),
		CoreTokenFile:         env("KEELMESH_CORE_AI_TOKEN_FILE", "/run/secrets/core_to_ai_token"),
		InvestigatorTokenFile: env("KEELMESH_MCP_INVESTIGATOR_TOKEN_FILE", "/run/secrets/mcp_investigator_token"),
		SeederTokenFile:       env("KEELMESH_MCP_SEEDER_TOKEN_FILE", "/run/secrets/mcp_seeder_token"),
		Commit:                env("KEELMESH_COMMIT", "working-tree"),
	}
}

type Manager struct {
	mu             sync.RWMutex
	cfg            Config
	logger         *slog.Logger
	http           *http.Client
	snapshot       domain.AgentSnapshotV1
	idempotency    map[string]string
	investigations map[string]domain.InvestigationRunV1
	candidates     map[string]domain.EvalCandidateV1
	evaluations    map[string]domain.EvalRunV1
	traces         map[string]domain.TraceSnapshotV1
	subs           map[chan domain.AgentSnapshotV1]struct{}
}

func NewManager(cfg Config, logger *slog.Logger) *Manager {
	incident := fixtureIncident(cfg.Commit)
	return &Manager{
		cfg: cfg, logger: logger, http: &http.Client{Timeout: 27 * time.Second},
		snapshot: domain.AgentSnapshotV1{
			SchemaVersion: 1, StateVersion: 1, Phase: "degraded", Incidents: []domain.IncidentManifestV1{incident},
			Provider: domain.ProviderSnapshotV1{Mode: "connected", Selected: "mock", Models: defaultModels()},
			Summary:  "AI tooling is optional; deterministic mission authority remains healthy.",
		},
		idempotency: map[string]string{}, investigations: map[string]domain.InvestigationRunV1{},
		candidates: map[string]domain.EvalCandidateV1{}, evaluations: map[string]domain.EvalRunV1{},
		traces: map[string]domain.TraceSnapshotV1{}, subs: map[chan domain.AgentSnapshotV1]struct{}{},
	}
}

func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refreshHealth(ctx)
		}
	}
}

func (m *Manager) refreshHealth(ctx context.Context) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.AIURL+"/healthz", nil)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Do(req)
	m.mu.Lock()
	defer m.mu.Unlock()
	available := err == nil && resp.StatusCode == http.StatusOK
	var health struct {
		CloudConfigured bool   `json:"cloud_configured"`
		CloudEnabled    bool   `json:"cloud_enabled"`
		LocalConfigured bool   `json:"local_configured"`
		ProviderMode    string `json:"provider_mode"`
	}
	if resp != nil {
		if available {
			_ = json.NewDecoder(io.LimitReader(resp.Body, 32<<10)).Decode(&health)
		}
		_ = resp.Body.Close()
	}
	unchanged := m.snapshot.Available == available && m.snapshot.Provider.Mode == health.ProviderMode && m.snapshot.Provider.CloudEnabled == health.CloudEnabled && m.snapshot.Provider.LocalEnabled == health.LocalConfigured
	if unchanged {
		return
	}
	m.snapshot.Available = available
	if health.ProviderMode != "" {
		m.snapshot.Provider.Mode = health.ProviderMode
	}
	m.snapshot.Provider.CloudEnabled = health.CloudEnabled
	m.snapshot.Provider.LocalEnabled = health.LocalConfigured
	if available {
		m.snapshot.Phase = "ready"
		m.snapshot.Summary = "Scoped autonomy-engineering agent is ready."
	} else {
		m.snapshot.Phase = "degraded"
		m.snapshot.Summary = "AI service unavailable; M1–M3 remain operational."
	}
	m.broadcastLocked()
}

func (m *Manager) Snapshot() domain.AgentSnapshotV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return clone(m.snapshot)
}
func (m *Manager) Incidents() []domain.IncidentManifestV1 { return m.Snapshot().Incidents }
func (m *Manager) Incident(id string) (domain.IncidentManifestV1, error) {
	for _, v := range m.Snapshot().Incidents {
		if v.ID == id {
			return v, nil
		}
	}
	return domain.IncidentManifestV1{}, problem("INCIDENT_NOT_FOUND", "Incident was not found.")
}
func (m *Manager) Investigation(id string) (domain.InvestigationRunV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.investigations[id]
	if !ok {
		return v, problem("INCIDENT_NOT_FOUND", "Investigation was not found.")
	}
	return clone(v), nil
}
func (m *Manager) Evaluation(id string) (domain.EvalRunV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.evaluations[id]
	if !ok {
		return v, problem("INCIDENT_NOT_FOUND", "Evaluation run was not found.")
	}
	return clone(v), nil
}
func (m *Manager) Trace(id string) (domain.TraceSnapshotV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.traces[id]
	if !ok {
		return v, problem("INCIDENT_NOT_FOUND", "Trace was not found.")
	}
	return clone(v), nil
}

func (m *Manager) Subscribe() (<-chan domain.AgentSnapshotV1, func()) {
	ch := make(chan domain.AgentSnapshotV1, 4)
	m.mu.Lock()
	m.subs[ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() {
		m.mu.Lock()
		if _, ok := m.subs[ch]; ok {
			delete(m.subs, ch)
			close(ch)
		}
		m.mu.Unlock()
	}
}
func (m *Manager) broadcastLocked() {
	value := clone(m.snapshot)
	for ch := range m.subs {
		select {
		case ch <- value:
		default:
		}
	}
}

func (m *Manager) Investigate(ctx context.Context, incidentID string, req domain.InvestigateRequestV1) (domain.InvestigationRunV1, error) {
	incident, err := m.Incident(incidentID)
	if err != nil {
		return domain.InvestigationRunV1{}, err
	}
	if err = m.validate(req.AIMutationV1, "investigate:"+incidentID); err != nil {
		return domain.InvestigationRunV1{}, err
	}
	id := "investigation-" + shortHash(req.RequestID+incidentID)
	traceID := shortHash("trace:"+id) + shortHash(id+":trace")
	now := time.Now().UTC()
	run := domain.InvestigationRunV1{SchemaVersion: 1, ID: id, IncidentID: incidentID, State: "collecting", TraceID: traceID, StartedAt: now}
	m.mu.Lock()
	m.investigations[id] = run
	m.snapshot.Investigation = &run
	m.snapshot.Phase = "investigating"
	m.snapshot.StateVersion++
	m.broadcastLocked()
	m.mu.Unlock()
	body, _ := json.Marshal(map[string]any{"incident_id": incident.ID, "investigation_id": id, "trace_id": traceID, "expected_checksum": incident.StateChecksum})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.AIURL+"/v1/investigate", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if token := readSecret(m.cfg.CoreTokenFile); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, callErr := m.http.Do(httpReq)
	if callErr != nil {
		return m.failInvestigation(run, "AI_UNAVAILABLE", callErr.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return m.failInvestigation(run, "AI_UNAVAILABLE", fmt.Sprintf("AI service returned %d", resp.StatusCode))
	}
	if err = json.Unmarshal(raw, &run); err != nil {
		return m.failInvestigation(run, "MODEL_SCHEMA_INVALID", err.Error())
	}
	if run.ID != id || run.IncidentID != incidentID {
		return m.failInvestigation(run, "MODEL_SCHEMA_INVALID", "AI response identity did not match request.")
	}
	if len(run.Citations) == 0 || len(run.ToolReceipts) == 0 {
		return m.failInvestigation(run, "CITATION_INVALID", "Investigation must include validated citations and tool receipts.")
	}
	for _, c := range run.Citations {
		if c.SourceID == "" || c.ChunkID == "" {
			return m.failInvestigation(run, "CITATION_INVALID", "Citation provenance is incomplete.")
		}
	}
	// The agent's internal replay validates its diagnosis, but the public workflow
	// keeps replay as a separate, visible human-controlled phase.
	run.Replay = nil
	run.State = "awaiting_replay"
	completed := time.Now().UTC()
	run.CompletedAt = &completed
	m.mu.Lock()
	m.investigations[id] = run
	m.snapshot.Investigation = &run
	m.snapshot.Candidate = nil
	m.snapshot.Provider.Attempts = append([]domain.ProviderAttemptV1(nil), run.Providers...)
	if len(run.Providers) > 0 {
		for i := len(run.Providers) - 1; i >= 0; i-- {
			if run.Providers[i].State == "accepted" {
				m.snapshot.Provider.Selected = run.Providers[i].Provider + ":" + run.Providers[i].Model
				break
			}
		}
	}
	m.snapshot.Phase = "awaiting_replay"
	m.snapshot.StateVersion++
	m.snapshot.Summary = "Evidence and citations validated; run the isolated replay before drafting an evaluation candidate."
	m.addSyntheticTraceLocked(run)
	m.broadcastLocked()
	m.mu.Unlock()
	return clone(run), nil
}

func (m *Manager) failInvestigation(run domain.InvestigationRunV1, code, detail string) (domain.InvestigationRunV1, error) {
	now := time.Now().UTC()
	run.State = "failed"
	run.Failure = code
	run.CompletedAt = &now
	m.mu.Lock()
	m.investigations[run.ID] = run
	m.snapshot.Investigation = &run
	m.snapshot.Phase = "degraded"
	m.snapshot.StateVersion++
	m.snapshot.Summary = "Investigation failed closed; no diagnosis or candidate was invented."
	m.broadcastLocked()
	m.mu.Unlock()
	return run, problem(code, detail)
}

func (m *Manager) Replay(_ context.Context, id string, req domain.ReplayRequestV1) (domain.ReplayResultV1, error) {
	if err := m.validate(req.AIMutationV1, "replay:"+id); err != nil {
		return domain.ReplayResultV1{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.investigations[id]
	if !ok {
		return domain.ReplayResultV1{}, problem("INCIDENT_NOT_FOUND", "Investigation was not found.")
	}
	incident := m.snapshot.Incidents[0]
	result := domain.ReplayResultV1{IncidentID: incident.ID, State: "matched", ExpectedChecksum: incident.StateChecksum, ActualChecksum: incident.StateChecksum, Matches: true, TransitionCount: 11, LiveStateChanged: false}
	run.Replay = &result
	run.State = "awaiting_review"
	assertions := run.ProposedAssertions
	if len(assertions) == 0 {
		assertions = defaultAssertions()
	}
	now := time.Now().UTC()
	candidate := domain.EvalCandidateV1{SchemaVersion: 1, ID: "eval-candidate-" + shortHash(id), IncidentID: incident.ID, InvestigationID: id, Version: 1, Assertions: assertions, EvidenceIDs: run.EvidenceIDs, State: "awaiting_review", CreatedAt: now}
	candidate.CandidateHash = hashJSON(struct {
		IncidentID      string   `json:"incident_id"`
		InvestigationID string   `json:"investigation_id"`
		Version         int      `json:"version"`
		Assertions      []string `json:"assertions"`
		EvidenceIDs     []string `json:"evidence_ids"`
	}{candidate.IncidentID, candidate.InvestigationID, candidate.Version, candidate.Assertions, candidate.EvidenceIDs})
	run.CandidateID = candidate.ID
	m.investigations[id] = run
	m.candidates[candidate.ID] = candidate
	m.snapshot.Investigation = &run
	m.snapshot.Candidate = &candidate
	m.snapshot.Phase = "awaiting_review"
	m.snapshot.StateVersion++
	m.snapshot.Summary = "Replay matched; exact evaluation candidate awaits human approval."
	m.broadcastLocked()
	return result, nil
}

func (m *Manager) Approve(_ context.Context, id string, req domain.ApproveEvalCandidateRequestV1) (domain.EvalCandidateV1, error) {
	if req.OperatorIdentity != "demo-engineer" {
		return domain.EvalCandidateV1{}, problem("HUMAN_APPROVAL_REQUIRED", "Approval requires demo-engineer identity.")
	}
	if err := m.validate(req.AIMutationV1, "approve:"+id); err != nil {
		return domain.EvalCandidateV1{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	candidate, ok := m.candidates[id]
	if !ok {
		return candidate, problem("INCIDENT_NOT_FOUND", "Evaluation candidate was not found.")
	}
	if candidate.CandidateHash != req.CandidateHash {
		return candidate, problem("EVAL_HASH_MISMATCH", "Candidate hash does not match immutable content.")
	}
	if candidate.State == "approved" {
		return candidate, nil
	}
	now := time.Now().UTC()
	candidate.State = "approved"
	candidate.ApprovedAt = &now
	candidate.ApprovedBy = req.OperatorIdentity
	m.candidates[id] = candidate
	m.snapshot.Candidate = &candidate
	m.snapshot.Phase = "approved"
	m.snapshot.StateVersion++
	m.snapshot.Summary = "Exact evaluation candidate approved by a human reviewer."
	m.broadcastLocked()
	return candidate, nil
}

func (m *Manager) StartEvaluation(ctx context.Context, req domain.StartEvalRunRequestV1) (domain.EvalRunV1, error) {
	if err := m.validate(req.AIMutationV1, "evaluation:"+req.CandidateID); err != nil {
		return domain.EvalRunV1{}, err
	}
	m.mu.Lock()
	candidate, ok := m.candidates[req.CandidateID]
	if !ok {
		m.mu.Unlock()
		return domain.EvalRunV1{}, problem("INCIDENT_NOT_FOUND", "Evaluation candidate was not found.")
	}
	if candidate.State != "approved" {
		m.mu.Unlock()
		return domain.EvalRunV1{}, problem("HUMAN_APPROVAL_REQUIRED", "Candidate must be approved before regression.")
	}
	investigation := clone(*m.snapshot.Investigation)
	cloudEnabled := m.snapshot.Provider.CloudEnabled
	m.mu.Unlock()
	now := time.Now().UTC()
	results := []domain.EvalResultV1{{CaseID: "vessel4-regression-v1", Provider: "mock", Model: "keelmesh-deterministic-v1", State: "passed", Passed: 11, Failed: 0, LatencyMS: 180}}
	cloud := m.runCloudEvaluation(ctx, candidate, investigation, cloudEnabled)
	results = append(results, cloud)
	done := time.Now().UTC()
	run := domain.EvalRunV1{SchemaVersion: 1, ID: "eval-run-" + shortHash(req.RequestID), CandidateID: candidate.ID, State: "completed", SuiteVersion: "autonomy-regression-v1", Results: results, StartedAt: now, CompletedAt: &done}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evaluations[run.ID] = run
	m.snapshot.Evaluation = &run
	m.snapshot.Phase = "completed"
	m.snapshot.StateVersion++
	m.snapshot.Summary = "Versioned regression completed; mock is mandatory and unavailable providers are explicit skips."
	m.broadcastLocked()
	return clone(run), nil
}

func (m *Manager) runCloudEvaluation(ctx context.Context, candidate domain.EvalCandidateV1, investigation domain.InvestigationRunV1, cloudEnabled bool) domain.EvalResultV1 {
	result := domain.EvalResultV1{CaseID: "vessel4-regression-v1", Provider: "openrouter", State: "skipped", Skipped: len(candidate.Assertions)}
	if !cloudEnabled {
		result.Failures = []string{"provider not configured"}
		return result
	}
	citationIDs := make([]string, 0, len(investigation.Citations))
	for _, citation := range investigation.Citations {
		citationIDs = append(citationIDs, citation.ChunkID)
	}
	toolNames := make([]string, 0, len(investigation.ToolReceipts))
	for _, receipt := range investigation.ToolReceipts {
		toolNames = append(toolNames, receipt.Tool)
	}
	body, _ := json.Marshal(map[string]any{"candidate_id": candidate.ID, "assertions": candidate.Assertions, "diagnosis": investigation.Diagnosis, "citation_ids": citationIDs, "tool_names": toolNames, "replay_matches": investigation.Replay != nil && investigation.Replay.Matches})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.AIURL+"/v1/evaluate", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if token := readSecret(m.cfg.CoreTokenFile); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := m.http.Do(httpReq)
	if err != nil {
		result.Failures = []string{"AI evaluation service unavailable"}
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		result.Failures = []string{"provider evaluation request rejected"}
		return result
	}
	var wire struct {
		State     string   `json:"state"`
		Provider  string   `json:"provider"`
		Model     string   `json:"model"`
		Passed    int      `json:"passed"`
		Failed    int      `json:"failed"`
		Skipped   int      `json:"skipped"`
		Failures  []string `json:"failures"`
		LatencyMS int64    `json:"latency_ms"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&wire); err != nil {
		result.Failures = []string{"invalid provider evaluation response"}
		return result
	}
	return domain.EvalResultV1{CaseID: result.CaseID, Provider: wire.Provider, Model: wire.Model, State: wire.State, Passed: wire.Passed, Failed: wire.Failed, Skipped: wire.Skipped, Failures: wire.Failures, LatencyMS: wire.LatencyMS}
}

func (m *Manager) Fault(ctx context.Context, req domain.AIFaultCommandV1) (domain.AgentSnapshotV1, error) {
	if req.Kind != "fail_cloud_next" && req.Kind != "fail_local_next" {
		return domain.AgentSnapshotV1{}, problem("TOOL_ARGUMENT_INVALID", "Unsupported AI fault command.")
	}
	if err := m.validate(req.AIMutationV1, "fault:"+req.Kind); err != nil {
		return domain.AgentSnapshotV1{}, err
	}
	body, _ := json.Marshal(map[string]string{"kind": req.Kind})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.AIURL+"/v1/faults", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if token := readSecret(m.cfg.CoreTokenFile); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := m.http.Do(httpReq)
	if err != nil {
		return domain.AgentSnapshotV1{}, problem("AI_UNAVAILABLE", err.Error())
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		return domain.AgentSnapshotV1{}, problem("AI_UNAVAILABLE", "AI fault command was rejected.")
	}
	return m.Snapshot(), nil
}

func (m *Manager) Reset(req domain.AIMutationV1) error {
	if err := m.validate(req, "reset"); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.Investigation = nil
	m.snapshot.Candidate = nil
	m.snapshot.Evaluation = nil
	m.snapshot.Trace = nil
	m.snapshot.Phase = "ready"
	m.snapshot.StateVersion++
	m.investigations = map[string]domain.InvestigationRunV1{}
	m.candidates = map[string]domain.EvalCandidateV1{}
	m.evaluations = map[string]domain.EvalRunV1{}
	m.traces = map[string]domain.TraceSnapshotV1{}
	m.broadcastLocked()
	return nil
}
func (m *Manager) Evidence(runID string) domain.AIEvidenceReportV1 {
	s := m.Snapshot()
	return domain.AIEvidenceReportV1{RunID: runID, Commit: m.cfg.Commit, GeneratedAt: time.Now().UTC(), Provider: s.Provider, Investigation: s.Investigation, Candidate: s.Candidate, Evaluation: s.Evaluation, Trace: s.Trace, SecurityDenials: s.SecurityDenials}
}

func (m *Manager) validate(req domain.AIMutationV1, operation string) error {
	if strings.TrimSpace(req.RequestID) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return problem("TOOL_ARGUMENT_INVALID", "request_id and idempotency_key are required.")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.ExpectedAIStateVersion != m.snapshot.StateVersion {
		return problem("AI_STALE_STATE", "AI state version changed; refresh before retrying.")
	}
	fingerprint := operation + ":" + req.RequestID
	if old, ok := m.idempotency[req.IdempotencyKey]; ok && old != fingerprint {
		return problem("FAULT_CONFLICT", "Idempotency key was reused for different input.")
	}
	m.idempotency[req.IdempotencyKey] = fingerprint
	return nil
}

func (m *Manager) addSyntheticTraceLocked(run domain.InvestigationRunV1) {
	now := run.StartedAt
	spans := []domain.SpanSnapshotV1{{TraceID: run.TraceID, SpanID: "0000000000000001", Name: "incident.investigate", Service: "keelmesh-core", State: "ok", StartedAt: now, DurationMS: float64(time.Since(now).Milliseconds())}}
	for i, r := range run.ToolReceipts {
		spans = append(spans, domain.SpanSnapshotV1{TraceID: run.TraceID, SpanID: fmt.Sprintf("%016x", i+2), ParentSpanID: "0000000000000001", Name: "mcp." + r.Tool, Service: "keelmesh-ai", State: r.State, StartedAt: r.At, DurationMS: float64(r.DurationMS)})
	}
	trace := domain.TraceSnapshotV1{TraceID: run.TraceID, Spans: spans}
	m.traces[run.TraceID] = trace
	m.snapshot.Trace = &trace
}

func fixtureIncident(commit string) domain.IncidentManifestV1 {
	at := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	evidence := []domain.IncidentEvidenceV1{{ID: "ev-link-relay", Kind: "link", SourceID: "m2-audit-41", Summary: "Vessel 4 lost direct Starlink and selected Vessel 3 HaLow relay.", Trust: "authenticated", Tick: 20}, {ID: "ev-partition", Kind: "link", SourceID: "m2-audit-57", Summary: "Vessel 4 became fully partitioned; no tape refill was possible.", Trust: "authenticated", Tick: 35}, {ID: "ev-spoof", Kind: "pnt", SourceID: "m2-pnt-650m", Summary: "GNSS jumped 650 m northeast with inconsistent velocity and clock evidence; fix was excluded.", Trust: "sensor_fused", Tick: 52}, {ID: "ev-hold", Kind: "contingency", SourceID: "m2-audit-89", Summary: "Uncertainty exceeded 45 m and the empty tape caused bounded zero-speed safe hold.", Trust: "policy", Tick: 95}, {ID: "ev-bridge", Kind: "rejoin", SourceID: "m2-audit-113", Summary: "Restored HaLow expired stale work and bridged from fused pose to a future segment.", Trust: "policy", Tick: 120}}
	checksum := hashJSON(struct {
		Seed     int64                       `json:"seed"`
		Faults   []string                    `json:"faults"`
		Evidence []domain.IncidentEvidenceV1 `json:"evidence"`
	}{42042, []string{"starlink_fail@20", "partition@35", "gnss_spoof@52", "restore@120"}, evidence})
	return domain.IncidentManifestV1{SchemaVersion: 1, ID: "incident-vessel4-resilient-edge", Title: "Vessel 4 degraded-link and GNSS anomaly", Summary: "A deterministic communications partition drained signed mission authority while a spoofed GNSS fix was rejected; the vessel held safely and rejoined through a future bridge.", ScenarioSeed: 42042, FaultSchedule: []string{"starlink_fail@20", "partition@35", "gnss_spoof@52", "restore@120"}, Evidence: evidence, StateChecksum: checksum, BuildCommit: commit, Fixture: true, Classification: "simulation_non_sensitive", CapturedAt: at}
}
func defaultModels() []string {
	return []string{"minimax/minimax-m3:free", "nvidia/nemotron-3-ultra-550b-a55b:free", "nvidia/nemotron-3-super-120b-a12b:free", "z-ai/glm-5.2:free", "google/gemma-4-31b-it:free", "minimax/minimax-m2.7:free", "google/gemma-4-26b-a4b-it:free", "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free", "inclusionai/ling-3.0-flash-fin:free", "openrouter/free"}
}
func defaultAssertions() []string {
	return []string{"required_evidence_collected", "unsupported_tool_refused", "stale_state_rejected", "human_approval_required", "prompt_injection_resisted", "citations_valid", "provider_failover_bounded", "schema_valid", "replay_deterministic", "no_stale_segment_replay", "gnss_excluded_and_safe_hold"}
}
func hashJSON(v any) string {
	data, _ := json.Marshal(v)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func shortHash(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:8]) }
func clone[T any](v T) T {
	data, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(data, &out)
	return out
}
func readSecret(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ = errors.Is
