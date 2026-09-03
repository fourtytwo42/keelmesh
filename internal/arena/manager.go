package arena

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Code + ": " + e.Message }

type Manager struct {
	mu                      sync.Mutex
	version, tick, eventSeq int64
	phase                   string
	nodes                   map[string]domain.ArenaNodeV1
	coords                  map[string]domain.CoordinatorV1
	contacts                map[string][]domain.ContactTrackV1
	actions                 []domain.WorkspaceActionV1
	events                  []domain.ArenaEventV1
	plans                   map[string]domain.EngagementPlanV1
	leases                  map[string]domain.EngagementLeaseV1
	effects                 []domain.ArenaEffectV1
	sessions                map[string]domain.AgentSessionV1
	idempotency             map[string]string
	secret                  []byte
	localFaction            string
	localNodeID             string
}

func New() *Manager {
	m := &Manager{secret: []byte("keelmesh-m7-logical-referee"), nodes: map[string]domain.ArenaNodeV1{}, coords: map[string]domain.CoordinatorV1{}, contacts: map[string][]domain.ContactTrackV1{}, plans: map[string]domain.EngagementPlanV1{}, leases: map[string]domain.EngagementLeaseV1{}, sessions: map[string]domain.AgentSessionV1{}, idempotency: map[string]string{}}
	m.seed()
	return m
}

func NewFromEnv() *Manager {
	m := New()
	faction := strings.ToUpper(strings.TrimSpace(os.Getenv("KEELMESH_NODE_FACTION")))
	if faction == "A" || faction == "B" {
		m.localFaction = faction
	}
	m.localNodeID = strings.TrimSpace(os.Getenv("KEELMESH_NODE_ID"))
	return m
}

func (m *Manager) seed() {
	m.version = 1
	m.tick = 0
	m.eventSeq = 0
	m.phase = "lobby"
	m.actions = nil
	m.events = nil
	m.plans = map[string]domain.EngagementPlanV1{}
	m.leases = map[string]domain.EngagementLeaseV1{}
	m.sessions = map[string]domain.AgentSessionV1{}
	classes := []string{"kestrel", "kestrel", "kestrel", "mariner", "mariner", "atlas"}
	vmids := [][]int{{220, 221, 222, 223, 224, 225}, {229, 231, 232, 233, 234, 236}}
	hosts := map[int]string{220: "fourtyfour", 221: "fourtyfour", 222: "fourtyfour", 223: "fourtyfour", 224: "fourtyfour", 225: "mini42", 229: "mini42", 231: "mini42", 232: "mini42", 233: "mini43", 234: "mini43", 236: "mini43"}
	for fi, f := range []string{"A", "B"} {
		for i := 0; i < 6; i++ {
			n := i + 1
			id := fmt.Sprintf("node-%s-%02d", strings.ToLower(f), n)
			cls := classes[i]
			cap, hull, solar := 18.0, 60, 1.2
			if cls == "mariner" {
				cap, hull, solar = 40, 100, 2
			}
			if cls == "atlas" {
				cap, hull, solar = 90, 180, 4
			}
			eq := []string{"eo_sensor"}
			if i == 5 {
				eq = []string{"enhanced_radar", "mesh_relay", "battery_upgrade", "disruption_pulse"}
			} else if i >= 3 {
				eq = []string{"enhanced_radar", "light_kinetic", "mine_pack"}
			} else if i == 0 {
				eq = []string{"enhanced_radar", "communications_jammer"}
			}
			vmid := vmids[fi][i]
			m.nodes[id] = domain.ArenaNodeV1{ID: id, Faction: f, VesselID: fmt.Sprintf("arena-%s-%02d", strings.ToLower(f), n), PlannedVMID: vmid, PlannedManagementIP: fmt.Sprintf("192.168.50.%d", vmid), Host: hosts[vmid], Role: map[bool]string{true: "coordinator", false: "follower"}[i == 0], Status: "provisioned", RadioState: "connected", ManagementConnected: true, InferenceConnected: true, Provider: "openrouter", Position: domain.GeoPointV2{-71.48 + float64(i)*.008 + float64(fi)*.26, 41.34 + float64(i%3)*.012}, HeadingDeg: map[bool]float64{true: 270, false: 90}[fi == 1], BatteryKWh: cap * .82, BatteryCapacityKWh: cap, SolarKW: solar, Hull: hull, HullMaximum: hull, Class: cls, Equipment: eq, TapeDepthSeconds: 60}
		}
	}
	m.coords["A"] = domain.CoordinatorV1{Faction: "A", NodeID: "node-a-01", Epoch: 1, Votes: 6, QuorumRequired: 4, State: "stable"}
	m.coords["B"] = domain.CoordinatorV1{Faction: "B", NodeID: "node-b-01", Epoch: 1, Votes: 6, QuorumRequired: 4, State: "stable"}
	m.contacts["A"] = []domain.ContactTrackV1{{ID: "track-a-001", Faction: "A", Classification: "unknown surface contact", Hostility: "unknown", Position: domain.GeoPointV2{-71.25, 41.36}, HeadingDeg: 270, SpeedMPS: 1.7, Confidence: .67, UncertaintyM: 180, Source: "node-a-06/radar", State: "direct"}}
	m.contacts["B"] = []domain.ContactTrackV1{{ID: "track-b-001", Faction: "B", Classification: "unknown surface contact", Hostility: "unknown", Position: domain.GeoPointV2{-71.46, 41.36}, HeadingDeg: 90, SpeedMPS: 1.5, Confidence: .64, UncertaintyM: 210, Source: "node-b-06/radar", State: "direct"}}
	m.emit("", "arena.reset", "Two knowledge-isolated factions and twelve symmetric logical nodes initialized.", nil)
}

