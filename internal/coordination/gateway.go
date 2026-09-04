package coordination

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type GatewayConfig struct {
	Mode             Mode
	Manifests        map[string]domain.CoordinationCellManifestV1
	CertificateFile  string
	TLSKeyFile       string
	TrustBundleFile  string
	SigningKeyFile   string
	OperationTimeout time.Duration
	StateFile        string
}

type Gateway struct {
	cfg       GatewayConfig
	client    *http.Client
	signKey   ed25519.PrivateKey
	mu        sync.RWMutex
	proofs    map[string]domain.QuorumCommitProofV1
	crossCell map[string]domain.CrossCellOperationV1
	applied   map[string]bool
}

func NewGateway(cfg GatewayConfig) (*Gateway, error) {
	if cfg.Mode == ModeSimulated {
		return &Gateway{cfg: cfg, proofs: map[string]domain.QuorumCommitProofV1{}, crossCell: map[string]domain.CrossCellOperationV1{}, applied: map[string]bool{}}, nil
	}
	if cfg.OperationTimeout <= 0 {
		cfg.OperationTimeout = 8 * time.Second
	}
	for _, manifest := range cfg.Manifests {
		if err := validateManifest(manifest); err != nil {
			return nil, err
		}
	}
	tlsConfig, err := loadRefereeClientTLS(cfg)
	if err != nil {
		return nil, err
	}
	signKey, err := loadSigningKey(cfg.SigningKeyFile)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig, MaxIdleConns: 24, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}
	gateway := &Gateway{cfg: cfg, client: &http.Client{Transport: transport, Timeout: cfg.OperationTimeout}, signKey: signKey, proofs: map[string]domain.QuorumCommitProofV1{}, crossCell: map[string]domain.CrossCellOperationV1{}, applied: map[string]bool{}}
	if err := gateway.loadState(); err != nil {
		return nil, err
	}
	return gateway, nil
}

func loadRefereeClientTLS(cfg GatewayConfig) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertificateFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	encoded, err := os.ReadFile(cfg.TrustBundleFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(encoded) {
		return nil, fmt.Errorf("referee trust bundle contains no certificates")
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}, RootCAs: roots, NextProtos: []string{"keelmesh-coordination-v1"}, VerifyConnection: func(state tls.ConnectionState) error {
		if len(state.PeerCertificates) == 0 {
			return fmt.Errorf("PEER_IDENTITY_INVALID: missing node certificate")
		}
		leaf := state.PeerCertificates[0]
		nodeID, cellID := identityFromURIs(leaf.URIs)
		manifest, ok := cfg.Manifests[cellID]
		if !ok {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: unknown cell")
		}
		for _, member := range manifest.Members {
			if member.NodeID != nodeID {
				continue
			}
			host, _, _ := net.SplitHostPort(member.ManagementAddress)
			if member.ManagementTLSSerial != "" && !strings.EqualFold(member.ManagementTLSSerial, leaf.SerialNumber.Text(16)) {
				return fmt.Errorf("PEER_IDENTITY_INVALID: TLS serial differs from manifest")
			}
			if !containsIP(leaf.IPAddresses, net.ParseIP(host)) {
				return fmt.Errorf("PEER_IDENTITY_INVALID: management IP SAN differs from manifest")
			}
			return nil
		}
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: node is not a voter")
	}}, nil
}

func (g *Gateway) CanonicalCommand(cellID, commandID, requestID, idempotencyKey, actor, kind, entityID string, expectedVersion int64, payload any, activation *time.Time) (domain.ReplicatedCommandV1, error) {
	encoded, hash, err := canonicalPayload(payload)
	if err != nil {
		return domain.ReplicatedCommandV1{}, err
	}
	return domain.ReplicatedCommandV1{SchemaVersion: 1, CommandID: commandID, RequestID: requestID, IdempotencyKey: idempotencyKey, ActorIdentity: actor, CellID: strings.ToUpper(cellID), Kind: kind, EntityID: entityID, ExpectedVersion: expectedVersion, Payload: encoded, PayloadHash: hash, IssuedAt: nowUTC(), FutureActivationAt: activation}, nil
}

