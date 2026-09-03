package memory

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func testManager() *Manager {
	return New(Config{EmbedURL: "http://127.0.0.1:1", EmbedTokenFile: "missing"}, slog.Default())
}

func TestConversationAssemblyKeepsLatestTwelveTurns(t *testing.T) {
	m := testManager()
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		m.RecordExchange(ctx, fmt.Sprintf("turn-%d", i), "demo-operator", "voice", "", fmt.Sprintf("user %d", i), fmt.Sprintf("assistant %d", i), "mock")
	}
	assembly := m.Assemble(ctx, "turn-next", "demo-operator", "voice", "", "what did we discuss")
	if len(assembly.RecentTurns) != 12 {
		t.Fatalf("got %d recent turns", len(assembly.RecentTurns))
	}
	if assembly.RecentTurns[0].Content != "user 2" {
		t.Fatalf("unexpected first retained turn: %+v", assembly.RecentTurns[0])
	}
}

func TestCandidateNeedsExactHashAndTombstoneCannotResurrect(t *testing.T) {
	m := testManager()
	ctx := context.Background()
	candidate, err := m.DraftCandidate(domain.MemoryScopeV1{Kind: "operator", ID: "demo-operator"}, "preference", "Prefer tighter formation spacing.", "demo-operator", "turn-1", .9)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.DecideCandidate(ctx, candidate.ID, "approve", CandidateMutation{Mutation: Mutation{RequestID: "bad-hash", IdempotencyKey: "bad-hash", ActorID: "demo-operator", ExpectedVersion: m.Snapshot().StateVersion}, CandidateHash: "tampered"})
	if err == nil {
		t.Fatal("tampered hash accepted")
	}
	_, err = m.DecideCandidate(ctx, candidate.ID, "approve", CandidateMutation{Mutation: Mutation{RequestID: "approve", IdempotencyKey: "approve", ActorID: "demo-operator", ExpectedVersion: m.Snapshot().StateVersion}, CandidateHash: candidate.CandidateHash})
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := m.Search(ctx, SearchRequest{Query: "formation spacing", ActorID: "demo-operator", Limit: 5})
	if err != nil || len(items) == 0 {
		t.Fatalf("committed memory not retrievable: %v", err)
	}
	itemID := items[0].ItemID
	_, err = m.Forget(ctx, itemID, Mutation{RequestID: "forget", IdempotencyKey: "forget", ActorID: "demo-operator", ExpectedVersion: m.Snapshot().StateVersion})
	if err != nil {
		t.Fatal(err)
	}
	item, err := m.Item(itemID, "demo-operator")
	if err != nil || !item.Tombstoned {
		t.Fatalf("memory was not tombstoned: %+v %v", item, err)
	}
}

func TestScopeIsolation(t *testing.T) {
	m := testManager()
	_, err := m.DraftCandidate(domain.MemoryScopeV1{Kind: "operator", ID: "alice"}, "preference", "private preference", "bob", "turn", .9)
	if err == nil {
		t.Fatal("cross-operator candidate was accepted")
	}
}
