package coordination

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (m *Manager) validateCrossCellCommand(command domain.ReplicatedCommandV1) error {
	if command.Kind != "cross_cell.certify" {
		return nil
	}
	if command.ActorIdentity != "referee-214" || command.ParentOperationID == "" || command.FutureActivationAt == nil {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: certificate command is incomplete")
	}
	var certificate domain.CrossCellActivationCertificateV1
	if err := json.Unmarshal(command.Payload, &certificate); err != nil {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: decode certificate: %w", err)
	}
	if certificate.SchemaVersion != 1 || certificate.Issuer != "referee-214" || certificate.OperationID != command.ParentOperationID || certificate.ActivationAt.IsZero() || !certificate.ActivationAt.Equal(command.FutureActivationAt.UTC()) {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: certificate binding mismatch")
	}
	if certificate.ActivationAt.Before(command.IssuedAt.UTC()) || certificate.ActivationAt.Before(time.Now().UTC().Add(-time.Second)) {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: activation tick is stale")
	}
	signature, err := base64.StdEncoding.DecodeString(certificate.Signature)
	if err != nil {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: invalid referee signature encoding")
	}
	payload, err := crossCellCertificatePayload(certificate)
	if err != nil || len(m.refereeKey) != ed25519.PublicKeySize || !ed25519.Verify(m.refereeKey, payload, signature) {
		return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: invalid referee signature")
	}
	for _, cellID := range []string{"A", "B"} {
		proof, ok := certificate.PrepareProofs[cellID]
		if !ok || proof.CellID != cellID || proof.State != "verified" || proof.Required != 4 || len(proof.Acknowledgements) < 4 || proof.CommandHash != certificate.CommandHash {
			return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: invalid %s prepare proof", cellID)
		}
		if cellID != m.cfg.Identity.CellID {
			continue
		}
		seen := map[string]bool{}
		for _, acknowledgement := range proof.Acknowledgements {
			if seen[acknowledgement.NodeID] {
				return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: duplicate %s proof signer", cellID)
			}
			seen[acknowledgement.NodeID] = true
			receipt := domain.AppliedCommandReceiptV1{CommandID: proof.CommandID, CellID: proof.CellID, Term: proof.Term, LogIndex: proof.LogIndex, AuthorityEpoch: proof.AuthorityEpoch, CommandHash: proof.CommandHash, ResultingStateHash: proof.ResultingStateHash}
			if err := verifyAcknowledgement(m.cfg.Manifest, receipt, acknowledgement); err != nil {
				return fmt.Errorf("CROSS_CELL_CERTIFICATE_INCOMPLETE: invalid local prepare proof: %w", err)
			}
		}
	}
	return nil
}
