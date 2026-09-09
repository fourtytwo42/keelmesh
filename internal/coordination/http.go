package coordination

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/edgeexec"
)

const maxCoordinationBody = 1 << 20

func (m *Manager) StartManagement(ctx context.Context) error {
	if m.cfg.Mode == ModeSimulated || m.cfg.Identity.NodeID == "" {
		return nil
	}
	serverTLS, clientTLS, err := loadNodeTLSConfigs(m.cfg.Identity, m.cfg.Manifest, m.cfg.ManagementCertificateFile, m.cfg.ManagementTLSKeyFile, m.cfg.TrustBundleFile, managementPlane, true)
	if err != nil {
		return err
	}
	m.managementTLS = newTLSConfigSwitcher(serverTLS, serverTLS)
	m.client = &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS, MaxIdleConns: 12, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}, Timeout: m.cfg.ApplyTimeout + 2*time.Second}
	listener, err := listenTLS(ctx, m.cfg.ManagementAddress, m.managementTLS.serverConfig())
	if err != nil {
		return err
	}
	handler := m.InternalHandler()
	if m.tracer != nil {
		handler = m.tracer.Middleware(handler)
	}
	m.server = &http.Server{Handler: handler, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 8 * time.Second, WriteTimeout: 8 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		if serveErr := m.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			m.logger.Error("coordination management server failed", "error", serveErr)
		}
	}()
	return nil
}

func (m *Manager) InternalHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/coordination/commands:propose", m.handlePropose)
	mux.HandleFunc("GET /internal/v1/coordination/status", func(w http.ResponseWriter, _ *http.Request) { writeCoordinationJSON(w, http.StatusOK, m.Snapshot()) })
	mux.HandleFunc("GET /internal/v1/coordination/log", func(w http.ResponseWriter, _ *http.Request) {
		writeCoordinationJSON(w, http.StatusOK, map[string]any{"receipts": m.Receipts(200)})
	})
	mux.HandleFunc("GET /internal/v1/coordination/advertisement", m.handleAdvertisement)
	mux.HandleFunc("GET /internal/v1/coordination/proofs/{command_id}", m.handleAcknowledgement)
	mux.HandleFunc("POST /internal/v1/coordination/cross-cell:prepare", m.handleCrossCell("cross_cell.prepare"))
	mux.HandleFunc("POST /internal/v1/coordination/cross-cell:certify", m.handleCrossCell("cross_cell.certify"))
	mux.HandleFunc("POST /internal/v1/coordination/cross-cell:abort", m.handleCrossCell("cross_cell.abort"))
	mux.HandleFunc("POST /internal/v1/execution/programs:install", m.handleProgramInstall)
	mux.HandleFunc("GET /internal/v1/execution/programs/{id}", m.handleProgramGet)
	mux.HandleFunc("POST /internal/v1/execution/adaptations", m.handleAdaptation)
	mux.HandleFunc("POST /internal/v1/execution/reconcile", m.handleExecutionReconciliation)
	return mux
}

type programInstallRequestV1 struct {
	Program domain.TrajectoryProgramV2 `json:"program"`
	Proof   domain.QuorumCommitProofV1 `json:"proof"`
}

func (m *Manager) handleProgramInstall(w http.ResponseWriter, r *http.Request) {
	if m.execution == nil {
		respondCoordination(w, nil, fmt.Errorf("PROGRAM_NOT_INSTALLED: node execution store is unavailable"), http.StatusCreated)
		return
	}
	var request programInstallRequestV1
	if !decodeExecution(w, r, &request) {
		return
	}
	if err := m.verifyProgramProof(request.Program, request.Proof); err != nil {
		respondCoordination(w, nil, err, http.StatusCreated)
		return
	}
	receipt, err := m.execution.Install(request.Program)
	if err == nil {
		payload := receipt
		payload.Signature = ""
		raw, _ := json.Marshal(payload)
		receipt.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(m.signKey, raw))
	}
	respondCoordination(w, receipt, err, http.StatusCreated)
}

func (m *Manager) handleProgramGet(w http.ResponseWriter, r *http.Request) {
	if m.execution == nil {
		respondCoordination(w, nil, fmt.Errorf("PROGRAM_NOT_INSTALLED: node execution store is unavailable"), http.StatusOK)
		return
	}
	program, ok, err := m.execution.Program(r.PathValue("id"))
	if err == nil && !ok {
		err = fmt.Errorf("PROGRAM_NOT_INSTALLED: %s", r.PathValue("id"))
	}
	respondCoordination(w, program, err, http.StatusOK)
}

func (m *Manager) handleAdaptation(w http.ResponseWriter, r *http.Request) {
	if m.execution == nil {
		respondCoordination(w, nil, fmt.Errorf("PROGRAM_NOT_INSTALLED: node execution store is unavailable"), http.StatusCreated)
		return
	}
	var value domain.GroupAdaptationV1
	if !decodeExecution(w, r, &value) {
		return
	}
	if err := m.verifyAdaptation(value); err != nil {
		respondCoordination(w, nil, err, http.StatusCreated)
		return
	}
	err := m.execution.Adapt(value)
	respondCoordination(w, map[string]any{"adaptation_id": value.AdaptationID, "state": "accepted"}, err, http.StatusCreated)
}

func (m *Manager) handleExecutionReconciliation(w http.ResponseWriter, r *http.Request) {
	if m.execution == nil {
		respondCoordination(w, nil, fmt.Errorf("PROGRAM_NOT_INSTALLED: node execution store is unavailable"), http.StatusCreated)
		return
	}
	var value domain.ExecutionReconciliationV1
	if !decodeExecution(w, r, &value) {
		return
	}
	err := m.execution.Reconcile(value)
	respondCoordination(w, value, err, http.StatusCreated)
}

