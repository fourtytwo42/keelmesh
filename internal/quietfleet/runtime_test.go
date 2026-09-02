package quietfleet

import (
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func fixture() *Runtime {
	plan := domain.PlanCandidateV1{ID: "plan", ContentHash: "sha256:plan", Assignments: []domain.AssignmentV1{}}
	for i := 2; i <= 5; i++ {
		plan.Assignments = append(plan.Assignments, domain.AssignmentV1{VesselID: []string{"vessel-02", "vessel-03", "vessel-04", "vessel-05"}[i-2], SpeedMPS: 1.4, Route: []domain.Point{{-70, 40}, {-69.99, 40.01}, {-69.98, 40.02}, {-69.97, 40.03}}})
	}
	return New(domain.MissionLeaseV1{MissionID: "mission"}, plan, 7)
}

func TestRejectArmCommitAndFutureActivation(t *testing.T) {
	r := fixture()
	for _, command := range []string{EnterMode, InjectSlowdown} {
		if _, err := r.Apply(command, ""); err != nil {
			t.Fatal(err)
		}
	}
	s := r.Snapshot()
	if s.Metrics.QuorumCount != 3 || s.Metrics.AffectedArmed != 3 {
		t.Fatalf("metrics=%+v", s.Metrics)
	}
	if _, err := r.Apply(CommitProposal, s.Proposal.ContentHash); err == nil {
		t.Fatal("rejected proposal committed")
	}
	if _, err := r.Apply(SubmitRevision, ""); err != nil {
		t.Fatal(err)
	}
	s = r.Snapshot()
	original := hash(s.ActiveAssignments)
	if _, err := r.Apply(CommitProposal, s.Proposal.ContentHash); err != nil {
		t.Fatal(err)
	}
	committed := r.Snapshot()
	if hash(committed.ActiveAssignments) != original || committed.Commit.ActivationTick <= committed.MissionTick {
		t.Fatal("commit changed active state early")
	}
	if _, err := r.Apply(AdvanceActivation, ""); err != nil {
		t.Fatal(err)
	}
	if r.Snapshot().Phase != "activated" || hash(r.Snapshot().ActiveAssignments) == original {
		t.Fatal("revision did not activate")
	}
}

func TestTamperedHashAndBudgetStayClosed(t *testing.T) {
	r := fixture()
	_, _ = r.Apply(EnterMode, "")
	_, _ = r.Apply(InjectSlowdown, "")
	_, _ = r.Apply(SubmitRevision, "")
	if _, err := r.Apply(CommitProposal, "sha256:tampered"); err == nil || err.Error() != "COMMIT_HASH_MISMATCH" {
		t.Fatalf("err=%v", err)
	}
	if r.Snapshot().Metrics.BytesSent > r.Snapshot().Metrics.ByteBudget {
		t.Fatal("budget exceeded")
	}
}
