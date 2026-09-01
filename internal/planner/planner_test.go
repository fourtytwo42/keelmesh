package planner

import (
	"reflect"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/geometry"
	"github.com/fourtytwo42/keelmesh/internal/scenario"
)

func TestGoldenPlansAreDeterministicAndPolicyValid(t *testing.T) {
	s := scenario.Golden()
	intent := domain.MissionIntentV1{
		SchemaVersion: domain.SchemaVersion, ID: "intent-golden", SourceStateVersion: 2,
		Area:        s.SuggestedArea.Geometry,
		Constraints: domain.IntentConstraintsV1{MinimumReserve: .30, MaximumDurationMinutes: 20, AvoidZones: []string{s.Exclusion.ID}},
	}
	p := Planner{Scenario: s}
	first, err := p.Generate(intent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.Generate(intent)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same seed and intent produced different plans")
	}
	if len(first) != 2 {
		t.Fatalf("got %d candidates, want 2", len(first))
	}

	recommended := 0
	for _, plan := range first {
		if plan.Recommended {
			recommended++
		}
		if plan.Policy.Status != "approval_required" {
			t.Fatalf("%s policy=%s reasons=%v metrics=%+v", plan.Name, plan.Policy.Status, plan.Policy.ReasonCodes, plan.Metrics)
		}
		if len(plan.Assignments) != 6 {
			t.Fatalf("%s assigned %d vessels", plan.Name, len(plan.Assignments))
		}
		if plan.ContentHash == "" || plan.Metrics.CoveragePercent <= 0 || plan.Metrics.DurationMinutes <= 0 {
			t.Fatalf("%s has incomplete computed output: %+v", plan.Name, plan.Metrics)
		}
		for _, assignment := range plan.Assignments {
			for i := 1; i < len(assignment.Route); i++ {
				for sample := 0; sample <= 100; sample++ {
					f := float64(sample) / 100
					pt := domain.Point{
						assignment.Route[i-1][0] + f*(assignment.Route[i][0]-assignment.Route[i-1][0]),
						assignment.Route[i-1][1] + f*(assignment.Route[i][1]-assignment.Route[i-1][1]),
					}
					if !geometry.PointInPolygon(pt, s.Boundary.Geometry.Coordinates[0]) {
						t.Fatalf("%s route left operational boundary at %v", plan.Name, pt)
					}
					if strictlyInside(pt, s.Exclusion.Geometry.Coordinates[0]) {
						t.Fatalf("%s route entered exclusion zone at %v", plan.Name, pt)
					}
				}
			}
		}
	}
	if recommended != 1 {
		t.Fatalf("got %d recommended plans, want 1", recommended)
	}
}

func strictlyInside(p domain.Point, ring []domain.Point) bool {
	minX, minY, maxX, maxY := geometry.Bounds(ring)
	return p[0] > minX && p[0] < maxX && p[1] > minY && p[1] < maxY
}