func (m *Manager) Snapshot(faction string) domain.ArenaSnapshotV1 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshot(faction)
}
func (m *Manager) snapshot(faction string) domain.ArenaSnapshotV1 {
	if m.localFaction != "" {
		faction = m.localFaction
	}
	if faction != "B" {
		faction = "A"
	}
	nodes := make([]domain.ArenaNodeV1, 0, 6)
	friendly := []string{}
	for _, n := range m.nodes {
		if n.Faction == faction {
			nodes = append(nodes, n)
			friendly = append(friendly, n.ID)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Strings(friendly)
	coords := []domain.CoordinatorV1{m.coords[faction]}
	knowledge := domain.KnowledgeSnapshotV1{Faction: faction, FriendlyIDs: friendly, Contacts: append([]domain.ContactTrackV1{}, m.contacts[faction]...)}
	knowledge.Checksum = hash(knowledge)
	ev := m.events
	if len(ev) > 30 {
		ev = ev[len(ev)-30:]
	}
	return domain.ArenaSnapshotV1{SchemaVersion: 1, StateVersion: m.version, MatchID: "arena-match-001", Mode: "distributed", Phase: m.phase, SimulationRate: 20, MissionTick: m.tick, SimulatedTime: fmt.Sprintf("%02d:%02d", (6+int(m.tick/1800))%24, (int(m.tick/30))%60), ViewerFaction: faction, Credits: map[string]int{faction: 12000}, Nodes: nodes, Coordinators: coords, Knowledge: knowledge, WorkspaceActions: append([]domain.WorkspaceActionV1{}, m.actions...), Events: append([]domain.ArenaEventV1{}, ev...), ManagementPlane: "protected · connected", InferencePlane: "protected · direct HTTPS", RadioPlane: m.radioSummary(faction), ProvisioningState: "12_nodes_provisioned", ProvisioningBlockers: []string{}}
}

func (m *Manager) InfrastructureSnapshot() domain.ArenaSnapshotV1 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.localFaction != "" {
		return m.snapshot(m.localFaction)
	}
	s := m.snapshot("A")
	s.ViewerFaction = "referee"
	s.Nodes = s.Nodes[:0]
	for _, n := range m.nodes {
		s.Nodes = append(s.Nodes, n)
	}
	sort.Slice(s.Nodes, func(i, j int) bool { return s.Nodes[i].PlannedVMID < s.Nodes[j].PlannedVMID })
	s.Coordinators = []domain.CoordinatorV1{m.coords["A"], m.coords["B"]}
	s.Knowledge = domain.KnowledgeSnapshotV1{}
	return s
}

func (m *Manager) LocalIdentity() (string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.localFaction, m.localNodeID
}

func (m *Manager) Reset(req domain.ArenaMutationV1) (domain.ArenaSnapshotV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req); err != nil {
		return domain.ArenaSnapshotV1{}, err
	}
	m.seed()
	m.version++
	return m.snapshot(req.ActorID), nil
}
func (m *Manager) Start(req domain.ArenaMutationV1) (domain.ArenaSnapshotV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req); err != nil {
		return domain.ArenaSnapshotV1{}, err
	}
	if m.phase != "lobby" {
		return domain.ArenaSnapshotV1{}, &Error{"INVALID_MATCH_PHASE", "Match can start only from the lobby."}
	}
	m.phase = "running"
	m.tick = 1
	m.version++
	m.emit("", "match.started", "Fleet Arena started at 20× simulation time.", nil)
	return m.snapshot(req.ActorID), nil
}

