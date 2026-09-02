package arena

import (
	"strings"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func mutation(v int64, id string) domain.ArenaMutationV1 {
	return domain.ArenaMutationV1{RequestID: "request-" + id, IdempotencyKey: "key-" + id, ExpectedVersion: v, ActorID: "A"}
}

func TestPlayerKnowledgeIsFactionScoped(t *testing.T) {
	m := New()
	a, b := m.Snapshot("A"), m.Snapshot("B")
	if len(a.Nodes) != 6 || len(b.Nodes) != 6 {
		t.Fatalf("expected six nodes per faction: %d %d", len(a.Nodes), len(b.Nodes))
	}
	for _, n := range a.Nodes {
		if n.Faction != "A" {
			t.Fatalf("A received foreign node: %#v", n)
		}
	}
	for _, n := range b.Nodes {
		if n.Faction != "B" {
			t.Fatalf("B received foreign node: %#v", n)
		}
	}
	if a.Knowledge.Contacts[0].ID == b.Knowledge.Contacts[0].ID {
		t.Fatal("factions received the same contact projection")
	}
	if len(a.Coordinators) != 1 || a.Coordinators[0].Faction != "A" {
		t.Fatal("opponent coordination leaked")
	}
	if len(m.InfrastructureSnapshot().Nodes) != 12 {
		t.Fatal("referee topology should contain twelve nodes")
	}
}

func TestRadioFaultCannotTargetProtectedPlanes(t *testing.T) {
	m := New()
	v := m.Snapshot("A").StateVersion
	_, err := m.Fault(FaultRequest{ArenaMutationV1: mutation(v, "protected"), Faction: "A", Kind: "fail_management"})
	if err == nil || !strings.Contains(err.Error(), "PROTECTED_PLANE") {
		t.Fatalf("expected protected-plane rejection, got %v", err)
	}
	for _, n := range m.Snapshot("A").Nodes {
		if !n.ManagementConnected || !n.InferenceConnected {
			t.Fatal("protected connectivity changed")
		}
	}
}

func TestCoordinatorFailoverKeepsQuorumAndInference(t *testing.T) {
	m := New()
	v := m.Snapshot("A").StateVersion
	s, err := m.Fault(FaultRequest{ArenaMutationV1: mutation(v, "failover"), Faction: "A", Kind: "partition_coordinator"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Coordinators[0].NodeID != "node-a-02" || s.Coordinators[0].Epoch != 2 || s.Coordinators[0].Votes != 5 {
		t.Fatalf("unexpected coordinator: %#v", s.Coordinators[0])
	}
	for _, n := range s.Nodes {
		if !n.ManagementConnected || !n.InferenceConnected {
			t.Fatalf("protected plane lost on %s", n.ID)
		}
	}
}

func TestPiratePersonaChangesVoiceWithoutExpandingAuthority(t *testing.T) {
	m := New()
	s := m.Snapshot("A")
	session, err := m.CreateSession(AgentMessageRequest{
		ArenaMutationV1: mutation(s.StateVersion, "pirate-session"),
		Faction:         "A",
		Persona:         "pirate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(session.Message, "Aye, Captain") || !strings.Contains(session.Message, "no authority beyond") {
		t.Fatalf("pirate session omitted persona or authority boundary: %q", session.Message)
	}
	turn, err := m.AgentMessage(session.ID, AgentMessageRequest{
		ArenaMutationV1: mutation(m.Snapshot("A").StateVersion, "pirate-turn"),
		Faction:         "A",
		Text:            "Frame my fleet and radar contacts",
		Persona:         "pirate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(turn.Message, "Arrr, Captain") || !strings.Contains(turn.Message, "exact signed proposal") || !turn.AwaitingApproval {
		t.Fatalf("pirate turn weakened approval boundary: %#v", turn)
	}
}

func TestExactEngagementApprovalAndBoundedEffect(t *testing.T) {
	m := New()
	s := m.Snapshot("A")
	p, err := m.PlanEngagement(PlanRequest{ArenaMutationV1: mutation(s.StateVersion, "plan"), Faction: "A", FriendlyNodeIDs: []string{"node-a-05"}, TargetTrackIDs: []string{"track-a-001"}, Equipment: []string{"light_kinetic"}, MaximumEffects: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Authorize(p.ID, AuthorizeRequest{ArenaMutationV1: mutation(m.Snapshot("A").StateVersion, "bad-hash"), PlanHash: p.ContentHash + "tampered", OperatorID: "demo-player-a"})
	if err == nil || !strings.Contains(err.Error(), "HASH_MISMATCH") {
		t.Fatalf("expected exact hash rejection, got %v", err)
	}
	// A rejected request intentionally consumes its version boundary; use a fresh manager for the valid flow.
	m = New()
	s = m.Snapshot("A")
	p, err = m.PlanEngagement(PlanRequest{ArenaMutationV1: mutation(s.StateVersion, "valid-plan"), Faction: "A", FriendlyNodeIDs: []string{"node-a-05"}, TargetTrackIDs: []string{"track-a-001"}, Equipment: []string{"light_kinetic"}, MaximumEffects: 1})
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Authorize(p.ID, AuthorizeRequest{ArenaMutationV1: mutation(m.Snapshot("A").StateVersion, "authorize"), PlanHash: p.ContentHash, OperatorID: "demo-player-a"})
	if err != nil {
		t.Fatal(err)
	}
	e, err := m.ApplyEffect(EffectRequest{ArenaMutationV1: mutation(m.Snapshot("A").StateVersion, "effect"), LeaseID: l.ID, TargetTrackID: "track-a-001", Equipment: "light_kinetic"})
	if err != nil {
		t.Fatal(err)
	}
	if e.RemainingUses != 0 || e.ReceiptHash == "" {
		t.Fatalf("bad effect receipt: %#v", e)
	}
}
