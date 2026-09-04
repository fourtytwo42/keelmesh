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

func TestConversationAssemblyContinuesAcrossMissionContext(t *testing.T) {
	m := testManager()
	ctx := context.Background()
	m.RecordExchange(ctx, "global", "demo-operator", "voice-and-chat", "", "Tell me about Atlantic Beacon.", "Atlantic Beacon is a container ship.", "mock")
	m.RecordExchange(ctx, "mission", "demo-operator", "voice-and-chat", "mission-1", "Plan a rendezvous.", "I prepared three options.", "mock")
	assembly := m.Assemble(ctx, "follow-up", "demo-operator", "voice-and-chat", "mission-1", "Which boat was I discussing?")
	if len(assembly.RecentTurns) != 4 || assembly.RecentTurns[0].Content != "Tell me about Atlantic Beacon." || assembly.RecentTurns[3].Content != "I prepared three options." {
		t.Fatalf("voice/text history did not continue across mission context: %+v", assembly.RecentTurns)
	}
}

func TestConversationTurnsAreSessionScopedAndBounded(t *testing.T) {
	m := testManager()
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		m.RecordExchange(ctx, fmt.Sprintf("shared-%d", i), "demo-operator", "browser-a", "", fmt.Sprintf("user %d", i), fmt.Sprintf("assistant %d", i), "mock")
	}
	m.RecordExchange(ctx, "other-session", "demo-operator", "browser-b", "", "private question", "private answer", "mock")
	m.RecordExchange(ctx, "mission-turn", "demo-operator", "browser-a", "mission-1", "mission question", "mission answer", "mock")

	turns := m.ConversationTurns("demo-operator", "browser-a", 4)
	if len(turns) != 4 || turns[0].Content != "user 3" || turns[3].Content != "mission answer" {
		t.Fatalf("unexpected bounded transcript: %+v", turns)
	}
	for _, turn := range turns {
		if turn.SessionID != "browser-a" {
			t.Fatalf("cross-scope turn leaked into global chat: %+v", turn)
		}
	}
}

func TestClearConversationPreservesLongTermMemoryAndOtherSessions(t *testing.T) {
	m := testManager()
	ctx := context.Background()
	m.RecordExchange(ctx, "remember", "demo-operator", "browser-a", "", "Remember I prefer wide spacing.", "Understood.", "mock")
	m.RecordExchange(ctx, "other", "demo-operator", "browser-b", "", "Other question", "Other answer", "mock")
	before := m.Snapshot()
	after, err := m.ClearConversation(ctx, "browser-a", Mutation{RequestID: "clear-a", IdempotencyKey: "clear-a", ActorID: "demo-operator", ExpectedVersion: before.StateVersion})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ConversationTurns("demo-operator", "browser-a", 50)) != 0 {
		t.Fatal("cleared session transcript remained visible")
	}
	if len(m.ConversationTurns("demo-operator", "browser-b", 50)) != 2 {
		t.Fatal("another session transcript was cleared")
	}
	if after.CommittedItems == 0 || len(m.items) == 0 {
		t.Fatal("long-term semantic memory was cleared with chat")
	}
	if _, err = m.ClearConversation(ctx, "browser-a", Mutation{RequestID: "clear-a", IdempotencyKey: "clear-a", ActorID: "demo-operator", ExpectedVersion: before.StateVersion}); err != nil {
		t.Fatalf("idempotent clear failed: %v", err)
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
