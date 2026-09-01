package core

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/geometry"
	"github.com/fourtytwo42/keelmesh/internal/planner"
	"github.com/fourtytwo42/keelmesh/internal/scenario"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string        { return e.Message }
func errCode(code, msg string) *Error { return &Error{Code: code, Message: msg} }

type CompileRequest struct {
	RequestID            string          `json:"request_id"`
	ExpectedStateVersion int64           `json:"expected_state_version"`
	Text                 string          `json:"text"`
	Area                 *domain.Polygon `json:"area"`
}
type PlansRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
	IntentID             string `json:"intent_id"`
}
type PreviewRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
}
type AuthorizeRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
	PlanHash             string `json:"plan_hash"`
	OperatorID           string `json:"operator_id"`
}
type StartRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedStateVersion int64  `json:"expected_state_version"`
	LeaseID              string `json:"lease_id"`
	PlanHash             string `json:"plan_hash"`
	IdempotencyKey       string `json:"idempotency_key"`
}

type missionRuntime struct {
	plan     domain.PlanCandidateV1
	distance map[string]float64
	total    map[string]float64
	lastTick time.Time
}

type Engine struct {
	mu           sync.RWMutex
	scenario     scenario.Scenario
	planner      planner.Planner
	stateVersion int64
	vessels      []domain.VesselV1
	intent       *domain.MissionIntentV1
	plans        map[string]domain.PlanCandidateV1
	previews     map[string]domain.PlanPreviewV1
	lease        *domain.MissionLeaseV1
	mission      domain.MissionStateV1
	runtime      *missionRuntime
	audit        []domain.AuditEventV1
	secret       []byte
	sequence     int64
	subs         map[chan domain.StreamMessageV1]struct{}
	idempotency  map[string]string
}

func New() *Engine {
	s := scenario.Golden()
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	return &Engine{scenario: s, planner: planner.Planner{Scenario: s}, stateVersion: 1, vessels: cloneVessels(s.Vessels), plans: map[string]domain.PlanCandidateV1{}, previews: map[string]domain.PlanPreviewV1{}, mission: domain.MissionStateV1{Phase: "idle"}, secret: secret, subs: map[chan domain.StreamMessageV1]struct{}{}, idempotency: map[string]string{}}
}

func (e *Engine) Run(ctx context.Context) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	broadcast := 0
	for {
		select {
		case now := <-tick.C:
			e.tick(now)
			broadcast++
			if broadcast%2 == 0 {
				e.broadcastSnapshot()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) Bootstrap() domain.BootstrapV1 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return domain.BootstrapV1{SchemaVersion: domain.SchemaVersion, Snapshot: e.snapshotLocked(), Boundary: e.scenario.Boundary, SuggestedArea: e.scenario.SuggestedArea, ExclusionZone: e.scenario.Exclusion, HoldingArea: e.scenario.Holding, Capabilities: []string{"draw_area", "typed_intent", "deterministic_plans", "plan_preview", "exact_hash_authorization", "simulated_execution"}, Audit: cloneAudit(e.audit)}
}
func (e *Engine) Snapshot() domain.FleetSnapshotV1 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.snapshotLocked()
}
func (e *Engine) snapshotLocked() domain.FleetSnapshotV1 {
	return domain.FleetSnapshotV1{SchemaVersion: domain.SchemaVersion, StateVersion: e.stateVersion, ScenarioID: e.scenario.ID, ScenarioName: e.scenario.Name, SimulationRate: e.scenario.SimulationRate, Vessels: cloneVessels(e.vessels), Mission: e.mission}
}

