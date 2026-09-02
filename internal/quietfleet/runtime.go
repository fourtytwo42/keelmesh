package quietfleet

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const (
	EnterMode         = "enter_mode"
	InjectSlowdown    = "inject_slowdown"
	SubmitRevision    = "submit_revision"
	CommitProposal    = "commit_proposal"
	AdvanceActivation = "advance_to_activation"
	AutoRun           = "auto_run"
)

type Runtime struct {
	snapshot domain.QuietFleetSnapshotV1
	key      []byte
	original map[string]domain.AssignmentV1
}

func New(lease domain.MissionLeaseV1, plan domain.PlanCandidateV1, stateVersion int64) *Runtime {
	seed := sha256.Sum256([]byte(plan.ContentHash + "|quiet-fleet-v1"))
	r := &Runtime{key: seed[:], original: map[string]domain.AssignmentV1{}}
	for _, assignment := range plan.Assignments {
		if member(assignment.VesselID) {
			r.original[assignment.VesselID] = cloneAssignment(assignment)
		}
	}
	contract := domain.GroupMissionContractV1{SchemaVersion: 1, ID: "group-contract-v1", MissionID: lease.MissionID, PlanHash: plan.ContentHash, AuthorityEpoch: 1, Members: members(), CoordinatorOrder: []string{"vessel-02", "vessel-03", "vessel-05", "vessel-04"}, Quorum: 3, WindowIntervalSeconds: 20, MaximumBytesPerWindow: 4096, MaximumRounds: 3, MinimumActivationLead: 20, TapeBoundarySeconds: 10, AllowedAdaptations: []string{"pace", "lane_reassignment", "workload_redistribution"}, BulkTrafficSuppressed: true}
	contract.ContentHash = hash(contractWire(contract))
	contract.Signature = r.sign(contract.ContentHash)
	r.snapshot = domain.QuietFleetSnapshotV1{SchemaVersion: 1, StateVersion: stateVersion, ScenarioID: "quiet-fleet-v1", Phase: "ready", Contract: contract, CoordinatorID: contract.CoordinatorOrder[0], Vessel4SpeedMPS: speed(r.original["vessel-04"]), ActiveAssignments: assignments(r.original), Decisions: []domain.NodeDecisionV1{}, Windows: []domain.CoordinationWindowV1{}, Metrics: domain.CoordinationMetricsV1{ByteBudget: contract.MaximumBytesPerWindow * contract.MaximumRounds, QuorumRequired: contract.Quorum, AffectedRequired: 4}, Summary: "Four vessels are eligible for the signed Quiet Fleet coordination cell.", NextAction: EnterMode, AutoRunAvailable: true, InferenceLabel: "simulated edge inference · deterministic fixture"}
	return r
}

func (r *Runtime) Snapshot() domain.QuietFleetSnapshotV1 {
	b, _ := json.Marshal(r.snapshot)
	var out domain.QuietFleetSnapshotV1
	_ = json.Unmarshal(b, &out)
	return out
}

func (r *Runtime) SetStateVersion(version int64) { r.snapshot.StateVersion = version }

