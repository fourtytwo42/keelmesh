package geometry

import (
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestValidatePolygonRejectsInvalidGeometry(t *testing.T) {
	boundary := domain.NewPolygon([]domain.Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}})
	tests := []domain.Polygon{
		domain.NewPolygon([]domain.Point{{1, 1}, {9, 9}, {1, 9}, {9, 1}, {1, 1}}),
		domain.NewPolygon([]domain.Point{{1, 1}, {11, 1}, {1, 2}, {1, 1}}),
		domain.NewPolygon([]domain.Point{{1, 1}, {2, 2}, {1, 1}}),
	}
	for i, poly := range tests {
		if err := ValidatePolygon(poly, boundary); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}