func (g *Gateway) Commit(ctx context.Context, command domain.ReplicatedCommandV1) (domain.AppliedCommandReceiptV1, domain.QuorumCommitProofV1, error) {
	if g.cfg.Mode == ModeSimulated {
		return domain.AppliedCommandReceiptV1{}, domain.QuorumCommitProofV1{}, fmt.Errorf("QUORUM_UNAVAILABLE: gateway is in simulated mode")
	}
	manifest, ok := g.cfg.Manifests[command.CellID]
	if !ok {
		return domain.AppliedCommandReceiptV1{}, domain.QuorumCommitProofV1{}, fmt.Errorf("CELL_MEMBERSHIP_DENIED: unknown cell %s", command.CellID)
	}
	leader, err := g.DiscoverLeader(ctx, command.CellID)
	if err != nil {
		return domain.AppliedCommandReceiptV1{}, domain.QuorumCommitProofV1{}, err
	}
	var receipt domain.AppliedCommandReceiptV1
	if err := g.requestJSON(ctx, http.MethodPost, leader.ManagementURL+"/internal/v1/coordination/commands:propose", command, &receipt); err != nil {
		return receipt, domain.QuorumCommitProofV1{}, err
	}
	proof, err := g.collectProof(ctx, manifest, receipt)
	if err != nil {
		return receipt, proof, err
	}
	g.mu.Lock()
	g.proofs[command.CommandID] = proof
	persistErr := g.persistLocked()
	g.mu.Unlock()
	if persistErr != nil {
		return receipt, proof, persistErr
	}
	return receipt, proof, nil
}

func (g *Gateway) DiscoverLeader(ctx context.Context, cellID string) (domain.CoordinatorAdvertisementV1, error) {
	manifest, ok := g.cfg.Manifests[cellID]
	if !ok {
		return domain.CoordinatorAdvertisementV1{}, fmt.Errorf("CELL_MEMBERSHIP_DENIED: unknown cell")
	}
	var candidates []domain.CoordinatorAdvertisementV1
	for _, member := range manifest.Members {
		var value domain.CoordinatorAdvertisementV1
		url := "https://" + member.ManagementAddress + "/internal/v1/coordination/advertisement"
		if err := g.requestJSON(ctx, http.MethodGet, url, nil, &value); err != nil {
			continue
		}
		if value.CellID != cellID || value.NodeID != member.NodeID || value.State != "ready" || nowUTC().After(value.ExpiresAt) || verifyAdvertisement(manifest, value) != nil {
			continue
		}
		candidates = append(candidates, value)
	}
	if len(candidates) == 0 {
		return domain.CoordinatorAdvertisementV1{}, fmt.Errorf("LEADER_NOT_READY: no valid signed leader advertisement")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Term != candidates[j].Term {
			return candidates[i].Term > candidates[j].Term
		}
		return candidates[i].CommitIndex > candidates[j].CommitIndex
	})
	return candidates[0], nil
}

func verifyAdvertisement(manifest domain.CoordinationCellManifestV1, value domain.CoordinatorAdvertisementV1) error {
	for _, member := range manifest.Members {
		if member.NodeID != value.NodeID {
			continue
		}
		publicKey, err := decodePublicKey(member.SigningPublicKey)
		if err != nil {
			return err
		}
		signature, err := base64.StdEncoding.DecodeString(value.Signature)
		if err != nil {
			return err
		}
		payload, _ := advertisementPayload(value)
		if !ed25519.Verify(publicKey, payload, signature) {
			return fmt.Errorf("PEER_IDENTITY_INVALID: invalid leader advertisement signature")
		}
		return nil
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: advertiser is not a member")
}

func (g *Gateway) collectProof(ctx context.Context, manifest domain.CoordinationCellManifestV1, receipt domain.AppliedCommandReceiptV1) (domain.QuorumCommitProofV1, error) {
	proof := domain.QuorumCommitProofV1{SchemaVersion: 1, CommandID: receipt.CommandID, CellID: receipt.CellID, Term: receipt.Term, LogIndex: receipt.LogIndex, AuthorityEpoch: receipt.AuthorityEpoch, CommandHash: receipt.CommandHash, ResultingStateHash: receipt.ResultingStateHash, Required: manifest.Quorum, State: "collecting"}
	seen := map[string]bool{}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		type acknowledgementResult struct {
			value domain.SignedNodeAcknowledgementV1
			err   error
		}
		results := make(chan acknowledgementResult, len(manifest.Members))
		var wait sync.WaitGroup
		for _, member := range manifest.Members {
			if seen[member.NodeID] {
				continue
			}
			wait.Add(1)
			go func(member domain.CoordinationCellMemberV1) {
				defer wait.Done()
				attemptCtx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
				defer cancel()
				var acknowledgement domain.SignedNodeAcknowledgementV1
				url := "https://" + member.ManagementAddress + "/internal/v1/coordination/proofs/" + receipt.CommandID
				err := g.requestJSON(attemptCtx, http.MethodGet, url, nil, &acknowledgement)
				if err == nil {
					err = verifyAcknowledgement(manifest, receipt, acknowledgement)
				}
				results <- acknowledgementResult{value: acknowledgement, err: err}
			}(member)
		}
		wait.Wait()
		close(results)
		for result := range results {
			if result.err != nil || seen[result.value.NodeID] {
				continue
			}
			seen[result.value.NodeID] = true
			proof.Acknowledgements = append(proof.Acknowledgements, result.value)
		}
		if len(proof.Acknowledgements) >= manifest.Quorum {
			sort.Slice(proof.Acknowledgements, func(i, j int) bool { return proof.Acknowledgements[i].NodeID < proof.Acknowledgements[j].NodeID })
			proof.State = "verified"
			proof.CompletedAt = nowUTC()
			return proof, nil
		}
		select {
		case <-ctx.Done():
			return proof, fmt.Errorf("COMMIT_PROOF_INVALID: only %d of %d acknowledgements were verified", len(proof.Acknowledgements), manifest.Quorum)
		case <-ticker.C:
		}
	}
}