type FaultRequest struct {
	domain.ArenaMutationV1
	Faction string `json:"faction"`
	Kind    string `json:"kind"`
	NodeID  string `json:"node_id"`
}

func (m *Manager) Fault(req FaultRequest) (domain.ArenaSnapshotV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.ArenaSnapshotV1{}, err
	}
	if strings.Contains(req.Kind, "management") || strings.Contains(req.Kind, "inference") {
		return domain.ArenaSnapshotV1{}, &Error{"PROTECTED_PLANE", "Management and inference planes cannot be targeted by radio faults."}
	}
	f := strings.ToUpper(req.Faction)
	if f != "A" && f != "B" {
		return domain.ArenaSnapshotV1{}, &Error{"INVALID_FACTION", "Faction must be A or B."}
	}
	switch req.Kind {
	case "fail_starlink":
		for id, n := range m.nodes {
			if n.Faction == f {
				n.RadioState = "halow-only"
				m.nodes[id] = n
			}
		}
	case "partition_coordinator":
		c := m.coords[f]
		old := m.nodes[c.NodeID]
		old.RadioState = "partitioned"
		old.Role = "isolated"
		m.nodes[old.ID] = old
		ids := m.eligible(f, old.ID)
		if len(ids) < 4 {
			return domain.ArenaSnapshotV1{}, &Error{"QUORUM_NOT_MET", "Four friendly votes are required."}
		}
		next := m.nodes[ids[0]]
		next.Role = "coordinator"
		m.nodes[next.ID] = next
		c.NodeID = next.ID
		c.Epoch++
		c.Votes = len(ids)
		c.FailoverCount++
		c.RecoveryMS = 860
		c.State = "stable"
		m.coords[f] = c
		m.emit(f, "coordinator.changed", fmt.Sprintf("%s elected at epoch %d; management and inference remained connected.", next.ID, c.Epoch), map[string]any{"previous": old.ID, "current": next.ID})
	case "restore_radio":
		for id, n := range m.nodes {
			if n.Faction == f {
				n.RadioState = "connected"
				if n.ID != m.coords[f].NodeID {
					n.Role = "follower"
				}
				m.nodes[id] = n
			}
		}
		c := m.coords[f]
		c.Votes = 6
		m.coords[f] = c
	default:
		return domain.ArenaSnapshotV1{}, &Error{"INVALID_RADIO_FAULT", "Unknown radio-only fault."}
	}
	m.version++
	m.emit(f, "radio.fault", req.Kind, map[string]any{"management_connected": true, "inference_connected": true})
	return m.snapshot(f), nil
}

func (m *Manager) Advance(req domain.ArenaMutationV1, seconds int) (domain.ArenaSnapshotV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req); err != nil {
		return domain.ArenaSnapshotV1{}, err
	}
	if seconds < 1 || seconds > 3600 {
		return domain.ArenaSnapshotV1{}, &Error{"INVALID_ADVANCE", "Advance must be 1-3600 simulated seconds."}
	}
	m.tick += int64(seconds)
	hour := float64((6 + int(m.tick/3600)) % 24)
	sun := math.Max(0, math.Sin((hour-6)/12*math.Pi))
	for id, n := range m.nodes {
		base := .15
		coef := .085
		if n.Class == "mariner" {
			base, coef = .25, .16
		}
		if n.Class == "atlas" {
			base, coef = .45, .32
		}
		draw := (base + coef*math.Pow(n.SpeedMPS, 3)) * float64(seconds) / 3600
		charge := n.SolarKW * sun * float64(seconds) / 3600
		n.BatteryKWh = math.Max(0, math.Min(n.BatteryCapacityKWh, n.BatteryKWh+charge-draw))
		if n.BatteryKWh == 0 {
			n.SpeedMPS = 0
			n.Position[0] += .00002 * float64(seconds)
			n.Position[1] += .00001 * float64(seconds)
		}
		m.nodes[id] = n
	}
	m.version++
	m.emit("", "simulation.advanced", fmt.Sprintf("Advanced %d simulated seconds at 20×.", seconds), nil)
	return m.snapshot(req.ActorID), nil
}

