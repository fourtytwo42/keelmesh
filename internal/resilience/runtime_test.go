package resilience

import (
	"fmt"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"testing"
	"time"
)

func fixture() *Runtime {
	route := []domain.Point{{-70, 42}, {-69.99, 42.01}}
	assignments := []domain.AssignmentV1{}
	vessels := []domain.VesselV1{}
	assets := []string{}
	for i := 1; i <= 6; i++ {
		id := fmt.Sprintf("vessel-%02d", i)
		assignments = append(assignments, domain.AssignmentV1{VesselID: id, Route: route, SpeedMPS: 2})
		vessels = append(vessels, domain.VesselV1{ID: id, Position: route[0]})
		assets = append(assets, id)
	}
	lease := domain.MissionLeaseV1{ID: "lease", MissionID: "mission", PlanID: "plan", PlanHash: "hash", AssetIDs: assets, MinReserve: .3, ExpiresAt: time.Now().Add(time.Hour)}
	return New(lease, domain.PlanCandidateV1{ID: "plan", ContentHash: "hash", Assignments: assignments}, vessels, 2)
}

func TestDeterministicIncidentEndsInBridgeWithoutReplay(t *testing.T) {
	r := fixture()
	for _, f := range []string{FaultFailStarlink, FaultPartition, FaultGNSSSpoof, FaultRestore} {
		if _, err := r.Apply(f); err != nil {
			t.Fatal(err)
		}
	}
	s := r.Snapshot()
	if s.Phase != "rejoined" || s.Bridge == nil || s.Bridge.TargetSequence != 9 || len(s.DiscardedSequences) != 3 {
		t.Fatalf("snapshot=%#v", s)
	}
	if s.RawGNSSPosition == nil || s.Nodes[4].PNT.Position == *s.RawGNSSPosition {
		t.Fatal("spoof affected fused position")
	}
}

func TestFaultsFailClosedOutOfOrder(t *testing.T) {
	if _, err := fixture().Apply(FaultRestore); err == nil {
		t.Fatal("out of order fault accepted")
	}
}