func verifyAcknowledgement(manifest domain.CoordinationCellManifestV1, receipt domain.AppliedCommandReceiptV1, ack domain.SignedNodeAcknowledgementV1) error {
	if ack.CellID != receipt.CellID || ack.Term != receipt.Term || ack.LogIndex != receipt.LogIndex || ack.AuthorityEpoch != receipt.AuthorityEpoch || ack.CommandHash != receipt.CommandHash || ack.ResultingStateHash != receipt.ResultingStateHash {
		return fmt.Errorf("COMMIT_PROOF_INVALID: acknowledgement does not match receipt")
	}
	for _, member := range manifest.Members {
		if member.NodeID != ack.NodeID {
			continue
		}
		publicKey, err := decodePublicKey(member.SigningPublicKey)
		if err != nil {
			return err
		}
		signature, err := base64.StdEncoding.DecodeString(ack.Signature)
		if err != nil {
			return err
		}
		payload, _ := acknowledgementPayload(ack)
		if !ed25519.Verify(publicKey, payload, signature) {
			return fmt.Errorf("COMMIT_PROOF_INVALID: invalid member signature")
		}
		return nil
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: acknowledgement signer is not a member")
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("PEER_IDENTITY_INVALID: invalid application public key")
	}
	return ed25519.PublicKey(decoded), nil
}

func (g *Gateway) requestJSON(ctx context.Context, method, url string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := g.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError domain.APIError
		_ = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&apiError)
		if apiError.Code == "" {
			apiError.Code = "QUORUM_UNAVAILABLE"
		}
		return fmt.Errorf("%s: %s", apiError.Code, apiError.Message)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(output)
}

func (g *Gateway) Proof(commandID string) (domain.QuorumCommitProofV1, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	proof, ok := g.proofs[commandID]
	return proof, ok
}

func (g *Gateway) Mode() Mode { return g.cfg.Mode }

func (g *Gateway) Cells(ctx context.Context) map[string][]domain.CoordinationCellSnapshotV1 {
	result := make(map[string][]domain.CoordinationCellSnapshotV1, len(g.cfg.Manifests))
	for cellID, manifest := range g.cfg.Manifests {
		for _, member := range manifest.Members {
			var snapshot domain.CoordinationCellSnapshotV1
			url := "https://" + member.ManagementAddress + "/internal/v1/coordination/status"
			if g.client == nil || g.requestJSON(ctx, http.MethodGet, url, nil, &snapshot) != nil {
				snapshot = domain.CoordinationCellSnapshotV1{SchemaVersion: 1, CellID: cellID, ClusterID: manifest.ClusterID, Mode: string(g.cfg.Mode), LocalNodeID: member.NodeID, State: "unreachable", QuorumRequired: manifest.Quorum, UpdatedAt: nowUTC()}
			}
			result[cellID] = append(result[cellID], snapshot)
		}
	}
	return result
}