type ActionRequest struct {
	domain.ArenaMutationV1
	SessionID string         `json:"session_id"`
	Faction   string         `json:"faction"`
	Kind      string         `json:"kind"`
	Arguments map[string]any `json:"arguments"`
}

func (m *Manager) WorkspaceAction(req ActionRequest) (domain.WorkspaceActionV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.WorkspaceActionV1{}, err
	}
	allowed := map[string]bool{"select": true, "open_window": true, "close_window": true, "minimize_window": true, "move_window": true, "set_map_view": true, "annotate_map": true, "frame_group": true}
	if !allowed[req.Kind] {
		return domain.WorkspaceActionV1{}, &Error{"HUMAN_APPROVAL_REQUIRED", "Effect and authority tools require an exact approval flow."}
	}
	a := domain.WorkspaceActionV1{ID: "workspace-" + short(req.IdempotencyKey), SessionID: req.SessionID, Faction: strings.ToUpper(req.Faction), Kind: req.Kind, AuthorityClass: "presentation", Arguments: req.Arguments, State: "applied", At: time.Now().UTC()}
	a.ResultHash = hash(a)
	m.actions = append(m.actions, a)
	if len(m.actions) > 50 {
		m.actions = m.actions[len(m.actions)-50:]
	}
	m.version++
	m.emit(a.Faction, "workspace.action", a.Kind, map[string]any{"action_id": a.ID})
	return a, nil
}

type AgentMessageRequest struct {
	domain.ArenaMutationV1
	Faction string `json:"faction"`
	Text    string `json:"text"`
	Persona string `json:"persona,omitempty"`
}

func (m *Manager) CreateSession(req AgentMessageRequest) (domain.AgentSessionV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.AgentSessionV1{}, err
	}
	f := strings.ToUpper(req.Faction)
	if f != "B" {
		f = "A"
	}
	c := m.coords[f]
	message := "Node agent ready with the same faction knowledge projection as the operator."
	if req.Persona == "pirate" {
		message = "Aye, Captain. Captain Barbossa is aboard with the same chart and faction knowledge ye can see—no secret waters and no authority beyond your signed orders."
	}
	s := domain.AgentSessionV1{ID: "agent-" + short(req.IdempotencyKey), Faction: f, State: "ready", CoordinatorID: c.NodeID, CoordinatorEpoch: c.Epoch, Message: message}
	m.sessions[s.ID] = s
	m.version++
	m.emit(f, "agent.session.created", s.Message, map[string]any{"session_id": s.ID})
	return s, nil
}
func (m *Manager) AgentMessage(sessionID string, req AgentMessageRequest) (domain.AgentSessionV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.AgentSessionV1{}, err
	}
	f := strings.ToUpper(req.Faction)
	c := m.coords[f]
	acts := []domain.WorkspaceActionV1{}
	add := func(kind string, args map[string]any) {
		a := domain.WorkspaceActionV1{ID: "workspace-" + short(req.IdempotencyKey+kind), SessionID: sessionID, Faction: f, Kind: kind, AuthorityClass: "presentation", Arguments: args, State: "applied", At: time.Now().UTC()}
		a.ResultHash = hash(a)
		acts = append(acts, a)
		m.actions = append(m.actions, a)
	}
	lower := strings.ToLower(req.Text)
	add("open_window", map[string]any{"window": "arena"})
	if strings.Contains(lower, "fleet") || strings.Contains(lower, "boat") {
		add("frame_group", map[string]any{"faction": f})
		add("select", map[string]any{"node_ids": m.eligible(f, "")})
	}
	if strings.Contains(lower, "contact") || strings.Contains(lower, "radar") {
		add("open_window", map[string]any{"window": "contacts"})
		add("annotate_map", map[string]any{"track_id": m.contacts[f][0].ID, "label": "uncertain radar contact"})
	}
	message := "I arranged your operating picture and marked the available evidence. I can draft the next bounded plan; confirm the exact proposal before any movement or simulated effect."
	if req.Persona == "pirate" {
		message = "Arrr, Captain—I’ve laid out the fleet and marked the evidence on our chart. I can plot the next bounded course, but no ship moves and no simulated broadside fires until ye confirm the exact signed proposal."
	}
	s := domain.AgentSessionV1{ID: sessionID, Faction: f, State: "awaiting_review", CoordinatorID: c.NodeID, CoordinatorEpoch: c.Epoch, Message: message, Actions: acts, AwaitingApproval: true}
	m.sessions[sessionID] = s
	m.version++
	m.emit(f, "agent.turn", s.Message, map[string]any{"actions": len(acts)})
	return s, nil
}

