package coordination

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/hashicorp/raft"
)

func testCommand(t *testing.T, id, key, kind string, epoch uint64, payload any) domain.ReplicatedCommandV1 {
	t.Helper()
	encoded, hash, err := canonicalPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	return domain.ReplicatedCommandV1{SchemaVersion: 1, CommandID: id, RequestID: "request-" + id, IdempotencyKey: key, ActorIdentity: "operator", CellID: "A", AuthorityEpoch: epoch, Kind: kind, EntityID: "mission-1", Payload: encoded, PayloadHash: hash, IssuedAt: time.Unix(100, 0).UTC()}
}

func applyTestCommand(t *testing.T, fsm *stateMachine, command domain.ReplicatedCommandV1, index uint64) applyResult {
	t.Helper()
	encoded, _ := json.Marshal(command)
	result, ok := fsm.Apply(&raft.Log{Term: 1, Index: index, Data: encoded}).(applyResult)
	if !ok {
		t.Fatal("unexpected FSM response")
	}
	return result
}

func TestFSMIdempotencyAndConflict(t *testing.T) {
	fsm := newStateMachine()
	epoch := testCommand(t, "epoch-1", "epoch-key", "coordination.epoch_advance", 1, map[string]int{"term": 1})
	if result := applyTestCommand(t, fsm, epoch, 1); result.err != nil {
		t.Fatal(result.err)
	}
	command := testCommand(t, "command-1", "stable-key", "mission.create", 1, map[string]string{"name": "Harbor Watch"})
	first := applyTestCommand(t, fsm, command, 2)
	if first.err != nil {
		t.Fatal(first.err)
	}
	duplicate := applyTestCommand(t, fsm, command, 3)
	if duplicate.err != nil || duplicate.receipt.LogIndex != first.receipt.LogIndex {
		t.Fatalf("duplicate was not stable: %#v", duplicate)
	}
	conflict := command
	conflict.CommandID = "command-2"
	conflict.Payload, conflict.PayloadHash, _ = canonicalPayload(map[string]string{"name": "Different"})
	if result := applyTestCommand(t, fsm, conflict, 4); result.err == nil {
		t.Fatal("expected idempotency conflict")
	}
}

func TestFSMSnapshotRestorePreservesStateHash(t *testing.T) {
	fsm := newStateMachine()
	if result := applyTestCommand(t, fsm, testCommand(t, "epoch", "epoch", "coordination.epoch_advance", 1, map[string]int{"term": 1}), 1); result.err != nil {
		t.Fatal(result.err)
	}
	if result := applyTestCommand(t, fsm, testCommand(t, "mission", "mission", "mission.create", 1, map[string]string{"name": "Sound Patrol"}), 2); result.err != nil {
		t.Fatal(result.err)
	}
	_, _, expectedHash, _ := fsm.summary()
	snapshotValue, err := fsm.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	sink := &memorySnapshotSink{}
	if err := snapshotValue.Persist(sink); err != nil {
		t.Fatal(err)
	}
	restored := newStateMachine()
	if err := restored.Restore(io.NopCloser(bytes.NewReader(sink.Bytes()))); err != nil {
		t.Fatal(err)
	}
	_, _, actualHash, _ := restored.summary()
	if actualHash != expectedHash {
		t.Fatalf("state hash changed after restore: %s != %s", actualHash, expectedHash)
	}
}

func TestFSMRejectsMalformedHashAndStaleEpoch(t *testing.T) {
	fsm := newStateMachine()
	malformed := testCommand(t, "bad", "bad", "mission.create", 0, map[string]bool{"ok": true})
	malformed.PayloadHash = "bad"
	if result := applyTestCommand(t, fsm, malformed, 1); result.err == nil {
		t.Fatal("expected malformed hash rejection")
	}
	stale := testCommand(t, "stale", "stale", "mission.create", 2, map[string]bool{"ok": true})
	if result := applyTestCommand(t, fsm, stale, 2); result.err == nil {
		t.Fatal("expected stale epoch rejection")
	}
}

func TestFSMProjectionRevisionAndDeletionShareEntityKey(t *testing.T) {
	fsm := newStateMachine()
	if result := applyTestCommand(t, fsm, testCommand(t, "epoch", "epoch", "coordination.epoch_advance", 1, map[string]int{"term": 1}), 1); result.err != nil {
		t.Fatal(result.err)
	}
	create := testCommand(t, "create", "create", "mission.create", 1, map[string]string{"name": "Initial"})
	if result := applyTestCommand(t, fsm, create, 2); result.err != nil {
		t.Fatal(result.err)
	}
	revise := testCommand(t, "revise", "revise", "mission.revise", 1, map[string]string{"name": "Revised"})
	if result := applyTestCommand(t, fsm, revise, 3); result.err != nil {
		t.Fatal(result.err)
	}
	if len(fsm.state.Projections) != 2 || string(fsm.state.Projections["mission:mission-1"]) != string(revise.Payload) {
		t.Fatalf("revision did not replace the entity projection: %#v", fsm.state.Projections)
	}
	remove := testCommand(t, "delete", "delete", "mission.delete", 1, map[string]string{"id": "mission-1"})
	if result := applyTestCommand(t, fsm, remove, 4); result.err != nil {
		t.Fatal(result.err)
	}
	if _, exists := fsm.state.Projections["mission:mission-1"]; exists {
		t.Fatalf("delete retained entity projection: %#v", fsm.state.Projections)
	}
}

type memorySnapshotSink struct{ bytes.Buffer }

func (m *memorySnapshotSink) ID() string    { return "memory" }
func (m *memorySnapshotSink) Cancel() error { return nil }
func (m *memorySnapshotSink) Close() error  { return nil }
