package coordination

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/edgeexec"
	"github.com/fourtytwo42/keelmesh/internal/observability"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type Manager struct {
	cfg             Config
	logger          *slog.Logger
	fsm             *stateMachine
	raft            *raft.Raft
	transport       *raft.NetworkTransport
	raftTLS         *tlsConfigSwitcher
	managementTLS   *tlsConfigSwitcher
	store           io.Closer
	signKey         ed25519.PrivateKey
	refereeKey      ed25519.PublicKey
	ready           atomic.Bool
	closed          atomic.Bool
	elections       atomic.Uint64
	electionStarted atomic.Int64
	lastElectionMS  atomic.Int64
	mu              sync.RWMutex
	server          *http.Server
	client          *http.Client
	startedAt       time.Time
	tracer          *observability.Tracer
	execution       *edgeexec.Store
}

func NewManager(cfg Config, logger *slog.Logger) (*Manager, error) {
	if cfg.Mode == ModeSimulated || cfg.Identity.NodeID == "" {
		return &Manager{cfg: cfg, logger: logger, fsm: newStateMachine(), startedAt: nowUTC(), tracer: observability.NewTracer(observability.ConfigFromEnv("keelmesh-coordination-node"), logger)}, nil
	}
	if err := validateManifest(cfg.Manifest); err != nil {
		return nil, err
	}
	if err := validateConfigIdentity(cfg); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create coordination data directory: %w", err)
	}
	if err := os.Chmod(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("secure coordination data directory: %w", err)
	}
	serverTLS, clientTLS, err := loadNodeTLSConfigs(cfg.Identity, cfg.Manifest, cfg.RaftCertificateFile, cfg.RaftTLSKeyFile, cfg.TrustBundleFile, radioPlane, false)
	if err != nil {
		return nil, err
	}
	raftTLS := newTLSConfigSwitcher(serverTLS, clientTLS)
	stream, err := newTLSStreamLayer(cfg.RaftAddress, raftTLS)
	if err != nil {
		return nil, fmt.Errorf("listen on Raft radio address: %w", err)
	}
	transport := raft.NewNetworkTransport(stream, 3, 3*time.Second, os.Stderr)
	store, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft.db"))
	if err != nil {
		_ = transport.Close()
		return nil, fmt.Errorf("open Raft store: %w", err)
	}
	snapshots, err := raft.NewFileSnapshotStore(cfg.DataDir, 2, io.Discard)
	if err != nil {
		_ = transport.Close()
		return nil, fmt.Errorf("open Raft snapshots: %w", err)
	}
	leadership := make(chan bool, 4)
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.Identity.NodeID)
	raftConfig.SnapshotThreshold = cfg.SnapshotThreshold
	raftConfig.SnapshotInterval = cfg.SnapshotInterval
	raftConfig.NotifyCh = leadership
	raftConfig.LogOutput = os.Stderr
	fsm := newStateMachine()
	existing, err := raft.HasExistingState(store, store, snapshots)
	if err != nil {
		_ = store.Close()
		_ = transport.Close()
		return nil, fmt.Errorf("inspect Raft state: %w", err)
	}
	nodeRaft, err := raft.NewRaft(raftConfig, fsm, store, store, snapshots, transport)
	if err != nil {
		_ = store.Close()
		_ = transport.Close()
		return nil, fmt.Errorf("start Raft: %w", err)
	}
	if !existing && cfg.Bootstrap {
		servers := make([]raft.Server, 0, len(cfg.Manifest.Members))
		for _, member := range cfg.Manifest.Members {
			servers = append(servers, raft.Server{ID: raft.ServerID(member.NodeID), Address: raft.ServerAddress(member.RadioAddress), Suffrage: raft.Voter})
		}
		if err := nodeRaft.BootstrapCluster(raft.Configuration{Servers: servers}).Error(); err != nil && !strings.Contains(err.Error(), "bootstrap only works on new clusters") {
			_ = nodeRaft.Shutdown().Error()
			return nil, fmt.Errorf("bootstrap fixed cell: %w", err)
		}
	}
	signKey, err := loadSigningKey(cfg.SigningKeyFile)
	if err != nil {
		_ = nodeRaft.Shutdown().Error()
		return nil, fmt.Errorf("load application signing key: %w", err)
	}
	refereeKey, err := loadSigningPublicKey(cfg.RefereeSigningPublicKeyFile)
	if err != nil {
		_ = nodeRaft.Shutdown().Error()
		_ = store.Close()
		return nil, fmt.Errorf("load referee signing public key: %w", err)
	}
	execution, err := edgeexec.Open(filepath.Join(cfg.DataDir, "edge-execution.bbolt"), cfg.Identity.NodeID)
	if err != nil {
		_ = nodeRaft.Shutdown().Error()
		_ = store.Close()
		return nil, fmt.Errorf("open edge execution store: %w", err)
	}
	manager := &Manager{cfg: cfg, logger: logger, fsm: fsm, raft: nodeRaft, transport: transport, raftTLS: raftTLS, store: store, signKey: signKey, refereeKey: refereeKey, startedAt: nowUTC(), tracer: observability.NewTracer(observability.ConfigFromEnv("keelmesh-coordination-node"), logger), execution: execution}
	go manager.watchLeadership(leadership)
	return manager, nil
}