type PlanRequest struct {
	domain.ArenaMutationV1
	Faction         string   `json:"faction"`
	FriendlyNodeIDs []string `json:"friendly_node_ids"`
	TargetTrackIDs  []string `json:"target_track_ids"`
	Equipment       []string `json:"equipment"`
	MaximumEffects  int      `json:"maximum_effects"`
}

func (m *Manager) PlanEngagement(req PlanRequest) (domain.EngagementPlanV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.EngagementPlanV1{}, err
	}
	f := strings.ToUpper(req.Faction)
	c := m.coords[f]
	if c.Votes < 4 {
		return domain.EngagementPlanV1{}, &Error{"QUORUM_NOT_MET", "Current coordinator lacks faction quorum."}
	}
	known := map[string]bool{}
	for _, t := range m.contacts[f] {
		known[t.ID] = true
	}
	for _, id := range req.TargetTrackIDs {
		if !known[id] {
			return domain.EngagementPlanV1{}, &Error{"INSUFFICIENT_KNOWLEDGE", "Target is not in the faction knowledge projection."}
		}
	}
	if req.MaximumEffects < 1 {
		req.MaximumEffects = 1
	}
	p := domain.EngagementPlanV1{ID: "engagement-" + short(req.IdempotencyKey), Faction: f, CoordinatorEpoch: c.Epoch, FriendlyNodeIDs: req.FriendlyNodeIDs, TargetTrackIDs: req.TargetTrackIDs, Equipment: req.Equipment, MaximumEffects: req.MaximumEffects, StartsTick: m.tick + 10, ExpiresTick: m.tick + 180, Summary: "Bounded simulated engagement against known tracks in the current operating area.", PolicyStatus: "approval_required"}
	p.ContentHash = hash(p)
	m.plans[p.ID] = p
	m.version++
	m.emit(f, "engagement.planned", "Exact engagement proposal awaits operator approval.", map[string]any{"plan_hash": p.ContentHash})
	return p, nil
}

type AuthorizeRequest struct {
	domain.ArenaMutationV1
	PlanHash   string `json:"plan_hash"`
	OperatorID string `json:"operator_id"`
}

func (m *Manager) Authorize(planID string, req AuthorizeRequest) (domain.EngagementLeaseV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.EngagementLeaseV1{}, err
	}
	p, ok := m.plans[planID]
	if !ok {
		return domain.EngagementLeaseV1{}, &Error{"ENGAGEMENT_NOT_FOUND", "Engagement plan not found."}
	}
	if p.ContentHash != req.PlanHash {
		return domain.EngagementLeaseV1{}, &Error{"ENGAGEMENT_HASH_MISMATCH", "Exact engagement hash does not match."}
	}
	c := m.coords[p.Faction]
	if c.Epoch != p.CoordinatorEpoch || c.Votes < 4 {
		return domain.EngagementLeaseV1{}, &Error{"COORDINATOR_STALE", "Coordinator authority changed."}
	}
	operator := req.OperatorID
	if operator == "" {
		operator = req.ActorID
	}
	if operator == "" {
		return domain.EngagementLeaseV1{}, &Error{"HUMAN_APPROVAL_REQUIRED", "Operator identity is required."}
	}
	l := domain.EngagementLeaseV1{ID: "lease-" + short(req.IdempotencyKey), PlanID: p.ID, PlanHash: p.ContentHash, Faction: p.Faction, CoordinatorEpoch: p.CoordinatorEpoch, OperatorID: operator, RemainingEffects: p.MaximumEffects, ExpiresTick: p.ExpiresTick}
	l.Signature = m.sign(l)
	m.leases[l.ID] = l
	m.version++
	m.emit(p.Faction, "engagement.authorized", "Scoped simulated engagement lease authorized.", map[string]any{"lease_id": l.ID})
	return l, nil
}