func (e *Engine) Compile(req CompileRequest) (domain.MissionIntentV1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.checkVersion(req.ExpectedStateVersion); err != nil {
		return domain.MissionIntentV1{}, err
	}
	if req.Area == nil {
		return domain.MissionIntentV1{}, errCode("AREA_REQUIRED", "Select or draw a search area first.")
	}
	area := *req.Area
	area.Coordinates[0] = geometry.NormalizeRing(area.Coordinates[0])
	if err := geometry.ValidatePolygon(area, e.scenario.Boundary.Geometry); err != nil {
		return domain.MissionIntentV1{}, errCode("INVALID_GEOMETRY", err.Error())
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = "Search this area with six vessels. Maintain 30% reserve and avoid the exclusion zone."
	}
	assets, minReserve, maxDuration := parseIntent(text)
	e.stateVersion++
	trace := req.RequestID
	if trace == "" {
		trace = newID("trace")
	}
	intent := domain.MissionIntentV1{SchemaVersion: domain.SchemaVersion, ID: fmt.Sprintf("intent-%d", e.stateVersion), TraceID: trace, SourceStateVersion: e.stateVersion, Objective: "search_area", Area: area, RequestedAssetCount: assets, Constraints: domain.IntentConstraintsV1{MinimumReserve: minReserve, MaximumDurationMinutes: maxDuration, AvoidZones: []string{e.scenario.Exclusion.ID}}, SourceText: text}
	intent.ContentHash = hashValue(intent)
	e.intent = &intent
	e.plans = map[string]domain.PlanCandidateV1{}
	e.previews = map[string]domain.PlanPreviewV1{}
	e.lease = nil
	e.runtime = nil
	e.mission = domain.MissionStateV1{Phase: "intent_ready"}
	e.appendAuditLocked(trace, "intent.compiled", "Mission intent compiled", map[string]any{"intent_id": intent.ID, "state_version": e.stateVersion})
	return intent, nil
}

func (e *Engine) GeneratePlans(req PlansRequest) ([]domain.PlanCandidateV1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.checkVersion(req.ExpectedStateVersion); err != nil {
		return nil, err
	}
	if e.intent == nil || e.intent.ID != req.IntentID {
		return nil, errCode("STALE_STATE", "The intent is no longer current.")
	}
	plans, err := e.planner.Generate(*e.intent)
	if err != nil {
		return nil, err
	}
	e.plans = map[string]domain.PlanCandidateV1{}
	for _, p := range plans {
		e.plans[p.ID] = p
	}
	e.mission.Phase = "plans_ready"
	e.appendAuditLocked(e.intent.TraceID, "plans.generated", "Two deterministic plans generated", map[string]any{"plan_count": len(plans)})
	return append([]domain.PlanCandidateV1(nil), plans...), nil
}

func (e *Engine) Preview(planID string, req PreviewRequest) (domain.PlanPreviewV1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.checkVersion(req.ExpectedStateVersion); err != nil {
		return domain.PlanPreviewV1{}, err
	}
	plan, ok := e.plans[planID]
	if !ok {
		return domain.PlanPreviewV1{}, errCode("STALE_STATE", "Plan not found or stale.")
	}
	if plan.SourceStateVersion != e.stateVersion {
		return domain.PlanPreviewV1{}, errCode("STALE_STATE", "Plan state is stale.")
	}
	preview := planner.Preview(plan)
	e.previews[planID] = preview
	e.mission.Phase = "previewing"
	e.mission.PlanID = plan.ID
	e.mission.PlanHash = plan.ContentHash
	e.appendAuditLocked(e.intent.TraceID, "plan.previewed", "Plan preview generated; nothing was sent", map[string]any{"plan_id": plan.ID, "plan_hash": plan.ContentHash, "request_id": req.RequestID})
	return preview, nil
}

func (e *Engine) Authorize(planID string, req AuthorizeRequest) (domain.MissionLeaseV1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.checkVersion(req.ExpectedStateVersion); err != nil {
		return domain.MissionLeaseV1{}, err
	}
	plan, ok := e.plans[planID]
	if !ok {
		return domain.MissionLeaseV1{}, errCode("STALE_STATE", "Plan not found or stale.")
	}
	recomputed, _ := planner.HashPlan(plan)
	if req.PlanHash != plan.ContentHash || recomputed != plan.ContentHash {
		e.appendAuditLocked(e.intent.TraceID, "authorization.rejected", "Plan hash did not match", nil)
		return domain.MissionLeaseV1{}, errCode("PLAN_HASH_MISMATCH", "Authorization must reference the exact previewed plan.")
	}
	if plan.Policy.Status == "prohibited" {
		return domain.MissionLeaseV1{}, errCode("POLICY_REJECTED", "Policy prohibits this plan.")
	}
	if req.OperatorID != "demo-operator" {
		return domain.MissionLeaseV1{}, errCode("POLICY_REJECTED", "Unknown operator identity.")
	}
	now := time.Now().UTC()
	assets := make([]string, 0, len(plan.Assignments))
	for _, a := range plan.Assignments {
		assets = append(assets, a.VesselID)
	}
	lease := domain.MissionLeaseV1{SchemaVersion: domain.SchemaVersion, ID: newID("lease"), MissionID: "mission-" + e.intent.ID, PlanID: plan.ID, PlanHash: plan.ContentHash, OperatorID: req.OperatorID, AssetIDs: assets, Area: e.intent.Area, MinReserve: e.intent.Constraints.MinimumReserve, IssuedAt: now, ExpiresAt: now.Add(15 * time.Minute)}
	lease.Signature = e.signLease(lease)
	e.lease = &lease
	e.mission = domain.MissionStateV1{ID: lease.MissionID, Phase: "authorized", PlanID: plan.ID, PlanHash: plan.ContentHash, LeaseID: lease.ID}
	e.appendAuditLocked(e.intent.TraceID, "mission.authorized", "Exact plan authorized", map[string]any{"lease_id": lease.ID, "plan_hash": plan.ContentHash, "operator_id": req.OperatorID})
	return lease, nil
}

