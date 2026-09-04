package coordination

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/hashicorp/raft"
)

type applyResult struct {
	receipt domain.AppliedCommandReceiptV1
	err     error
}

type fsmState struct {
	SchemaVersion  int                                       `json:"schema_version"`
	AuthorityEpoch uint64                                    `json:"authority_epoch"`
	StateVersion   int64                                     `json:"state_version"`
	Receipts       map[string]domain.AppliedCommandReceiptV1 `json:"receipts"`
	Idempotency    map[string]string                         `json:"idempotency"`
	Latest         *domain.AppliedCommandReceiptV1           `json:"latest,omitempty"`
}

type stateMachine struct {
	mu    sync.RWMutex
	state fsmState
}

func newStateMachine() *stateMachine {
	return &stateMachine{state: fsmState{SchemaVersion: 1, AuthorityEpoch: 0, StateVersion: 1, Receipts: map[string]domain.AppliedCommandReceiptV1{}, Idempotency: map[string]string{}}}
}

func (f *stateMachine) Apply(log *raft.Log) interface{} {
	var command domain.ReplicatedCommandV1
	if err := json.Unmarshal(log.Data, &command); err != nil {
		return applyResult{err: fmt.Errorf("decode replicated command: %w", err)}
	}
	if err := validateCommand(command); err != nil {
		return applyResult{err: err}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if existingHash, ok := f.state.Idempotency[command.IdempotencyKey]; ok {
		if existingHash != command.PayloadHash {
			return applyResult{err: fmt.Errorf("IDEMPOTENCY_CONFLICT: key already binds different content")}
		}
		if receipt, found := f.state.Receipts[command.CommandID]; found {
			return applyResult{receipt: receipt}
		}
	}
	if receipt, ok := f.state.Receipts[command.CommandID]; ok {
		return applyResult{receipt: receipt}
	}
	if command.Kind == "coordination.epoch_advance" {
		if command.AuthorityEpoch != f.state.AuthorityEpoch+1 {
			return applyResult{err: fmt.Errorf("AUTHORITY_EPOCH_STALE: expected %d", f.state.AuthorityEpoch+1)}
		}
		f.state.AuthorityEpoch = command.AuthorityEpoch
	} else if command.AuthorityEpoch != f.state.AuthorityEpoch {
		return applyResult{err: fmt.Errorf("AUTHORITY_EPOCH_STALE: command %d current %d", command.AuthorityEpoch, f.state.AuthorityEpoch)}
	}
	f.state.StateVersion++
	stateHash := f.hashLocked(command.CommandID, command.PayloadHash, log.Term, log.Index)
	receipt := domain.AppliedCommandReceiptV1{
		SchemaVersion: 1, CommandID: command.CommandID, CellID: command.CellID,
		Term: log.Term, LogIndex: log.Index, AuthorityEpoch: f.state.AuthorityEpoch,
		CommandHash: command.PayloadHash, ResultingStateHash: stateHash,
		AppliedAt: command.IssuedAt.UTC(), State: "committed",
	}
	f.state.Receipts[command.CommandID] = receipt
	f.state.Idempotency[command.IdempotencyKey] = command.PayloadHash
	f.state.Latest = &receipt
	return applyResult{receipt: receipt}
}

func validateCommand(command domain.ReplicatedCommandV1) error {
	if command.SchemaVersion != 1 || command.CommandID == "" || command.RequestID == "" || command.IdempotencyKey == "" || command.ActorIdentity == "" || command.CellID == "" || command.Kind == "" || command.PayloadHash == "" || command.IssuedAt.IsZero() {
		return fmt.Errorf("TOOL_ARGUMENT_INVALID: incomplete replicated command")
	}
	digest := sha256.Sum256(command.Payload)
	if hex.EncodeToString(digest[:]) != command.PayloadHash {
		return fmt.Errorf("COMMIT_HASH_MISMATCH: payload checksum does not match")
	}
	return nil
}

func (f *stateMachine) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	encoded, err := json.Marshal(f.state)
	if err != nil {
		return nil, err
	}
	return snapshot(encoded), nil
}

func (f *stateMachine) Restore(reader io.ReadCloser) error {
	defer reader.Close()
	var restored fsmState
	if err := json.NewDecoder(reader).Decode(&restored); err != nil {
		return err
	}
	if restored.Receipts == nil {
		restored.Receipts = map[string]domain.AppliedCommandReceiptV1{}
	}
	if restored.Idempotency == nil {
		restored.Idempotency = map[string]string{}
	}
	f.mu.Lock()
	f.state = restored
	f.mu.Unlock()
	return nil
}

func (f *stateMachine) receipt(id string) (domain.AppliedCommandReceiptV1, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	receipt, ok := f.state.Receipts[id]
	return receipt, ok
}

func (f *stateMachine) summary() (uint64, int64, string, *domain.AppliedCommandReceiptV1) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state.AuthorityEpoch, f.state.StateVersion, f.hashLocked("", "", 0, 0), cloneReceipt(f.state.Latest)
}

func cloneReceipt(value *domain.AppliedCommandReceiptV1) *domain.AppliedCommandReceiptV1 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (f *stateMachine) hashLocked(commandID, commandHash string, term, index uint64) string {
	ids := make([]string, 0, len(f.state.Receipts))
	for id := range f.state.Receipts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	view := struct {
		Epoch       uint64   `json:"epoch"`
		Version     int64    `json:"version"`
		ReceiptIDs  []string `json:"receipt_ids"`
		CommandID   string   `json:"command_id,omitempty"`
		CommandHash string   `json:"command_hash,omitempty"`
		Term        uint64   `json:"term,omitempty"`
		Index       uint64   `json:"index,omitempty"`
	}{f.state.AuthorityEpoch, f.state.StateVersion, ids, commandID, commandHash, term, index}
	encoded, _ := json.Marshal(view)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

type snapshot []byte

func (s snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := io.Copy(sink, bytes.NewReader(s)); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (snapshot) Release() {}

func canonicalPayload(value any) (json.RawMessage, string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(encoded)
	return encoded, hex.EncodeToString(digest[:]), nil
}

func nowUTC() time.Time { return time.Now().UTC() }
