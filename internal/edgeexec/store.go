package edgeexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/trajectory"
	"go.etcd.io/bbolt"
)

var (
	programBucket        = []byte("programs-v2")
	adaptationBucket     = []byte("adaptations-v1")
	reconciliationBucket = []byte("reconciliation-v1")
	decisionEpochBucket  = []byte("decision-epochs-v1")
)

type Store struct {
	db     *bbolt.DB
	nodeID string
}

func Open(path, nodeID string) (*Store, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	if err = db.Update(func(tx *bbolt.Tx) error {
		for _, name := range [][]byte{programBucket, adaptationBucket, reconciliationBucket, decisionEpochBucket} {
			if _, createErr := tx.CreateBucketIfNotExists(name); createErr != nil {
				return createErr
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, nodeID: nodeID}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	_ = s.db.Sync()
	return s.db.Close()
}

func (s *Store) Install(program domain.TrajectoryProgramV2) (domain.ProgramInstallReceiptV1, error) {
	if err := trajectory.ValidateFullProgramContent(program); err != nil {
		return domain.ProgramInstallReceiptV1{}, err
	}
	existing, exists, err := s.Program(program.ProgramID)
	if err != nil {
		return domain.ProgramInstallReceiptV1{}, err
	}
	if exists {
		existingRevision := trajectory.ProgramInstallRevision(existing)
		incomingRevision := trajectory.ProgramInstallRevision(program)
		if incomingRevision < existingRevision {
			return domain.ProgramInstallReceiptV1{}, fmt.Errorf("PROGRAM_REVISION_STALE: node has revision %d", existingRevision)
		}
		if incomingRevision == existingRevision && existing.ContentHash != program.ContentHash {
			return domain.ProgramInstallReceiptV1{}, fmt.Errorf("PROGRAM_HASH_MISMATCH: program ID is already bound to different content")
		}
		if existing.ContentHash == program.ContentHash {
			return receipt(s.nodeID, existing, "already_installed"), nil
		}
	}
	for _, candidate := range s.programsForMission(program.MissionID) {
		if trajectory.ProgramInstallRevision(candidate) > trajectory.ProgramInstallRevision(program) {
			return domain.ProgramInstallReceiptV1{}, fmt.Errorf("PROGRAM_REVISION_STALE: node has revision %d", trajectory.ProgramInstallRevision(candidate))
		}
	}
	raw, err := json.Marshal(program)
	if err != nil {
		return domain.ProgramInstallReceiptV1{}, err
	}
	if err = s.db.Update(func(tx *bbolt.Tx) error { return tx.Bucket(programBucket).Put([]byte(program.ProgramID), raw) }); err != nil {
		return domain.ProgramInstallReceiptV1{}, err
	}
	return receipt(s.nodeID, program, "installed"), nil
}

func (s *Store) Program(id string) (domain.TrajectoryProgramV2, bool, error) {
	var value domain.TrajectoryProgramV2
	err := s.db.View(func(tx *bbolt.Tx) error {
		raw := tx.Bucket(programBucket).Get([]byte(id))
		if raw == nil {
			return nil
		}
		return json.Unmarshal(append([]byte(nil), raw...), &value)
	})
	return value, value.ProgramID != "", err
}

func (s *Store) Adapt(adaptation domain.GroupAdaptationV1) error {
	if err := ValidateAdaptationContent(adaptation); err != nil {
		return err
	}
	program, ok, err := s.Program(adaptation.ProgramID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("PROGRAM_NOT_INSTALLED: %s", adaptation.ProgramID)
	}
	if adaptation.MissionID != program.MissionID || adaptation.Tick >= program.AuthorizationExpiryTick {
		return fmt.Errorf("PROGRAM_EXPIRED: adaptation is outside active mission authority")
	}
	if !adaptation.InsideEnvelope || adaptation.SpeedFactor < 0 || adaptation.SpeedFactor > 1.25 || abs(adaptation.HeadingDelta) > 45 || abs(adaptation.LateralOffsetM) > 250 {
		return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: deterministic limits rejected the adaptation")
	}
	if adaptation.DecisionScope != "group" && adaptation.DecisionScope != "local" {
		return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: invalid decision scope")
	}
	if _, assigned := program.Cursors[adaptation.VesselID]; !assigned {
		return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: vessel is not assigned to this program")
	}
	if adaptation.DecisionScope == "local" && adaptation.DecisionNodeID != adaptation.VesselID {
		return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: local decisions must be signed by the affected vessel")
	}
	if adaptation.DecisionScope == "group" {
		if _, assigned := program.Cursors[adaptation.DecisionNodeID]; !assigned {
			return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: group decision node is not assigned to this program")
		}
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		epochs := tx.Bucket(decisionEpochBucket)
		key := []byte(program.MissionID)
		var current int64
		if raw := epochs.Get(key); raw != nil {
			_ = json.Unmarshal(raw, &current)
		}
		if adaptation.DecisionEpoch < current {
			return fmt.Errorf("DECISION_EPOCH_STALE: have %d received %d", current, adaptation.DecisionEpoch)
		}
		if prior := tx.Bucket(adaptationBucket).Get([]byte(adaptation.AdaptationID)); prior != nil {
			var existing domain.GroupAdaptationV1
			if json.Unmarshal(prior, &existing) != nil || existing.ContentHash != adaptation.ContentHash {
				return fmt.Errorf("PROGRAM_HASH_MISMATCH: adaptation ID is already bound to different content")
			}
			return nil
		}
		epochRaw, _ := json.Marshal(adaptation.DecisionEpoch)
		adaptationRaw, _ := json.Marshal(adaptation)
		if err := epochs.Put(key, epochRaw); err != nil {
			return err
		}
		return tx.Bucket(adaptationBucket).Put([]byte(adaptation.AdaptationID), adaptationRaw)
	})
}

// ValidateAdaptationContent binds every operational change to the hash signed
// by the elected group node or isolated vessel. Transport identity alone is
// insufficient because a modified payload must fail before execution.
func ValidateAdaptationContent(adaptation domain.GroupAdaptationV1) error {
	if adaptation.AdaptationID == "" || adaptation.ProgramID == "" || adaptation.MissionID == "" || adaptation.VesselID == "" || adaptation.DecisionNodeID == "" || adaptation.ContentHash == "" {
		return fmt.Errorf("ADAPTATION_OUTSIDE_ENVELOPE: adaptation identity is incomplete")
	}
	unsigned := adaptation
	unsigned.ContentHash = ""
	unsigned.Signature = ""
	raw, err := json.Marshal(unsigned)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	expected := "sha256:" + hex.EncodeToString(sum[:])
	if adaptation.ContentHash != expected {
		return fmt.Errorf("PROGRAM_HASH_MISMATCH: adaptation content does not match its signed hash")
	}
	return nil
}

func (s *Store) Reconcile(value domain.ExecutionReconciliationV1) error {
	program, ok, err := s.Program(value.ProgramID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("PROGRAM_NOT_INSTALLED: %s", value.ProgramID)
	}
	if value.ProgramRevision < program.ActiveRevision {
		return fmt.Errorf("PROGRAM_REVISION_STALE: have %d received %d", program.ActiveRevision, value.ProgramRevision)
	}
	value.ReconciledAt = time.Now().UTC()
	raw, _ := json.Marshal(value)
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(reconciliationBucket).Put([]byte(value.NodeID+":"+value.ProgramID), raw)
	})
}

func ElectDecisionNode(reachableDecisionCapable []string) string {
	values := append([]string(nil), reachableDecisionCapable...)
	sort.Strings(values)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s *Store) programsForMission(missionID string) []domain.TrajectoryProgramV2 {
	values := []domain.TrajectoryProgramV2{}
	_ = s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(programBucket).ForEach(func(_, raw []byte) error {
			var value domain.TrajectoryProgramV2
			if json.Unmarshal(raw, &value) == nil && value.MissionID == missionID {
				values = append(values, value)
			}
			return nil
		})
	})
	return values
}

func receipt(nodeID string, program domain.TrajectoryProgramV2, state string) domain.ProgramInstallReceiptV1 {
	value := domain.ProgramInstallReceiptV1{SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: program.MissionID, NodeID: nodeID, Revision: trajectory.ProgramInstallRevision(program), ProgramHash: program.ContentHash, InstalledAt: time.Now().UTC(), State: state}
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	value.ReceiptHash = "sha256:" + hex.EncodeToString(sum[:])
	return value
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