func validateConfigIdentity(cfg Config) error {
	for _, member := range cfg.Manifest.Members {
		if member.NodeID != cfg.Identity.NodeID {
			continue
		}
		if member.Faction != cfg.Identity.CellID || member.RadioAddress != cfg.RaftAddress || strings.Split(member.ManagementAddress, ":")[0] != cfg.Identity.ManagementIP {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: configured identity differs from signed manifest")
		}
		return nil
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: local node is not in signed manifest")
}

func (m *Manager) watchLeadership(ch <-chan bool) {
	for leader := range ch {
		m.ready.Store(false)
		if !leader || m.closed.Load() {
			m.electionStarted.Store(time.Now().UnixNano())
			continue
		}
		m.elections.Add(1)
		if started := m.electionStarted.Swap(0); started > 0 {
			m.lastElectionMS.Store(time.Since(time.Unix(0, started)).Milliseconds())
		}
		go m.advanceEpoch()
	}
}

func (m *Manager) advanceEpoch() {
	for m.raft != nil && m.raft.State() == raft.Leader && !m.closed.Load() {
		term := parseUint(m.raft.Stats()["term"])
		// A newly elected leader can be announced before its restored FSM has
		// applied every committed entry.  Read the epoch only after a barrier so
		// the next epoch is derived from committed state, never startup state.
		if err := m.raft.Barrier(m.cfg.ApplyTimeout).Error(); err != nil {
			m.logger.Warn("coordination leader barrier pending", "cell", m.cfg.Identity.CellID, "term", term, "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if m.raft.State() != raft.Leader || parseUint(m.raft.Stats()["term"]) != term {
			continue
		}
		epoch, _, _, _ := m.fsm.summary()
		command := m.epochAdvanceCommand(epoch, term, nowUTC())
		if _, err := m.apply(command); err != nil {
			m.logger.Error("coordination leader epoch advance failed", "cell", m.cfg.Identity.CellID, "term", term, "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if m.raft.State() == raft.Leader && parseUint(m.raft.Stats()["term"]) == term {
			m.ready.Store(true)
		}
		return
	}
}

func (m *Manager) epochAdvanceCommand(epoch, term uint64, issuedAt time.Time) domain.ReplicatedCommandV1 {
	payload, hash, _ := canonicalPayload(map[string]any{"term": term, "leader": m.cfg.Identity.NodeID})
	identity := fmt.Sprintf("epoch-%s-term-%d-%s", m.cfg.Manifest.ClusterID, term, m.cfg.Identity.NodeID)
	return domain.ReplicatedCommandV1{
		SchemaVersion: 1, CommandID: identity, RequestID: "leadership-" + identity,
		IdempotencyKey: identity, ActorIdentity: "coordination-runtime",
		CellID: m.cfg.Identity.CellID, Term: term, AuthorityEpoch: epoch + 1,
		Kind: "coordination.epoch_advance", EntityID: m.cfg.Manifest.ClusterID,
		Payload: payload, PayloadHash: hash, IssuedAt: issuedAt.UTC(),
	}
}

func (m *Manager) Propose(ctx context.Context, command domain.ReplicatedCommandV1) (domain.AppliedCommandReceiptV1, error) {
	if m.cfg.Mode == ModeSimulated || m.raft == nil {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("QUORUM_UNAVAILABLE: Raft coordination is disabled")
	}
	if m.raft.State() != raft.Leader {
		_, leaderID := m.raft.LeaderWithID()
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("NOT_COORDINATOR: current leader is %s", leaderID)
	}
	if !m.ready.Load() {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("LEADER_NOT_READY: leader epoch has not committed")
	}
	if err := m.raft.Barrier(m.cfg.ApplyTimeout).Error(); err != nil {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("QUORUM_UNAVAILABLE: leader barrier failed: %w", err)
	}
	epoch, _, _, _ := m.fsm.summary()
	command.CellID = m.cfg.Identity.CellID
	command.Term = parseUint(m.raft.Stats()["term"])
	command.AuthorityEpoch = epoch
	if command.IssuedAt.IsZero() {
		command.IssuedAt = nowUTC()
	}
	if err := m.validateCrossCellCommand(command); err != nil {
		return domain.AppliedCommandReceiptV1{}, err
	}
	select {
	case <-ctx.Done():
		return domain.AppliedCommandReceiptV1{}, ctx.Err()
	default:
	}
	_, span := m.tracer.Start(ctx, "raft.propose_apply", map[string]string{"coordination.cell": command.CellID, "coordination.term": strconv.FormatUint(command.Term, 10), "coordination.epoch": strconv.FormatUint(command.AuthorityEpoch, 10), "command.id": command.CommandID, "command.kind": command.Kind, "command.hash": command.PayloadHash})
	receipt, err := m.apply(command)
	state := "committed"
	if err != nil {
		state = "error"
	}
	span.End(state, map[string]string{"raft.log.index": strconv.FormatUint(receipt.LogIndex, 10), "result.state_hash": receipt.ResultingStateHash})
	return receipt, err
}

func (m *Manager) Mode() Mode { return m.cfg.Mode }

func (m *Manager) CellID() string { return m.cfg.Identity.CellID }

// ProposeOrForward commits locally when this node is leader and otherwise forwards
// the exact canonical command to the current same-cell leader over management mTLS.
func (m *Manager) ProposeOrForward(ctx context.Context, command domain.ReplicatedCommandV1) (domain.AppliedCommandReceiptV1, error) {
	if m.raft == nil {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("QUORUM_UNAVAILABLE: Raft coordination is disabled")
	}
	if m.raft.State() == raft.Leader {
		return m.Propose(ctx, command)
	}
	_, leaderID := m.raft.LeaderWithID()
	if leaderID == "" {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("LEADER_NOT_READY: no current leader")
	}
	member, ok := m.member(string(leaderID))
	if !ok {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("CELL_MEMBERSHIP_DENIED: elected leader is not in the signed manifest")
	}
	if m.client == nil {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("LEADER_NOT_READY: management forwarding client is unavailable")
	}
	var receipt domain.AppliedCommandReceiptV1
	if err := m.requestManagementJSON(ctx, http.MethodPost, "https://"+member.ManagementAddress+"/internal/v1/coordination/commands:propose", command, &receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func (m *Manager) Commit(ctx context.Context, command domain.ReplicatedCommandV1) (domain.AppliedCommandReceiptV1, domain.QuorumCommitProofV1, error) {
	receipt, err := m.ProposeOrForward(ctx, command)
	if err != nil {
		return receipt, domain.QuorumCommitProofV1{}, err
	}
	ctx, proofSpan := m.tracer.Start(ctx, "quorum.proof_collect", map[string]string{"coordination.cell": receipt.CellID, "command.id": receipt.CommandID, "raft.log.index": strconv.FormatUint(receipt.LogIndex, 10), "quorum.required": strconv.Itoa(m.cfg.Manifest.Quorum)})
	proof := domain.QuorumCommitProofV1{SchemaVersion: 1, CommandID: receipt.CommandID, CellID: receipt.CellID, Term: receipt.Term, LogIndex: receipt.LogIndex, AuthorityEpoch: receipt.AuthorityEpoch, CommandHash: receipt.CommandHash, ResultingStateHash: receipt.ResultingStateHash, Required: m.cfg.Manifest.Quorum, State: "collecting"}
	seen := map[string]bool{}
	for {
		for _, peer := range m.cfg.Manifest.Members {
			if seen[peer.NodeID] {
				continue
			}
			var acknowledgement domain.SignedNodeAcknowledgementV1
			attemptCtx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
			err := m.requestManagementJSON(attemptCtx, http.MethodGet, "https://"+peer.ManagementAddress+"/internal/v1/coordination/proofs/"+receipt.CommandID, nil, &acknowledgement)
			cancel()
			if err != nil || verifyAcknowledgement(m.cfg.Manifest, receipt, acknowledgement) != nil {
				continue
			}
			seen[acknowledgement.NodeID] = true
			proof.Acknowledgements = append(proof.Acknowledgements, acknowledgement)
		}
		if len(proof.Acknowledgements) >= proof.Required {
			sort.Slice(proof.Acknowledgements, func(i, j int) bool { return proof.Acknowledgements[i].NodeID < proof.Acknowledgements[j].NodeID })
			proof.State = "verified"
			proof.CompletedAt = nowUTC()
			proofSpan.End("completed", map[string]string{"quorum.acknowledgements": strconv.Itoa(len(proof.Acknowledgements))})
			return receipt, proof, nil
		}
		select {
		case <-ctx.Done():
			proofSpan.End("error", map[string]string{"quorum.acknowledgements": strconv.Itoa(len(proof.Acknowledgements))})
			return receipt, proof, fmt.Errorf("COMMIT_PROOF_INVALID: only %d of %d acknowledgements were verified", len(proof.Acknowledgements), proof.Required)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (m *Manager) member(nodeID string) (domain.CoordinationCellMemberV1, bool) {
	for _, member := range m.cfg.Manifest.Members {
		if member.NodeID == nodeID {
			return member, true
		}
	}
	return domain.CoordinationCellMemberV1{}, false
}

func (m *Manager) requestManagementJSON(ctx context.Context, method, url string, body any, destination any) error {
	ctx, span := m.tracer.Start(ctx, "coordination.peer "+method, map[string]string{"server.address": url, "coordination.transport": "mtls-management"})
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		span.End("error", map[string]string{"error.type": "request_build"})
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("traceparent", span.Traceparent())
	response, err := m.client.Do(request)
	if err != nil {
		span.End("error", map[string]string{"error.type": "transport"})
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		span.End("error", map[string]string{"http.response.status_code": strconv.Itoa(response.StatusCode)})
		var apiErr domain.APIError
		if json.NewDecoder(io.LimitReader(response.Body, maxCoordinationBody)).Decode(&apiErr) == nil && apiErr.Code != "" {
			return fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
		}
		return fmt.Errorf("QUORUM_UNAVAILABLE: management peer returned %s", response.Status)
	}
	if destination == nil {
		span.End("completed", map[string]string{"http.response.status_code": strconv.Itoa(response.StatusCode)})
		return nil
	}
	err = json.NewDecoder(io.LimitReader(response.Body, maxCoordinationBody)).Decode(destination)
	state := "completed"
	if err != nil {
		state = "error"
	}
	span.End(state, map[string]string{"http.response.status_code": strconv.Itoa(response.StatusCode)})
	return err
}

func (m *Manager) apply(command domain.ReplicatedCommandV1) (domain.AppliedCommandReceiptV1, error) {
	encoded, err := json.Marshal(command)
	if err != nil {
		return domain.AppliedCommandReceiptV1{}, err
	}
	future := m.raft.Apply(encoded, m.cfg.ApplyTimeout)
	if err := future.Error(); err != nil {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("RAFT_APPLY_TIMEOUT: %w", err)
	}
	result, ok := future.Response().(applyResult)
	if !ok {
		return domain.AppliedCommandReceiptV1{}, fmt.Errorf("RAFT_APPLY_TIMEOUT: unexpected FSM response")
	}
	if result.err != nil {
		return domain.AppliedCommandReceiptV1{}, result.err
	}
	return result.receipt, nil
}

func (m *Manager) Acknowledgement(commandID string) (domain.SignedNodeAcknowledgementV1, error) {
	receipt, ok := m.fsm.receipt(commandID)
	if !ok {
		return domain.SignedNodeAcknowledgementV1{}, fmt.Errorf("command receipt not found")
	}
	ack := domain.SignedNodeAcknowledgementV1{NodeID: m.cfg.Identity.NodeID, CellID: receipt.CellID, Term: receipt.Term, LogIndex: receipt.LogIndex, AuthorityEpoch: receipt.AuthorityEpoch, CommandHash: receipt.CommandHash, ResultingStateHash: receipt.ResultingStateHash}
	payload, _ := acknowledgementPayload(ack)
	ack.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(m.signKey, payload))
	return ack, nil
}

func (m *Manager) Receipts(limit int) []domain.AppliedCommandReceiptV1 { return m.fsm.receipts(limit) }

func acknowledgementPayload(ack domain.SignedNodeAcknowledgementV1) ([]byte, error) {
	ack.Signature = ""
	return json.Marshal(ack)
}

func (m *Manager) Advertisement() (domain.CoordinatorAdvertisementV1, error) {
	snapshot := m.Snapshot()
	state := "electing"
	if m.raft != nil && m.raft.State() == raft.Leader && m.ready.Load() {
		state = "ready"
	}
	value := domain.CoordinatorAdvertisementV1{SchemaVersion: 1, CellID: m.cfg.Identity.CellID, NodeID: m.cfg.Identity.NodeID, Term: snapshot.Term, AuthorityEpoch: snapshot.AuthorityEpoch, ManagementURL: "https://" + m.cfg.ManagementAddress, CommitIndex: snapshot.CommitIndex, IssuedAt: nowUTC(), ExpiresAt: nowUTC().Add(5 * time.Second), State: state}
	payload, _ := advertisementPayload(value)
	value.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(m.signKey, payload))
	return value, nil
}

func advertisementPayload(value domain.CoordinatorAdvertisementV1) ([]byte, error) {
	value.Signature = ""
	return json.Marshal(value)
}

func (m *Manager) Snapshot() domain.CoordinationCellSnapshotV1 {
	epoch, stateVersion, stateHash, latest := m.fsm.summary()
	value := domain.CoordinationCellSnapshotV1{SchemaVersion: 1, CellID: m.cfg.Identity.CellID, ClusterID: m.cfg.Manifest.ClusterID, Mode: string(m.cfg.Mode), LocalNodeID: m.cfg.Identity.NodeID, State: "disabled", AuthorityEpoch: epoch, StateVersion: stateVersion, StateHash: stateHash, QuorumRequired: m.cfg.Manifest.Quorum, LatestReceipt: latest, UpdatedAt: nowUTC(), ElectionCount: m.elections.Load(), LastElectionMS: m.lastElectionMS.Load()}
	if m.raft == nil {
		return value
	}
	stats := m.raft.Stats()
	leaderAddress, leaderID := m.raft.LeaderWithID()
	value.LeaderAddress = string(leaderAddress)
	value.LeaderNodeID = string(leaderID)
	value.Term = parseUint(stats["term"])
	value.CommitIndex = parseUint(stats["commit_index"])
	value.AppliedIndex = parseUint(stats["applied_index"])
	value.LastLogIndex = parseUint(stats["last_log_index"])
	value.ReachableVoters = 1
	value.State = strings.ToLower(m.raft.State().String())
	if value.State == "leader" && !m.ready.Load() {
		value.State = "electing"
	} else if value.State == "leader" {
		value.ReachableVoters = m.cfg.Manifest.Quorum
	} else if value.LeaderNodeID != "" && !m.raft.LastContact().IsZero() {
		value.ReachableVoters = 2
	}
	for _, member := range m.cfg.Manifest.Members {
		role := "follower"
		if member.NodeID == value.LeaderNodeID {
			role = "leader"
		}
		reachable := member.NodeID == m.cfg.Identity.NodeID || (member.NodeID == value.LeaderNodeID && !m.raft.LastContact().IsZero())
		value.Peers = append(value.Peers, domain.CoordinationPeerV1{NodeID: member.NodeID, Host: member.Host, Role: role, Reachable: reachable, AppliedIndex: value.AppliedIndex})
	}
	return value
}

func (m *Manager) Close(ctx context.Context) error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}
	if m.server != nil {
		_ = m.server.Shutdown(ctx)
	}
	if m.tracer != nil {
		m.tracer.Close(ctx)
	}
	if m.execution != nil {
		_ = m.execution.Close()
	}
	if m.raft != nil {
		err := m.raft.Shutdown().Error()
		if m.store != nil {
			if closeErr := m.store.Close(); err == nil {
				err = closeErr
			}
		}
		return err
	}
	return nil
}

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}