func (r *Runtime) Apply(kind, proposalHash string) (string, error) {
	switch kind {
	case EnterMode:
		if r.snapshot.Phase != "ready" {
			return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
		}
		r.snapshot.Phase, r.snapshot.NextAction = "quiet", InjectSlowdown
		r.snapshot.Summary = "Quiet Fleet entered low-radio-duty mode under the existing signed mission authority."
		return "quiet_fleet.mode.entered", nil
	case InjectSlowdown:
		if r.snapshot.Phase != "quiet" {
			return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
		}
		r.snapshot.MissionTick = nextWindow(r.snapshot.MissionTick, r.snapshot.Contract.WindowIntervalSeconds)
		r.snapshot.Vessel4SpeedMPS = .8
		proposal := r.buildProposal(1)
		decisions := r.decide(proposal, true)
		if err := r.recordWindow(proposal, decisions); err != nil {
			return "", err
		}
		r.snapshot.Proposal, r.snapshot.Decisions = &proposal, decisions
		r.refreshMetrics()
		r.snapshot.Phase, r.snapshot.NextAction = "proposal_rejected", SubmitRevision
		r.snapshot.Summary = "Proposal 1 reached quorum, but Vessel 2 rejected an unsafe 2.2 m/s assignment; no commit exists."
		return "quiet_fleet.proposal.rejected", nil
	case SubmitRevision:
		if r.snapshot.Phase != "proposal_rejected" {
			return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
		}
		r.snapshot.MissionTick = nextWindow(r.snapshot.MissionTick, r.snapshot.Contract.WindowIntervalSeconds)
		proposal := r.buildProposal(2)
		decisions := r.decide(proposal, false)
		if err := r.recordWindow(proposal, decisions); err != nil {
			return "", err
		}
		r.snapshot.Proposal, r.snapshot.Decisions = &proposal, decisions
		r.refreshMetrics()
		r.snapshot.Phase, r.snapshot.NextAction = "revised_armed", CommitProposal
		r.snapshot.Summary = "Revised assignments satisfy every local envelope; original-cell quorum and all affected arms are present."
		return "quiet_fleet.proposal.armed", nil
	case CommitProposal:
		if (r.snapshot.Phase != "proposal_rejected" && r.snapshot.Phase != "revised_armed") || r.snapshot.Proposal == nil {
			return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
		}
		if proposalHash == "" || proposalHash != r.snapshot.Proposal.ContentHash {
			return "", fmt.Errorf("COMMIT_HASH_MISMATCH")
		}
		if r.snapshot.Metrics.QuorumCount < r.snapshot.Contract.Quorum {
			return "", fmt.Errorf("QUORUM_NOT_MET")
		}
		if r.snapshot.Metrics.AffectedArmed != r.snapshot.Metrics.AffectedRequired {
			return "", fmt.Errorf("AFFECTED_NODE_NOT_ARMED")
		}
		activation := int64(math.Ceil(float64(r.snapshot.MissionTick+r.snapshot.Contract.MinimumActivationLead)/float64(r.snapshot.Contract.TapeBoundarySeconds))) * r.snapshot.Contract.TapeBoundarySeconds
		commit := domain.GroupCommitV1{SchemaVersion: 1, ID: "group-commit-v1", ProposalHash: proposalHash, AuthorityEpoch: 1, CommitTick: r.snapshot.MissionTick, ActivationTick: activation, ArmedNodes: armedNodes(r.snapshot.Decisions)}
		commit.ContentHash = hash(commitWire(commit))
		commit.Signature = r.sign(commit.ContentHash)
		bytes, _ := json.Marshal(commit)
		if err := r.recordCommitWindow(len(bytes)); err != nil {
			return "", err
		}
		r.snapshot.Commit = &commit
		r.snapshot.Phase, r.snapshot.NextAction = "committed", AdvanceActivation
		r.snapshot.Summary = fmt.Sprintf("Exact proposal committed for future tape boundary tick %d; active routes remain unchanged.", activation)
		return "quiet_fleet.commit.prepared", nil
	case AdvanceActivation:
		if r.snapshot.Phase != "committed" || r.snapshot.Commit == nil || r.snapshot.Proposal == nil {
			return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
		}
		r.snapshot.MissionTick = r.snapshot.Commit.ActivationTick
		r.snapshot.ActiveAssignments = cloneAssignments(r.snapshot.Proposal.Assignments)
		r.snapshot.Phase, r.snapshot.NextAction = "activated", ""
		r.snapshot.Summary = "Future boundary reached; the signed revision activated atomically without replay or authority expansion."
		return "quiet_fleet.commit.activated", nil
	case AutoRun:
		for _, command := range []string{EnterMode, InjectSlowdown, SubmitRevision} {
			if _, err := r.Apply(command, ""); err != nil {
				return "", err
			}
		}
		if _, err := r.Apply(CommitProposal, r.snapshot.Proposal.ContentHash); err != nil {
			return "", err
		}
		return r.Apply(AdvanceActivation, "")
	default:
		return "", fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
	}
}