func (e *Engine) Start(missionID string, req StartRequest) (domain.MissionStateV1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if req.IdempotencyKey == "" {
		return domain.MissionStateV1{}, errCode("IDEMPOTENCY_CONFLICT", "An idempotency key is required.")
	}
	fingerprint := req.LeaseID + "|" + req.PlanHash
	if prior, ok := e.idempotency[req.IdempotencyKey]; ok {
		if prior != fingerprint {
			return domain.MissionStateV1{}, errCode("IDEMPOTENCY_CONFLICT", "The idempotency key was already used for a different mission start.")
		}
		return e.mission, nil
	}
	if e.lease == nil {
		return domain.MissionStateV1{}, errCode("LEASE_REQUIRED", "Authorize the exact plan before starting.")
	}
	if !hmac.Equal([]byte(e.lease.Signature), []byte(e.signLease(*e.lease))) {
		return domain.MissionStateV1{}, errCode("LEASE_REQUIRED", "Lease signature is invalid.")
	}
	if req.ExpectedStateVersion != e.stateVersion {
		return domain.MissionStateV1{}, errCode("STALE_STATE", "Fleet state changed; refresh before starting.")
	}
	if e.lease.ID != req.LeaseID || e.lease.MissionID != missionID || e.lease.PlanHash != req.PlanHash {
		return domain.MissionStateV1{}, errCode("LEASE_REQUIRED", "Lease does not match this mission and plan.")
	}
	if time.Now().After(e.lease.ExpiresAt) {
		return domain.MissionStateV1{}, errCode("LEASE_EXPIRED", "The mission lease expired.")
	}
	plan := e.plans[e.lease.PlanID]
	totals := map[string]float64{}
	distances := map[string]float64{}
	for _, a := range plan.Assignments {
		totals[a.VesselID] = geometry.RouteDistanceKM(a.Route)
		distances[a.VesselID] = 0
	}
	now := time.Now().UTC()
	e.runtime = &missionRuntime{plan: plan, distance: distances, total: totals, lastTick: now}
	e.mission.Phase = "executing"
	e.mission.StartedAt = now.Format(time.RFC3339Nano)
	e.idempotency[req.IdempotencyKey] = fingerprint
	e.stateVersion++
	e.appendAuditLocked(e.intent.TraceID, "mission.started", "Authorized mission execution started", map[string]any{"mission_id": missionID, "simulation_rate": e.scenario.SimulationRate})
	return e.mission, nil
}

func (e *Engine) Audit(traceID string) []domain.AuditEventV1 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []domain.AuditEventV1{}
	for _, a := range e.audit {
		if traceID == "" || a.TraceID == traceID {
			out = append(out, a)
		}
	}
	return cloneAudit(out)
}
func (e *Engine) Subscribe() (chan domain.StreamMessageV1, func()) {
	ch := make(chan domain.StreamMessageV1, 8)
	e.mu.Lock()
	e.subs[ch] = struct{}{}
	e.mu.Unlock()
	return ch, func() {
		e.mu.Lock()
		if _, ok := e.subs[ch]; ok {
			delete(e.subs, ch)
			close(ch)
		}
		e.mu.Unlock()
	}
}

