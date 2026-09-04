package coordination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
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
	m.server = &http.Server{Handler: m.InternalHandler(), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 8 * time.Second, WriteTimeout: 8 * time.Second, IdleTimeout: 30 * time.Second}
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
	return mux
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
