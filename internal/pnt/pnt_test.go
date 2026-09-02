package pnt

import (
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"testing"
)

func TestSpoofIsExcludedBeforeFusion(t *testing.T) {
	actual := domain.Point{-70, 42}
	estimate, ghost := Spoof(actual)
	if estimate.Position != actual || estimate.Integrity != "unsafe" {
		t.Fatalf("estimate=%#v", estimate)
	}
	if d := DistanceMeters(actual, ghost); d < 600 || d > 700 {
		t.Fatalf("jump=%f", d)
	}
}