func (e *Engine) tick(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.runtime == nil || e.mission.Phase != "executing" {
		return
	}
	dt := now.Sub(e.runtime.lastTick).Seconds() * e.scenario.SimulationRate
	e.runtime.lastTick = now
	complete := 0
	progressSum := 0.0
	for i := range e.vessels {
		v := &e.vessels[i]
		var a *domain.AssignmentV1
		for j := range e.runtime.plan.Assignments {
			if e.runtime.plan.Assignments[j].VesselID == v.ID {
				a = &e.runtime.plan.Assignments[j]
				break
			}
		}
		if a == nil {
			continue
		}
		e.runtime.distance[v.ID] += a.SpeedMPS * dt / 1000
		total := e.runtime.total[v.ID]
		if e.runtime.distance[v.ID] >= total {
			e.runtime.distance[v.ID] = total
			complete++
		}
		pt, idx, f := geometry.InterpolateRoute(a.Route, e.runtime.distance[v.ID])
		old := v.Position
		v.Position = pt
		v.SpeedMPS = a.SpeedMPS
		v.RouteIndex = idx
		v.RouteProgress = f
		if pt != old {
			v.HeadingDeg = heading(old, pt)
		}
		progress := 1.0
		if total > 0 {
			progress = e.runtime.distance[v.ID] / total
		}
		progressSum += progress
	}
	e.mission.Progress = progressSum / float64(len(e.runtime.plan.Assignments))
	if complete == len(e.runtime.plan.Assignments) {
		e.mission.Phase = "completed"
		e.mission.Progress = 1
		e.appendAuditLocked(e.intent.TraceID, "mission.completed", "All six vessels completed their assigned routes", nil)
		e.runtime = nil
		e.stateVersion++
	}
}

func (e *Engine) broadcastSnapshot() {
	snap := e.Snapshot()
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sequence++
	msg := domain.StreamMessageV1{SchemaVersion: domain.SchemaVersion, Sequence: e.sequence, Kind: "fleet.snapshot", Snapshot: &snap}
	for ch := range e.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
func (e *Engine) appendAuditLocked(trace, kind, summary string, details map[string]any) {
	event := domain.AuditEventV1{SchemaVersion: domain.SchemaVersion, ID: newID("event"), TraceID: trace, Kind: kind, At: time.Now().UTC(), Summary: summary, Details: details}
	e.audit = append(e.audit, event)
	if len(e.audit) > 200 {
		e.audit = e.audit[len(e.audit)-200:]
	}
	e.sequence++
	msg := domain.StreamMessageV1{SchemaVersion: domain.SchemaVersion, Sequence: e.sequence, Kind: "audit.event.appended", Audit: &event}
	for ch := range e.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
func (e *Engine) checkVersion(v int64) error {
	if v != e.stateVersion {
		return errCode("STALE_STATE", fmt.Sprintf("Expected state version %d, current version is %d.", v, e.stateVersion))
	}
	return nil
}
func (e *Engine) signLease(l domain.MissionLeaseV1) string {
	l.Signature = ""
	b, _ := json.Marshal(l)
	m := hmac.New(sha256.New, e.secret)
	_, _ = m.Write(b)
	return "hmac-sha256:" + hex.EncodeToString(m.Sum(nil))
}

var percentRE = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:percent|%)`)
var minuteRE = regexp.MustCompile(`(?i)(\d{1,3})\s*(?:minute|min)`)
var assetRE = regexp.MustCompile(`(?i)(\d{1,2})\s+(?:closest\s+)?vessels?`)

func parseIntent(text string) (int, float64, float64) {
	assets := 6
	reserve := .30
	duration := 20.0
	if strings.Contains(strings.ToLower(text), "six") {
		assets = 6
	}
	if m := assetRE.FindStringSubmatch(text); len(m) > 1 {
		if v, _ := strconv.Atoi(m[1]); v > 0 {
			assets = v
		}
	}
	if m := percentRE.FindStringSubmatch(text); len(m) > 1 {
		if v, _ := strconv.Atoi(m[1]); v > 0 {
			reserve = float64(v) / 100
		}
	}
	if m := minuteRE.FindStringSubmatch(text); len(m) > 1 {
		if v, _ := strconv.Atoi(m[1]); v > 0 {
			duration = float64(v)
		}
	}
	return assets, reserve, duration
}
func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
func hashValue(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}
func cloneVessels(in []domain.VesselV1) []domain.VesselV1 {
	return append([]domain.VesselV1(nil), in...)
}
func cloneAudit(in []domain.AuditEventV1) []domain.AuditEventV1 {
	out := make([]domain.AuditEventV1, len(in))
	copy(out, in)
	return out
}
func heading(a, b domain.Point) float64 {
	angle := math.Atan2(b[0]-a[0], b[1]-a[1]) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	return angle
}