func (g *Gateway) CellLog(ctx context.Context, cellID string) ([]domain.AppliedCommandReceiptV1, error) {
	leader, err := g.DiscoverLeader(ctx, cellID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Receipts []domain.AppliedCommandReceiptV1 `json:"receipts"`
	}
	if err := g.requestJSON(ctx, http.MethodGet, leader.ManagementURL+"/internal/v1/coordination/log", nil, &response); err != nil {
		return nil, err
	}
	return response.Receipts, nil
}

func (g *Gateway) SecuritySnapshot() map[string]any {
	cells := map[string]any{}
	for cellID, manifest := range g.cfg.Manifests {
		members := make([]map[string]any, 0, len(manifest.Members))
		for _, member := range manifest.Members {
			members = append(members, map[string]any{"node_id": member.NodeID, "vm_id": member.VMID, "raft_tls_serial": member.RaftTLSSerial, "management_tls_serial": member.ManagementTLSSerial, "management_address": member.ManagementAddress, "radio_address": member.RadioAddress})
		}
		cells[cellID] = map[string]any{"cluster_id": manifest.ClusterID, "trust_version": manifest.TrustVersion, "manifest_expires_at": manifest.ExpiresAt, "quorum": manifest.Quorum, "members": members, "revoked_serial_count": len(manifest.RevokedSerials)}
	}
	return map[string]any{"mode": g.cfg.Mode, "transport": "mTLS 1.3 / Ed25519", "referee_role": "non-voting", "cells": cells}
}

func (g *Gateway) CrossCell(ctx context.Context, operationID, requestID, idempotencyKey, actor, kind, entityID string, expectedVersion int64, payload any, activation time.Time) (domain.CrossCellOperationV1, error) {
	encoded, hash, err := canonicalPayload(payload)
	if err != nil {
		return domain.CrossCellOperationV1{}, err
	}
	operation := domain.CrossCellOperationV1{SchemaVersion: 1, ID: operationID, CommandHash: hash, State: "preparing", ActivationAt: activation.UTC(), PrepareProofs: map[string]domain.QuorumCommitProofV1{}, CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	if err := g.storeCrossCell(operation); err != nil {
		return operation, err
	}
	prepared := []string{}
	for _, cellID := range []string{"A", "B"} {
		command := domain.ReplicatedCommandV1{SchemaVersion: 1, CommandID: operationID + "-prepare-" + strings.ToLower(cellID), RequestID: requestID, IdempotencyKey: idempotencyKey + ":prepare:" + strings.ToLower(cellID), ActorIdentity: actor, CellID: cellID, Kind: "cross_cell.prepare", EntityID: entityID, ExpectedVersion: expectedVersion, Payload: encoded, PayloadHash: hash, IssuedAt: nowUTC(), FutureActivationAt: &activation, ParentOperationID: operationID}
		_, proof, commitErr := g.Commit(ctx, command)
		if commitErr != nil {
			operation.State = "aborting"
			operation.UpdatedAt = nowUTC()
			for _, preparedCell := range prepared {
				abortPayload, abortHash, _ := canonicalPayload(map[string]any{"operation_id": operationID, "reason": "peer_prepare_failed"})
				abort := domain.ReplicatedCommandV1{SchemaVersion: 1, CommandID: operationID + "-abort-" + strings.ToLower(preparedCell), RequestID: requestID, IdempotencyKey: idempotencyKey + ":abort:" + strings.ToLower(preparedCell), ActorIdentity: actor, CellID: preparedCell, Kind: "cross_cell.abort", EntityID: entityID, Payload: abortPayload, PayloadHash: abortHash, IssuedAt: nowUTC(), ParentOperationID: operationID}
				_, _, _ = g.Commit(ctx, abort)
			}
			_ = g.storeCrossCell(operation)
			return operation, fmt.Errorf("CROSS_CELL_PREPARE_FAILED: cell %s: %w", cellID, commitErr)
		}
		operation.PrepareProofs[cellID] = proof
		prepared = append(prepared, cellID)
	}
	certificate := domain.CrossCellActivationCertificateV1{SchemaVersion: 1, OperationID: operationID, CommandHash: hash, ActivationAt: activation.UTC(), PrepareProofs: operation.PrepareProofs, IssuedAt: nowUTC(), Issuer: "referee-214"}
	certificatePayload, _ := crossCellCertificatePayload(certificate)
	certificate.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(g.signKey, certificatePayload))
	for _, cellID := range []string{"A", "B"} {
		certPayload, certHash, _ := canonicalPayload(certificate)
		command := domain.ReplicatedCommandV1{SchemaVersion: 1, CommandID: operationID + "-certify-" + strings.ToLower(cellID), RequestID: requestID, IdempotencyKey: idempotencyKey + ":certify:" + strings.ToLower(cellID), ActorIdentity: "referee-214", CellID: cellID, Kind: "cross_cell.certify", EntityID: entityID, Payload: certPayload, PayloadHash: certHash, IssuedAt: nowUTC(), FutureActivationAt: &activation, ParentOperationID: operationID}
		_, certificationProof, commitErr := g.Commit(ctx, command)
		if commitErr == nil {
			commitErr = g.AcceptEffect(certificationProof)
		}
		if commitErr != nil {
			operation.State = "prepared"
			operation.UpdatedAt = nowUTC()
			_ = g.storeCrossCell(operation)
			return operation, fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: cell %s: %w", cellID, commitErr)
		}
	}
	operation.Certificate = &certificate
	operation.State = "armed"
	operation.UpdatedAt = nowUTC()
	if err := g.storeCrossCell(operation); err != nil {
		return operation, err
	}
	return operation, nil
}