func (r *Runtime) Advance(seconds int64) error {
	if seconds <= 0 || seconds > 60 {
		return fmt.Errorf("INVALID_QUIET_FLEET_PHASE")
	}
	if r.snapshot.Commit != nil && r.snapshot.MissionTick+seconds >= r.snapshot.Commit.ActivationTick {
		_, err := r.Apply(AdvanceActivation, "")
		return err
	}
	r.snapshot.MissionTick += seconds
	return nil
}

func (r *Runtime) buildProposal(revision int) domain.GroupAdaptationProposalV1 {
	updated := map[string]domain.AssignmentV1{}
	for id, assignment := range r.original {
		updated[id] = cloneAssignment(assignment)
	}
	if revision == 1 {
		a := updated["vessel-02"]
		a.SpeedMPS = 2.2
		updated["vessel-02"] = a
		a = updated["vessel-04"]
		a.SpeedMPS = .8
		updated["vessel-04"] = a
	} else {
		for id, target := range map[string]float64{"vessel-02": 1.5, "vessel-03": 1.6, "vessel-04": .8, "vessel-05": 1.6} {
			a := updated[id]
			a.SpeedMPS = target
			updated[id] = a
		}
		v4 := updated["vessel-04"]
		if len(v4.Route) > 3 {
			cut := len(v4.Route) / 2
			v3 := updated["vessel-03"]
			v3.Route = append(v3.Route, v4.Route[cut:]...)
			updated["vessel-03"] = v3
			v5 := updated["vessel-05"]
			v5.Route = append(v5.Route, v4.Route[cut:]...)
			updated["vessel-05"] = v5
			v4.Route = v4.Route[:cut+1]
			updated["vessel-04"] = v4
		}
	}
	p := domain.GroupAdaptationProposalV1{SchemaVersion: 1, ID: fmt.Sprintf("adaptation-v%d", revision), Revision: revision, AuthorityEpoch: 1, CoordinatorID: r.snapshot.CoordinatorID, Reason: "Vessel 4 propulsion degraded to 0.8 m/s", Source: r.snapshot.InferenceLabel, CreatedTick: r.snapshot.MissionTick, ExpiresTick: r.snapshot.MissionTick + 40, AffectedNodes: members(), Assignments: assignments(updated)}
	p.ContentHash = hash(proposalWire(p))
	p.Signature = r.sign(p.ContentHash)
	return p
}

func (r *Runtime) decide(p domain.GroupAdaptationProposalV1, rejectV2 bool) []domain.NodeDecisionV1 {
	out := make([]domain.NodeDecisionV1, 0, 4)
	for _, id := range members() {
		decision, reason := "armed", ""
		if id == "vessel-02" && rejectV2 {
			decision, reason = "reject", "SPEED_ENVELOPE_EXCEEDED"
		}
		d := domain.NodeDecisionV1{SchemaVersion: 1, NodeID: id, ProposalHash: p.ContentHash, Decision: decision, ReasonCode: reason, DecidedTick: r.snapshot.MissionTick}
		d.Signature = r.sign(hash(d))
		out = append(out, d)
	}
	return out
}

func (r *Runtime) recordWindow(p domain.GroupAdaptationProposalV1, decisions []domain.NodeDecisionV1) error {
	payload, _ := json.Marshal(struct {
		Proposal  domain.GroupAdaptationProposalV1 `json:"proposal"`
		Decisions []domain.NodeDecisionV1          `json:"decisions"`
	}{p, decisions})
	if len(payload) > r.snapshot.Contract.MaximumBytesPerWindow {
		return fmt.Errorf("COORDINATION_BUDGET_EXCEEDED")
	}
	if len(r.snapshot.Windows) >= r.snapshot.Contract.MaximumRounds {
		return fmt.Errorf("COORDINATION_BUDGET_EXCEEDED")
	}
	r.snapshot.Windows = append(r.snapshot.Windows, domain.CoordinationWindowV1{Round: len(r.snapshot.Windows) + 1, OpensTick: r.snapshot.MissionTick, ClosesTick: r.snapshot.MissionTick + 2, BytesUsed: len(payload), ByteBudget: r.snapshot.Contract.MaximumBytesPerWindow, MessageCount: len(decisions) + 1, State: "closed"})
	return nil
}