func (m *Manager) verifyProgramProof(program domain.TrajectoryProgramV2, proof domain.QuorumCommitProofV1) error {
	raw, _ := json.Marshal(program)
	digest := sha256.Sum256(raw)
	if proof.CellID != m.cfg.Identity.CellID || proof.State != "verified" || proof.CommandHash != hex.EncodeToString(digest[:]) || proof.Required != m.cfg.Manifest.Quorum {
		return fmt.Errorf("COMMIT_PROOF_INVALID: proof does not bind this program and cell")
	}
	receipt := domain.AppliedCommandReceiptV1{CommandID: proof.CommandID, CellID: proof.CellID, Term: proof.Term, LogIndex: proof.LogIndex, AuthorityEpoch: proof.AuthorityEpoch, CommandHash: proof.CommandHash, ResultingStateHash: proof.ResultingStateHash}
	seen := map[string]bool{}
	for _, acknowledgement := range proof.Acknowledgements {
		if seen[acknowledgement.NodeID] || verifyAcknowledgement(m.cfg.Manifest, receipt, acknowledgement) != nil {
			return fmt.Errorf("COMMIT_PROOF_INVALID: duplicate or invalid acknowledgement")
		}
		seen[acknowledgement.NodeID] = true
	}
	if len(seen) < m.cfg.Manifest.Quorum {
		return fmt.Errorf("COMMIT_PROOF_INVALID: need %d valid signatures", m.cfg.Manifest.Quorum)
	}
	return nil
}

func (m *Manager) verifyAdaptation(value domain.GroupAdaptationV1) error {
	if err := edgeexec.ValidateAdaptationContent(value); err != nil {
		return err
	}
	payload := value
	payload.Signature = ""
	raw, _ := json.Marshal(payload)
	signature, err := base64.StdEncoding.DecodeString(value.Signature)
	if err != nil {
		return fmt.Errorf("PEER_IDENTITY_INVALID: adaptation signature is invalid")
	}
	for _, member := range m.cfg.Manifest.Members {
		if member.NodeID != value.DecisionNodeID {
			continue
		}
		key, keyErr := decodePublicKey(member.SigningPublicKey)
		if keyErr != nil || !ed25519.Verify(key, raw, signature) {
			return fmt.Errorf("PEER_IDENTITY_INVALID: adaptation signature is invalid")
		}
		return nil
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: adaptation signer is not in this cell")
}

func decodeExecution(w http.ResponseWriter, r *http.Request, destination any) bool {
	body := http.MaxBytesReader(w, r.Body, 32<<20)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeCoordinationJSON(w, http.StatusBadRequest, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: err.Error()})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeCoordinationJSON(w, http.StatusBadRequest, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: "only one JSON object is permitted"})
		return false
	}
	return true
}

func (m *Manager) handlePropose(w http.ResponseWriter, r *http.Request) {
	var command domain.ReplicatedCommandV1
	if !decodeCoordination(w, r, &command) {
		return
	}
	receipt, err := m.ProposeOrForward(r.Context(), command)
	respondCoordination(w, receipt, err, http.StatusCreated)
}

func (m *Manager) handleCrossCell(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var command domain.ReplicatedCommandV1
		if !decodeCoordination(w, r, &command) {
			return
		}
		command.Kind = kind
		receipt, err := m.Propose(r.Context(), command)
		respondCoordination(w, receipt, err, http.StatusCreated)
	}
}

func (m *Manager) handleAdvertisement(w http.ResponseWriter, _ *http.Request) {
	value, err := m.Advertisement()
	respondCoordination(w, value, err, http.StatusOK)
}

func (m *Manager) handleAcknowledgement(w http.ResponseWriter, r *http.Request) {
	value, err := m.Acknowledgement(r.PathValue("command_id"))
	respondCoordination(w, value, err, http.StatusOK)
}

func decodeCoordination(w http.ResponseWriter, r *http.Request, destination any) bool {
	body := http.MaxBytesReader(w, r.Body, maxCoordinationBody)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeCoordinationJSON(w, http.StatusBadRequest, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: err.Error()})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeCoordinationJSON(w, http.StatusBadRequest, domain.APIError{Code: "TOOL_ARGUMENT_INVALID", Message: "only one JSON object is permitted"})
		return false
	}
	return true
}

func respondCoordination(w http.ResponseWriter, value any, err error, success int) {
	if err == nil {
		writeCoordinationJSON(w, success, value)
		return
	}
	code := coordinationErrorCode(err)
	status := http.StatusUnprocessableEntity
	if code == "NOT_COORDINATOR" {
		status = http.StatusTemporaryRedirect
	} else if code == "QUORUM_UNAVAILABLE" || code == "LEADER_NOT_READY" || code == "RAFT_APPLY_TIMEOUT" {
		status = http.StatusServiceUnavailable
	} else if strings.Contains(code, "IDENTITY") || strings.Contains(code, "MEMBERSHIP") || strings.Contains(code, "CERTIFICATE") {
		status = http.StatusForbidden
	}
	writeCoordinationJSON(w, status, domain.APIError{Code: code, Message: err.Error()})
}

func coordinationErrorCode(err error) string {
	message := err.Error()
	if before, _, ok := strings.Cut(message, ":"); ok && before != "" && !strings.Contains(before, " ") {
		return before
	}
	return "INTERNAL"
}

func writeCoordinationJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		_ = fmt.Errorf("encode coordination response: %w", err)
	}
}