func crossCellCertificatePayload(certificate domain.CrossCellActivationCertificateV1) ([]byte, error) {
	certificate.Signature = ""
	return json.Marshal(certificate)
}

func (g *Gateway) CrossCellOperation(id string) (domain.CrossCellOperationV1, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	value, ok := g.crossCell[id]
	return value, ok
}

func (g *Gateway) storeCrossCell(operation domain.CrossCellOperationV1) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.crossCell[operation.ID] = operation
	return g.persistLocked()
}

func (g *Gateway) AcceptEffect(proof domain.QuorumCommitProofV1) error {
	manifest, ok := g.cfg.Manifests[proof.CellID]
	if !ok || proof.State != "verified" || len(proof.Acknowledgements) < manifest.Quorum {
		return fmt.Errorf("COMMIT_PROOF_INVALID: incomplete proof")
	}
	seen := map[string]bool{}
	for _, ack := range proof.Acknowledgements {
		if seen[ack.NodeID] {
			return fmt.Errorf("COMMIT_PROOF_INVALID: duplicate acknowledgement signer")
		}
		seen[ack.NodeID] = true
		receipt := domain.AppliedCommandReceiptV1{CommandID: proof.CommandID, CellID: proof.CellID, Term: proof.Term, LogIndex: proof.LogIndex, AuthorityEpoch: proof.AuthorityEpoch, CommandHash: proof.CommandHash, ResultingStateHash: proof.ResultingStateHash}
		if err := verifyAcknowledgement(manifest, receipt, ack); err != nil {
			return err
		}
	}
	key := fmt.Sprintf("%s:%d:%d:%s", proof.CellID, proof.AuthorityEpoch, proof.LogIndex, proof.CommandID)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.applied[key] {
		return nil
	}
	g.applied[key] = true
	return g.persistLocked()
}

type gatewayState struct {
	SchemaVersion int                                    `json:"schema_version"`
	Proofs        map[string]domain.QuorumCommitProofV1  `json:"proofs"`
	CrossCell     map[string]domain.CrossCellOperationV1 `json:"cross_cell"`
	Applied       map[string]bool                        `json:"applied"`
}

func (g *Gateway) loadState() error {
	if g.cfg.StateFile == "" {
		return nil
	}
	encoded, err := os.ReadFile(g.cfg.StateFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load coordination gateway state: %w", err)
	}
	var state gatewayState
	if err := json.Unmarshal(encoded, &state); err != nil || state.SchemaVersion != 1 {
		return fmt.Errorf("load coordination gateway state: invalid durable state")
	}
	if state.Proofs != nil {
		g.proofs = state.Proofs
	}
	if state.CrossCell != nil {
		g.crossCell = state.CrossCell
	}
	if state.Applied != nil {
		g.applied = state.Applied
	}
	return nil
}

func (g *Gateway) persistLocked() error {
	if g.cfg.StateFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(g.cfg.StateFile), 0o700); err != nil {
		return fmt.Errorf("persist coordination gateway state: %w", err)
	}
	state := gatewayState{SchemaVersion: 1, Proofs: g.proofs, CrossCell: g.crossCell, Applied: g.applied}
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	temporary := g.cfg.StateFile + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, g.cfg.StateFile)
}