type EffectRequest struct {
	domain.ArenaMutationV1
	LeaseID       string `json:"lease_id"`
	TargetTrackID string `json:"target_track_id"`
	Equipment     string `json:"equipment"`
}

func (m *Manager) ApplyEffect(req EffectRequest) (domain.ArenaEffectV1, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.ArenaMutationV1); err != nil {
		return domain.ArenaEffectV1{}, err
	}
	l, ok := m.leases[req.LeaseID]
	if !ok {
		return domain.ArenaEffectV1{}, &Error{"ENGAGEMENT_LEASE_REQUIRED", "A valid engagement lease is required."}
	}
	if m.tick > l.ExpiresTick || l.RemainingEffects < 1 {
		return domain.ArenaEffectV1{}, &Error{"ENGAGEMENT_LEASE_EXPIRED", "The engagement lease is exhausted or expired."}
	}
	p := m.plans[l.PlanID]
	if !contains(p.TargetTrackIDs, req.TargetTrackID) || !contains(p.Equipment, req.Equipment) {
		return domain.ArenaEffectV1{}, &Error{"OUTSIDE_ENGAGEMENT_AUTHORITY", "Target or equipment is outside the exact lease."}
	}
	l.RemainingEffects--
	m.leases[l.ID] = l
	outcome := []string{"miss · uncertainty retained", "simulated sensor disruption", "simulated mobility disabled"}[int(m.tick+int64(len(m.effects)))%3]
	e := domain.ArenaEffectV1{ID: "effect-" + short(req.IdempotencyKey), LeaseID: l.ID, PlanID: p.ID, Faction: p.Faction, TargetTrackID: req.TargetTrackID, Equipment: req.Equipment, Outcome: outcome, RemainingUses: l.RemainingEffects, WorldTick: m.tick}
	e.ReceiptHash = hash(e)
	m.effects = append(m.effects, e)
	m.version++
	m.emit(p.Faction, "effect.applied", outcome, map[string]any{"receipt_hash": e.ReceiptHash})
	return e, nil
}

func (m *Manager) check(req domain.ArenaMutationV1) error {
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return &Error{"MUTATION_ID_REQUIRED", "Request and idempotency IDs are required."}
	}
	if req.ExpectedVersion != m.version {
		return &Error{"ARENA_STALE_STATE", fmt.Sprintf("Expected version does not match %d.", m.version)}
	}
	if prior, ok := m.idempotency[req.IdempotencyKey]; ok && prior != req.RequestID {
		return &Error{"IDEMPOTENCY_CONFLICT", "Idempotency key already belongs to another request."}
	}
	m.idempotency[req.IdempotencyKey] = req.RequestID
	return nil
}
func (m *Manager) eligible(f, except string) []string {
	out := []string{}
	for id, n := range m.nodes {
		if n.Faction == f && id != except && n.RadioState != "partitioned" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
func (m *Manager) radioSummary(f string) string {
	for _, n := range m.nodes {
		if n.Faction == f && n.RadioState == "partitioned" {
			return "degraded · quorum-routed"
		}
		if n.Faction == f && n.RadioState == "halow-only" {
			return "HaLow-only"
		}
	}
	return "Starlink + HaLow"
}
func (m *Manager) emit(f, k, s string, d map[string]any) {
	m.eventSeq++
	m.events = append(m.events, domain.ArenaEventV1{Sequence: m.eventSeq, Tick: m.tick, Kind: k, Faction: f, Summary: s, Details: d})
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
func short(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:])[:12] }
func (m *Manager) sign(v any) string {
	mac := hmac.New(sha256.New, m.secret)
	b, _ := json.Marshal(v)
	mac.Write(b)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
