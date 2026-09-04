package api

import (
	"net/http"
	"strings"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (s *Server) coordinationCellsV6(w http.ResponseWriter, r *http.Request) {
	if s.coordGateway != nil {
		writeJSON(w, http.StatusOK, map[string]any{"cells": s.coordGateway.Cells(r.Context())})
		return
	}
	if s.coordination != nil {
		value := s.coordination.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{"cells": map[string]any{value.CellID: []domain.CoordinationCellSnapshotV1{value}}})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "QUORUM_UNAVAILABLE", Message: "Coordination runtime is unavailable."})
}

func (s *Server) coordinationCellV6(w http.ResponseWriter, r *http.Request) {
	cellID := strings.ToUpper(r.PathValue("id"))
	if s.coordGateway != nil {
		cells := s.coordGateway.Cells(r.Context())
		if values, ok := cells[cellID]; ok {
			writeJSON(w, http.StatusOK, map[string]any{"cell": cellID, "nodes": values})
			return
		}
	}
	if s.coordination != nil && s.coordination.Snapshot().CellID == cellID {
		writeJSON(w, http.StatusOK, s.coordination.Snapshot())
		return
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "CELL_MEMBERSHIP_DENIED", Message: "Coordination cell not found."})
}

func (s *Server) coordinationLogV6(w http.ResponseWriter, r *http.Request) {
	cellID := strings.ToUpper(r.PathValue("id"))
	if s.coordGateway != nil {
		values, err := s.coordGateway.CellLog(r.Context(), cellID)
		if err != nil {
			writeCoordinationPublicError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cell": cellID, "receipts": values})
		return
	}
	if s.coordination != nil && s.coordination.Snapshot().CellID == cellID {
		writeJSON(w, http.StatusOK, map[string]any{"cell": cellID, "receipts": s.coordination.Receipts(200)})
		return
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "CELL_MEMBERSHIP_DENIED", Message: "Coordination cell not found."})
}

func (s *Server) coordinationProofV6(w http.ResponseWriter, r *http.Request) {
	if s.coordGateway != nil {
		if value, ok := s.coordGateway.Proof(r.PathValue("id")); ok {
			writeJSON(w, http.StatusOK, value)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "COMMIT_PROOF_INVALID", Message: "Quorum proof not found."})
}

func (s *Server) crossCellV6(w http.ResponseWriter, r *http.Request) {
	if s.coordGateway != nil {
		if value, ok := s.coordGateway.CrossCellOperation(r.PathValue("id")); ok {
			writeJSON(w, http.StatusOK, value)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, domain.APIError{Code: "CROSS_CELL_PREPARE_FAILED", Message: "Cross-cell operation not found."})
}

func (s *Server) coordinationSecurityV6(w http.ResponseWriter, _ *http.Request) {
	if s.coordGateway != nil {
		writeJSON(w, http.StatusOK, s.coordGateway.SecuritySnapshot())
		return
	}
	if s.coordination != nil {
		snapshot := s.coordination.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{"mode": snapshot.Mode, "transport": "mTLS 1.3 / Ed25519", "cell": snapshot.CellID, "node": snapshot.LocalNodeID})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "QUORUM_UNAVAILABLE", Message: "Coordination runtime is unavailable."})
}

func writeCoordinationPublicError(w http.ResponseWriter, err error) {
	code := "QUORUM_UNAVAILABLE"
	if before, _, ok := strings.Cut(err.Error(), ":"); ok && before != "" && !strings.Contains(before, " ") {
		code = before
	}
	writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: code, Message: err.Error()})
}
