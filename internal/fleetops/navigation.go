package fleetops

import (
	_ "embed"
	"encoding/json"
	"math"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const navigationShoreMarginM = 20.0

// The simulation and the map must use the same coastline. Embedding the exact
// rendered fixture keeps vessel safety independent of the web client and makes
// node-local execution fail closed when a route attempts a shortcut over land.
//
//go:embed assets/narragansett.geojson
var navigationChartJSON []byte

var navigationLandPolygons = loadNavigationLandPolygons()

func loadNavigationLandPolygons() [][][]domain.GeoPointV2 {
	type geometry struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	type feature struct {
		Properties map[string]any `json:"properties"`
		Geometry   geometry       `json:"geometry"`
	}
	var chart struct {
		Features []feature `json:"features"`
	}
	if err := json.Unmarshal(navigationChartJSON, &chart); err != nil {
		panic("fleetops: embedded navigation chart is invalid: " + err.Error())
	}
	land := make([][][]domain.GeoPointV2, 0)
	for _, item := range chart.Features {
		if item.Properties["kind"] != "land" {
			continue
		}
		switch item.Geometry.Type {
		case "Polygon":
			var polygon [][]domain.GeoPointV2
			if err := json.Unmarshal(item.Geometry.Coordinates, &polygon); err != nil {
				panic("fleetops: embedded land polygon is invalid: " + err.Error())
			}
			land = append(land, polygon)
		case "MultiPolygon":
			var polygons [][][]domain.GeoPointV2
			if err := json.Unmarshal(item.Geometry.Coordinates, &polygons); err != nil {
				panic("fleetops: embedded land multipolygon is invalid: " + err.Error())
			}
			land = append(land, polygons...)
		}
	}
	if len(land) == 0 {
		panic("fleetops: embedded navigation chart contains no land polygons")
	}
	return land
}

func navigationPointSafe(point domain.GeoPointV2) bool {
	return combatPositionInBounds(point) && !navigationPointOnLand(point) && navigationDistanceToShoreM(point) >= navigationShoreMarginM
}

func navigationPointOnLand(point domain.GeoPointV2) bool {
	for _, polygon := range navigationLandPolygons {
		if len(polygon) == 0 || !navigationPointInRing(point, polygon[0]) {
			continue
		}
		insideHole := false
		for _, hole := range polygon[1:] {
			if navigationPointInRing(point, hole) {
				insideHole = true
				break
			}
		}
		if !insideHole {
			return true
		}
	}
	return false
}

func navigationPointInRing(point domain.GeoPointV2, ring []domain.GeoPointV2) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > point[1]) != (b[1] > point[1]) && point[0] < (b[0]-a[0])*(point[1]-a[1])/(b[1]-a[1])+a[0] {
			inside = !inside
		}
	}
	return inside
}

func navigationDistanceToShoreM(point domain.GeoPointV2) float64 {
	minimum := math.Inf(1)
	for _, polygon := range navigationLandPolygons {
		for _, ring := range polygon {
			for i := 1; i < len(ring); i++ {
				if distance := navigationPointSegmentDistanceM(point, ring[i-1], ring[i]); distance < minimum {
					minimum = distance
				}
			}
		}
	}
	return minimum
}

func navigationPointSegmentDistanceM(point, start, end domain.GeoPointV2) float64 {
	latitudeScale := math.Max(.2, math.Cos(point[1]*math.Pi/180))
	px, py := point[0]*111000*latitudeScale, point[1]*111000
	sx, sy := start[0]*111000*latitudeScale, start[1]*111000
	ex, ey := end[0]*111000*latitudeScale, end[1]*111000
	dx, dy := ex-sx, ey-sy
	if dx == 0 && dy == 0 {
		return math.Hypot(px-sx, py-sy)
	}
	t := ((px-sx)*dx + (py-sy)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(sx+t*dx), py-(sy+t*dy))
}

func navigationSegmentSafe(start, end domain.GeoPointV2) bool {
	distance := combatDistanceM(start, end)
	steps := int(math.Max(1, math.Ceil(distance/10)))
	for i := 0; i <= steps; i++ {
		fraction := float64(i) / float64(steps)
		point := domain.GeoPointV2{
			start[0] + (end[0]-start[0])*fraction,
			start[1] + (end[1]-start[1])*fraction,
		}
		if !navigationPointSafe(point) {
			return false
		}
	}
	return true
}

func nearestNavigationSafePoint(point domain.GeoPointV2) domain.GeoPointV2 {
	point = clampCombatPosition(point)
	if navigationPointSafe(point) {
		return point
	}
	for radius := 25.0; radius <= 12000; radius += 25 {
		for bearing := 0.0; bearing < 360; bearing += 10 {
			candidate := pointAtBearing(point, bearing, radius)
			if navigationPointSafe(candidate) {
				return candidate
			}
		}
	}
	// This should be unreachable for the packaged coastal operating picture.
	// Returning the known offshore patrol point fails safer than retaining a
	// persisted land position.
	return blackwakePatrol[0]
}

// navigationMove advances one deterministic swept step. If the direct bearing
// intersects land it chooses the smallest safe port/starboard deflection and
// follows the shoreline instead of clipping through it.
func navigationMove(start, destination domain.GeoPointV2, speed, seconds float64) (domain.GeoPointV2, float64, float64) {
	start = nearestNavigationSafePoint(start)
	destination = nearestNavigationSafePoint(destination)
	distance := combatDistanceM(start, destination)
	if distance <= 1 || speed <= 0 || seconds <= 0 {
		return start, 0, 0
	}
	step := math.Min(distance, speed*seconds)
	directBearing := combatBearingDeg(start, destination)
	offsets := []float64{0, -10, 10, -20, 20, -30, 30, -45, 45, -60, 60, -75, 75, -90, 90, -120, 120, 180}
	best := start
	bestBearing := directBearing
	bestScore := math.Inf(1)
	for _, offset := range offsets {
		candidate := destination
		if step < distance || offset != 0 {
			candidate = pointAtBearing(start, directBearing+offset, step)
		}
		if !navigationSegmentSafe(start, candidate) {
			continue
		}
		score := combatDistanceM(candidate, destination) + math.Abs(offset)*.05
		if score < bestScore {
			best, bestBearing, bestScore = candidate, directBearing+offset, score
		}
	}
	if math.IsInf(bestScore, 1) {
		return start, 0, 0
	}
	return best, normalizeHeading(bestBearing), speed
}
