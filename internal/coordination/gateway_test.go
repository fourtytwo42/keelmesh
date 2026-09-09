package coordination

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestAcceptEffectRequiresFourDistinctValidSigners(t *testing.T) {
	manifest := domain.CoordinationCellManifestV1{SchemaVersion: 1, CellID: "A", ClusterID: "cell-a", Quorum: 4}
	privateKeys := map[string]ed25519.PrivateKey{}
	for index := 1; index <= 6; index++ {
		nodeID := fmt.Sprintf("node-a-%02d", index)
		publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
		privateKeys[nodeID] = privateKey
		manifest.Members = append(manifest.Members, domain.CoordinationCellMemberV1{NodeID: nodeID, Faction: "A", SigningPublicKey: base64.StdEncoding.EncodeToString(publicKey)})
	}
	receipt := domain.AppliedCommandReceiptV1{CommandID: "command-1", CellID: "A", Term: 2, LogIndex: 9, AuthorityEpoch: 1, CommandHash: "command-hash", ResultingStateHash: "state-hash"}
	proof := domain.QuorumCommitProofV1{SchemaVersion: 1, CommandID: receipt.CommandID, CellID: receipt.CellID, Term: receipt.Term, LogIndex: receipt.LogIndex, AuthorityEpoch: receipt.AuthorityEpoch, CommandHash: receipt.CommandHash, ResultingStateHash: receipt.ResultingStateHash, Required: 4, State: "verified"}
	for index := 1; index <= 4; index++ {
		nodeID := fmt.Sprintf("node-a-%02d", index)
		acknowledgement := domain.SignedNodeAcknowledgementV1{NodeID: nodeID, CellID: receipt.CellID, Term: receipt.Term, LogIndex: receipt.LogIndex, AuthorityEpoch: receipt.AuthorityEpoch, CommandHash: receipt.CommandHash, ResultingStateHash: receipt.ResultingStateHash}
		payload, _ := acknowledgementPayload(acknowledgement)
		acknowledgement.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKeys[nodeID], payload))
		proof.Acknowledgements = append(proof.Acknowledgements, acknowledgement)
	}
	gateway := &Gateway{cfg: GatewayConfig{Manifests: map[string]domain.CoordinationCellManifestV1{"A": manifest}}, proofs: map[string]domain.QuorumCommitProofV1{}, crossCell: map[string]domain.CrossCellOperationV1{}, applied: map[string]bool{}}
	if err := gateway.AcceptEffect(proof); err != nil {
		t.Fatal(err)
	}
	duplicate := proof
	duplicate.Acknowledgements = append([]domain.SignedNodeAcknowledgementV1(nil), proof.Acknowledgements...)
	duplicate.Acknowledgements[3] = duplicate.Acknowledgements[0]
	if err := gateway.AcceptEffect(duplicate); err == nil {
		t.Fatal("duplicate signer satisfied a four-vote proof")
	}
	tampered := proof
	tampered.Acknowledgements = append([]domain.SignedNodeAcknowledgementV1(nil), proof.Acknowledgements...)
	tampered.Acknowledgements[0].ResultingStateHash = "tampered"
	if err := gateway.AcceptEffect(tampered); err == nil {
		t.Fatal("tampered acknowledgement was accepted")
	}
}

func TestProgramInstallReceiptBindsNodeProgramAndSignature(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	manifest := domain.CoordinationCellManifestV1{SchemaVersion: 1, CellID: "A", Quorum: 4, Members: []domain.CoordinationCellMemberV1{{NodeID: "node-a-01", Faction: "A", SigningPublicKey: base64.StdEncoding.EncodeToString(publicKey)}}}
	program := domain.TrajectoryProgramV2{SchemaVersion: 2, ProgramID: "program-1", MissionID: "mission-1", ActiveRevision: 1, ContentHash: "sha256:program"}
	receipt := domain.ProgramInstallReceiptV1{SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: program.MissionID, NodeID: "node-a-01", Revision: 1, ProgramHash: program.ContentHash, InstalledAt: time.Unix(100, 0).UTC(), State: "installed"}
	hashPayload := receipt
	raw, _ := json.Marshal(hashPayload)
	digest := sha256.Sum256(raw)
	receipt.ReceiptHash = "sha256:" + hex.EncodeToString(digest[:])
	signedRaw, _ := json.Marshal(receipt)
	receipt.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, signedRaw))
	if err := verifyProgramInstallReceipt(manifest, "node-a-01", program, receipt); err != nil {
		t.Fatal(err)
	}
	tampered := receipt
	tampered.ProgramHash = "sha256:tampered"
	if err := verifyProgramInstallReceipt(manifest, "node-a-01", program, tampered); err == nil {
		t.Fatal("tampered install receipt was accepted")
	}
}
