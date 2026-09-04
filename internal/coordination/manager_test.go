package coordination

import (
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestEpochAdvanceIsStableWithinTermAndUniqueAcrossTerms(t *testing.T) {
	manager := &Manager{cfg: Config{
		Identity: domain.NodeIdentityV2{NodeID: "node-a-06", CellID: "A"},
		Manifest: domain.CoordinationCellManifestV1{ClusterID: "keelmesh-cell-a"},
	}}
	fsm := newStateMachine()

	first := manager.epochAdvanceCommand(0, 6, time.Unix(100, 0))
	firstResult := applyTestCommand(t, fsm, first, 1)
	if firstResult.err != nil {
		t.Fatal(firstResult.err)
	}

	// Model a lost apply response: the runtime rereads the now-advanced epoch
	// and retries in the same leadership term.  The immutable term identity
	// must return the original receipt rather than conflict or advance twice.
	retry := manager.epochAdvanceCommand(1, 6, time.Unix(101, 0))
	if retry.CommandID != first.CommandID || retry.IdempotencyKey != first.IdempotencyKey || retry.PayloadHash != first.PayloadHash {
		t.Fatal("same-term epoch retry changed its immutable identity")
	}
	retryResult := applyTestCommand(t, fsm, retry, 2)
	if retryResult.err != nil {
		t.Fatal(retryResult.err)
	}
	if retryResult.receipt.LogIndex != firstResult.receipt.LogIndex {
		t.Fatal("same-term retry did not return the original receipt")
	}

	second := manager.epochAdvanceCommand(1, 7, time.Unix(102, 0))
	if second.CommandID == first.CommandID || second.IdempotencyKey == first.IdempotencyKey {
		t.Fatal("new Raft term reused the prior epoch identity")
	}
	secondResult := applyTestCommand(t, fsm, second, 3)
	if secondResult.err != nil {
		t.Fatal(secondResult.err)
	}
	epoch, _, _, _ := fsm.summary()
	if epoch != 2 {
		t.Fatalf("expected authority epoch 2, got %d", epoch)
	}
}
