package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/geometry"
	"github.com/fourtytwo42/keelmesh/internal/scenario"
)

type Planner struct{ Scenario scenario.Scenario }

type strategy struct {
	id, name, key, summary string
	horizontal             bool
	speed                  float64
}

func (p Planner) Generate(intent domain.MissionIntentV1) ([]domain.PlanCandidateV1, error) {
	strategies := []strategy{
		{id: "fast", name: "Fast Parallel Sweep", key: "fast_parallel", summary: "Finishes sooner with a higher-speed east–west sweep.", horizontal: true, speed: 4.2},
		{id: "reserve", name: "Reserve-First Sweep", key: "reserve_first", summary: "Reduces speed and turn cost while preserving the search constraints.", horizontal: false, speed: 3.8},
	}
	plans := make([]domain.PlanCandidateV1, 0, 2)
	for _, s := range strategies {
		plan, err := p.build(intent, s)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	best := -1
	bestScore := -1.0
	for i := range plans {
		if plans[i].Policy.Status != "prohibited" && plans[i].Score.Total > bestScore {
			best = i
			bestScore = plans[i].Score.Total
		}
	}
	if best >= 0 {
		plans[best].Recommended = true
	}
	return plans, nil
}

func (p Planner) build(intent domain.MissionIntentV1, s strategy) (domain.PlanCandidateV1, error) {
	ring := geometry.NormalizeRing(intent.Area.Coordinates[0])
	lanes := makeLanes(ring, len(p.Scenario.Vessels), s.horizontal)
	assignments := assignNearest(p.Scenario.Vessels, lanes, s.speed, p.Scenario.Exclusion.Geometry)
	metrics := p.metrics(intent.Area, assignments, s.speed)
	policy := p.policy(intent, assignments, metrics)
	score := score(metrics, intent.Constraints)
	plan := domain.PlanCandidateV1{SchemaVersion: domain.SchemaVersion, ID: fmt.Sprintf("plan-%s-%s", s.id, intent.ID), IntentID: intent.ID, SourceStateVersion: intent.SourceStateVersion, Name: s.name, Strategy: s.key, Summary: s.summary, Assignments: assignments, Metrics: metrics, Policy: policy, Score: score}
	h, err := HashPlan(plan)
	if err != nil {
		return domain.PlanCandidateV1{}, err
	}
	plan.ContentHash = h
	return plan, nil
}

func makeLanes(ring []domain.Point, count int, horizontal bool) [][]domain.Point {
	minX, minY, maxX, maxY := geometry.Bounds(ring)
	lanes := make([][]domain.Point, 0, count)
	for i := 0; i < count; i++ {
		f := (float64(i) + .5) / float64(count)
		if horizontal {
			y := minY + (maxY-minY)*f
			xs := intersections(ring, y, true)
			if len(xs) >= 2 {
				a, b := domain.Point{xs[0], y}, domain.Point{xs[len(xs)-1], y}
				if i%2 == 1 {
					a, b = b, a
				}
				lanes = append(lanes, []domain.Point{a, b})
			}
		} else {
			x := minX + (maxX-minX)*f
			ys := intersections(ring, x, false)
			if len(ys) >= 2 {
				a, b := domain.Point{x, ys[0]}, domain.Point{x, ys[len(ys)-1]}
				if i%2 == 1 {
					a, b = b, a
				}
				lanes = append(lanes, []domain.Point{a, b})
			}
		}
	}
	return lanes
}

func intersections(ring []domain.Point, value float64, horizontal bool) []float64 {
	vals := []float64{}
	for i := 1; i < len(ring); i++ {
		a, b := ring[i-1], ring[i]
		if horizontal {
			if (a[1] > value) == (b[1] > value) || a[1] == b[1] {
				continue
			}
			vals = append(vals, a[0]+(value-a[1])*(b[0]-a[0])/(b[1]-a[1]))
		} else {
			if (a[0] > value) == (b[0] > value) || a[0] == b[0] {
				continue
			}
			vals = append(vals, a[1]+(value-a[0])*(b[1]-a[1])/(b[0]-a[0]))
		}
	}
	sort.Float64s(vals)
	return vals
}

func assignNearest(vessels []domain.VesselV1, lanes [][]domain.Point, speed float64, exclusion domain.Polygon) []domain.AssignmentV1 {
	remaining := append([]domain.VesselV1(nil), vessels...)
	out := make([]domain.AssignmentV1, 0, len(lanes))
	for _, lane := range lanes {
		best := 0
		bestD := math.Inf(1)
		for i, v := range remaining {
			if d := geometry.DistanceKM(v.Position, lane[0]); d < bestD {
				best, bestD = i, d
			}
		}
		v := remaining[best]
		remaining = append(remaining[:best], remaining[best+1:]...)
		route := avoidPolygon(append([]domain.Point{v.Position}, lane...), exclusion)
		out = append(out, domain.AssignmentV1{VesselID: v.ID, Route: route, SpeedMPS: speed, DistanceKM: round(geometry.RouteDistanceKM(route), 3)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VesselID < out[j].VesselID })
	return out
}

func avoidPolygon(route []domain.Point, exclusion domain.Polygon) []domain.Point {
	if len(route) < 2 {
		return route
	}
	minX, minY, maxX, maxY := geometry.Bounds(exclusion.Coordinates[0])
	pad := .0008
	out := []domain.Point{route[0]}
	for i := 1; i < len(route); i++ {
		a, b := out[len(out)-1], route[i]
		if segmentCrossesRect(a, b, minX, minY, maxX, maxY) {
			out = append(out, shortestRectDetour(a, b, minX, minY, maxX, maxY, pad)...)
		}
		out = append(out, b)
	}
	return out
}

func shortestRectDetour(a, b domain.Point, minX, minY, maxX, maxY, pad float64) []domain.Point {
	corners := []domain.Point{{minX - pad, minY - pad}, {minX - pad, maxY + pad}, {maxX + pad, maxY + pad}, {maxX + pad, minY - pad}}
	bestDistance := math.Inf(1)
	best := []domain.Point{}
	var search func([]int, []bool)
	search = func(order []int, used []bool) {
		if len(order) > 0 {
			candidate := []domain.Point{a}
			for _, index := range order {
				candidate = append(candidate, corners[index])
			}
			candidate = append(candidate, b)
			if clearOfRect(candidate, minX, minY, maxX, maxY) {
				if distance := geometry.RouteDistanceKM(candidate); distance < bestDistance {
					bestDistance = distance
					best = append([]domain.Point(nil), candidate[1:len(candidate)-1]...)
				}
			}
		}
		if len(order) == len(corners) {
			return
		}
		for i := range corners {
			if used[i] {
				continue
			}
			used[i] = true
			search(append(order, i), used)
			used[i] = false
		}
	}
	search(nil, make([]bool, len(corners)))
	return best
}

func clearOfRect(route []domain.Point, minX, minY, maxX, maxY float64) bool {
	for i := 1; i < len(route); i++ {
		if segmentCrossesRect(route[i-1], route[i], minX, minY, maxX, maxY) {
			return false
		}
	}
	return true
}

func segmentCrossesRect(a, b domain.Point, minX, minY, maxX, maxY float64) bool {
	if a[0] > minX && a[0] < maxX && a[1] > minY && a[1] < maxY {
		return true
	}
	if b[0] > minX && b[0] < maxX && b[1] > minY && b[1] < maxY {
		return true
	}
	edges := [][2]domain.Point{{{minX, minY}, {maxX, minY}}, {{maxX, minY}, {maxX, maxY}}, {{maxX, maxY}, {minX, maxY}}, {{minX, maxY}, {minX, minY}}}
	for _, e := range edges {
		if geometry.SegmentsIntersect(a, b, e[0], e[1]) {
			return true
		}
	}
	return false
}

func (p Planner) metrics(area domain.Polygon, assignments []domain.AssignmentV1, speed float64) domain.PlanMetricsV1 {
	var total, maxD, minReserve float64
	minReserve = 1
	for _, a := range assignments {
		total += a.DistanceKM
		maxD = math.Max(maxD, a.DistanceKM)
		start := .75
		for _, v := range p.Scenario.Vessels {
			if v.ID == a.VesselID {
				start = v.Reserve
			}
		}
		turns := math.Max(0, float64(len(a.Route)-2))
		consumption := a.DistanceKM*(.018+.0018*speed*speed) + turns*.0015
		minReserve = math.Min(minReserve, start-consumption)
	}
	coverage := coverage(area, p.Scenario.Exclusion.Geometry, assignments, p.Scenario.SensorRadiusKM)
	return domain.PlanMetricsV1{CoveragePercent: round(coverage*100, 1), DurationMinutes: round(maxD/(speed*0.06), 1), MinimumReserve: round(math.Max(0, minReserve), 3), TotalRouteDistanceKM: round(total, 2)}
}

func coverage(area, exclusion domain.Polygon, assignments []domain.AssignmentV1, radiusKM float64) float64 {
	minX, minY, maxX, maxY := geometry.Bounds(area.Coordinates[0])
	hit, total := 0, 0
	for yi := 0; yi < 30; yi++ {
		for xi := 0; xi < 36; xi++ {
			p := domain.Point{minX + (maxX-minX)*(float64(xi)+.5)/36, minY + (maxY-minY)*(float64(yi)+.5)/30}
			if !geometry.PointInPolygon(p, area.Coordinates[0]) || geometry.PointInPolygon(p, exclusion.Coordinates[0]) {
				continue
			}
			total++
			covered := false
			for _, a := range assignments {
				for i := 2; i < len(a.Route); i++ {
					if geometry.PointSegmentDistanceKM(p, a.Route[i-1], a.Route[i]) <= radiusKM {
						covered = true
						break
					}
				}
				if covered {
					break
				}
			}
			if covered {
				hit++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func (p Planner) policy(intent domain.MissionIntentV1, assignments []domain.AssignmentV1, m domain.PlanMetricsV1) domain.PolicyDecisionV1 {
	reasons := []string{}
	if m.MinimumReserve < intent.Constraints.MinimumReserve {
		reasons = append(reasons, "RESERVE_BELOW_MINIMUM")
	}
	if m.DurationMinutes > intent.Constraints.MaximumDurationMinutes {
		reasons = append(reasons, "DURATION_EXCEEDS_LIMIT")
	}
	for _, a := range assignments {
		for _, pt := range a.Route {
			if !geometry.PointInPolygon(pt, p.Scenario.Boundary.Geometry.Coordinates[0]) {
				reasons = append(reasons, "ROUTE_OUTSIDE_BOUNDARY")
				break
			}
		}
	}
	if len(reasons) > 0 {
		return domain.PolicyDecisionV1{Status: "prohibited", ReasonCodes: unique(reasons), Summary: "The plan violates a deterministic mission constraint."}
	}
	return domain.PolicyDecisionV1{Status: "approval_required", ReasonCodes: []string{"INITIAL_DEPLOYMENT"}, Summary: "Policy-valid and ready for explicit operator authorization."}
}

func score(m domain.PlanMetricsV1, c domain.IntentConstraintsV1) domain.ScoreBreakdownV1 {
	coverage := m.CoveragePercent
	reserve := clamp((m.MinimumReserve - c.MinimumReserve) / (1 - c.MinimumReserve) * 100)
	duration := clamp((1 - m.DurationMinutes/c.MaximumDurationMinutes) * 100)
	return domain.ScoreBreakdownV1{Coverage: round(coverage, 1), Reserve: round(reserve, 1), Duration: round(duration, 1), Total: round(.5*coverage+.3*reserve+.2*duration, 1)}
}

func HashPlan(plan domain.PlanCandidateV1) (string, error) {
	plan.ContentHash = ""
	plan.Recommended = false
	b, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func Preview(plan domain.PlanCandidateV1) domain.PlanPreviewV1 {
	duration := int(math.Ceil(plan.Metrics.DurationMinutes * 60))
	samples := make([]domain.PreviewSampleV1, 0, duration+1)
	for second := 0; second <= duration; second++ {
		positions := map[string]domain.Point{}
		for _, a := range plan.Assignments {
			d := float64(second) * a.SpeedMPS / 1000
			pt, _, _ := geometry.InterpolateRoute(a.Route, d)
			positions[a.VesselID] = pt
		}
		samples = append(samples, domain.PreviewSampleV1{Second: second, Positions: positions})
	}
	return domain.PlanPreviewV1{PlanID: plan.ID, PlanHash: plan.ContentHash, DurationSeconds: duration, Samples: samples}
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
func round(v float64, n int) float64 { p := math.Pow10(n); return math.Round(v*p) / p }
func unique(in []string) []string {
	m := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if !m[v] {
			m[v] = true
			out = append(out, v)
		}
	}
	return out
}