func (r *Runtime) recordCommitWindow(bytes int) error {
	if bytes > r.snapshot.Contract.MaximumBytesPerWindow || len(r.snapshot.Windows) >= r.snapshot.Contract.MaximumRounds {
		return fmt.Errorf("COORDINATION_BUDGET_EXCEEDED")
	}
	r.snapshot.Windows = append(r.snapshot.Windows, domain.CoordinationWindowV1{Round: len(r.snapshot.Windows) + 1, OpensTick: r.snapshot.MissionTick, ClosesTick: r.snapshot.MissionTick + 2, BytesUsed: bytes, ByteBudget: r.snapshot.Contract.MaximumBytesPerWindow, MessageCount: 1, State: "closed"})
	r.refreshMetrics()
	return nil
}

func (r *Runtime) refreshMetrics() {
	metrics := domain.CoordinationMetricsV1{Rounds: len(r.snapshot.Windows), ByteBudget: r.snapshot.Contract.MaximumBytesPerWindow * r.snapshot.Contract.MaximumRounds, QuorumRequired: r.snapshot.Contract.Quorum, AffectedRequired: 4, BulkMessagesSuppressed: len(r.snapshot.Windows) * 6}
	for _, window := range r.snapshot.Windows {
		metrics.BytesSent += window.BytesUsed
		metrics.MessagesSent += window.MessageCount
	}
	for _, decision := range r.snapshot.Decisions {
		if decision.Decision == "armed" {
			metrics.QuorumCount++
			metrics.AffectedArmed++
		}
	}
	r.snapshot.Metrics = metrics
}

func members() []string { return []string{"vessel-02", "vessel-03", "vessel-04", "vessel-05"} }
func member(id string) bool {
	for _, candidate := range members() {
		if id == candidate {
			return true
		}
	}
	return false
}
func assignments(in map[string]domain.AssignmentV1) []domain.AssignmentV1 {
	ids := make([]string, 0, len(in))
	for id := range in {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]domain.AssignmentV1, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneAssignment(in[id]))
	}
	return out
}
func cloneAssignment(in domain.AssignmentV1) domain.AssignmentV1 {
	in.Route = append([]domain.Point(nil), in.Route...)
	return in
}
func cloneAssignments(in []domain.AssignmentV1) []domain.AssignmentV1 {
	out := make([]domain.AssignmentV1, len(in))
	for i, a := range in {
		out[i] = cloneAssignment(a)
	}
	return out
}
func speed(a domain.AssignmentV1) float64 {
	if a.SpeedMPS == 0 {
		return 1.4
	}
	return a.SpeedMPS
}
func nextWindow(tick, interval int64) int64 { return ((tick / interval) + 1) * interval }
func armedNodes(decisions []domain.NodeDecisionV1) []string {
	out := []string{}
	for _, d := range decisions {
		if d.Decision == "armed" {
			out = append(out, d.NodeID)
		}
	}
	sort.Strings(out)
	return out
}
func hash(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func (r *Runtime) sign(value string) string {
	mac := hmac.New(sha256.New, r.key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
func contractWire(v domain.GroupMissionContractV1) any {
	v.ContentHash = ""
	v.Signature = ""
	return v
}
func proposalWire(v domain.GroupAdaptationProposalV1) any {
	v.ContentHash = ""
	v.Signature = ""
	return v
}
func commitWire(v domain.GroupCommitV1) any { v.ContentHash = ""; v.Signature = ""; return v }
