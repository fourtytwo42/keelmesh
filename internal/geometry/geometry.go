package geometry

import (
	"errors"
	"math"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const earthKMPerDegree = 111.32

func NormalizeRing(ring []domain.Point) []domain.Point {
	if len(ring) == 0 {
		return nil
	}
	out := append([]domain.Point(nil), ring...)
	if out[0] != out[len(out)-1] {
		out = append(out, out[0])
	}
	return out
}

func ValidatePolygon(poly domain.Polygon, boundary domain.Polygon) error {
	if poly.Type != "Polygon" || len(poly.Coordinates) != 1 {
		return errors.New("one GeoJSON polygon ring is required")
	}
	ring := NormalizeRing(poly.Coordinates[0])
	vertices := len(ring) - 1
	if vertices < 3 || vertices > 24 {
		return errors.New("polygon must contain 3 to 24 vertices")
	}
	if math.Abs(SignedArea(ring)) < 1e-10 {
		return errors.New("polygon area is too small")
	}
	for i := 0; i < vertices; i++ {
		if !PointInPolygon(ring[i], boundary.Coordinates[0]) {
			return errors.New("polygon must stay inside the operational boundary")
		}
		for j := i + 1; j < vertices; j++ {
			if adjacent(i, j, vertices) {
				continue
			}
			if SegmentsIntersect(ring[i], ring[i+1], ring[j], ring[j+1]) {
				return errors.New("polygon may not self-intersect")
			}
		}
	}
	return nil
}

func adjacent(i, j, n int) bool { return i == j || (i+1)%n == j || (j+1)%n == i }

func SignedArea(ring []domain.Point) float64 {
	var sum float64
	for i := 0; i+1 < len(ring); i++ {
		sum += ring[i][0]*ring[i+1][1] - ring[i+1][0]*ring[i][1]
	}
	return sum / 2
}

func Bounds(ring []domain.Point) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range ring {
		minX = math.Min(minX, p[0])
		minY = math.Min(minY, p[1])
		maxX = math.Max(maxX, p[0])
		maxY = math.Max(maxY, p[1])
	}
	return
}

func PointInPolygon(p domain.Point, ring []domain.Point) bool {
	inside := false
	n := len(ring)
	if n > 1 && ring[0] == ring[n-1] {
		n--
	}
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		pi, pj := ring[i], ring[j]
		if pointOnSegment(p, pj, pi) {
			return true
		}
		intersects := (pi[1] > p[1]) != (pj[1] > p[1]) && p[0] < (pj[0]-pi[0])*(p[1]-pi[1])/(pj[1]-pi[1])+pi[0]
		if intersects {
			inside = !inside
		}
	}
	return inside
}

func pointOnSegment(p, a, b domain.Point) bool {
	cross := (p[1]-a[1])*(b[0]-a[0]) - (p[0]-a[0])*(b[1]-a[1])
	if math.Abs(cross) > 1e-10 {
		return false
	}
	return p[0] >= math.Min(a[0], b[0])-1e-10 && p[0] <= math.Max(a[0], b[0])+1e-10 && p[1] >= math.Min(a[1], b[1])-1e-10 && p[1] <= math.Max(a[1], b[1])+1e-10
}

func orientation(a, b, c domain.Point) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}

func SegmentsIntersect(a, b, c, d domain.Point) bool {
	o1, o2, o3, o4 := orientation(a, b, c), orientation(a, b, d), orientation(c, d, a), orientation(c, d, b)
	if o1*o2 < 0 && o3*o4 < 0 {
		return true
	}
	return math.Abs(o1) < 1e-10 && pointOnSegment(c, a, b) || math.Abs(o2) < 1e-10 && pointOnSegment(d, a, b) || math.Abs(o3) < 1e-10 && pointOnSegment(a, c, d) || math.Abs(o4) < 1e-10 && pointOnSegment(b, c, d)
}

func DistanceKM(a, b domain.Point) float64 {
	lat := (a[1] + b[1]) / 2 * math.Pi / 180
	dx := (b[0] - a[0]) * earthKMPerDegree * math.Cos(lat)
	dy := (b[1] - a[1]) * earthKMPerDegree
	return math.Hypot(dx, dy)
}

func RouteDistanceKM(route []domain.Point) float64 {
	var d float64
	for i := 1; i < len(route); i++ {
		d += DistanceKM(route[i-1], route[i])
	}
	return d
}

func InterpolateRoute(route []domain.Point, distanceKM float64) (domain.Point, int, float64) {
	if len(route) == 0 {
		return domain.Point{}, 0, 0
	}
	if len(route) == 1 || distanceKM <= 0 {
		return route[0], 0, 0
	}
	remaining := distanceKM
	for i := 1; i < len(route); i++ {
		seg := DistanceKM(route[i-1], route[i])
		if remaining <= seg {
			t := remaining / seg
			return domain.Point{route[i-1][0] + (route[i][0]-route[i-1][0])*t, route[i-1][1] + (route[i][1]-route[i-1][1])*t}, i - 1, t
		}
		remaining -= seg
	}
	return route[len(route)-1], len(route) - 1, 1
}

func PointSegmentDistanceKM(p, a, b domain.Point) float64 {
	lat := p[1] * math.Pi / 180
	sx := earthKMPerDegree * math.Cos(lat)
	sy := earthKMPerDegree
	ax, ay := (a[0]-p[0])*sx, (a[1]-p[1])*sy
	bx, by := (b[0]-p[0])*sx, (b[1]-p[1])*sy
	dx, dy := bx-ax, by-ay
	t := 0.0
	if denom := dx*dx + dy*dy; denom > 0 {
		t = math.Max(0, math.Min(1, -(ax*dx+ay*dy)/denom))
	}
	return math.Hypot(ax+t*dx, ay+t*dy)
}
