package pnt

import (
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"math"
)

func Trusted(position domain.Point) domain.PntEstimateV1 {
	return estimate(position, 8, "trusted", []string{"gnss", "inertial", "radar_shoreline", "peer_relative"}, nil, nil, "mission")
}

func Partitioned(position domain.Point) domain.PntEstimateV1 {
	return estimate(position, 24, "suspect", []string{"inertial", "radar_shoreline"}, []string{"peer_relative"}, []string{"CORROBORATION_LOST"}, "dead_reckoning")
}

func Spoof(position domain.Point) (domain.PntEstimateV1, domain.Point) {
	// About 650 m northeast at the fictional chart latitude.
	ghost := domain.Point{position[0] + 0.0058, position[1] + 0.0041}
	e := estimate(position, 52, "unsafe", []string{"inertial"}, []string{"gnss", "peer_relative", "radar_shoreline"}, []string{"GNSS_POSITION_JUMP", "GNSS_VELOCITY_INCONSISTENT", "GNSS_CLOCK_INCONSISTENT", "UNCERTAINTY_LIMIT_EXCEEDED"}, "safe_hold")
	return e, ghost
}

func Recovered(position domain.Point) domain.PntEstimateV1 {
	return estimate(position, 9, "trusted", []string{"inertial", "radar_shoreline", "peer_relative"}, []string{"gnss"}, []string{"GNSS_REMAINS_EXCLUDED", "CORROBORATION_RESTORED"}, "rejoined")
}

func Observations(position domain.Point, phase string, sequence int64) []domain.PntObservationV1 {
	good := func(source string, uncertainty float64) domain.PntObservationV1 {
		return domain.PntObservationV1{SchemaVersion: 1, Source: source, Position: position, Sequence: sequence, UncertaintyM: uncertainty, IntegrityOK: true}
	}
	switch phase {
	case "partitioned":
		return []domain.PntObservationV1{good("inertial", 18), good("radar_shoreline", 20)}
	case "safe_hold":
		_, ghost := Spoof(position)
		return []domain.PntObservationV1{{SchemaVersion: 1, Source: "gnss", Position: ghost, SpeedMPS: 21, Sequence: sequence, UncertaintyM: 3, IntegrityOK: false, ReasonCodes: []string{"POSITION_JUMP", "VELOCITY_INCONSISTENT", "CLOCK_INCONSISTENT"}}, good("inertial", 42)}
	case "rejoined":
		return []domain.PntObservationV1{good("inertial", 12), good("radar_shoreline", 8), good("peer_relative", 7)}
	default:
		return []domain.PntObservationV1{good("gnss", 4), good("inertial", 9), good("radar_shoreline", 8), good("peer_relative", 7)}
	}
}

func DistanceMeters(a, b domain.Point) float64 {
	dx := (b[0] - a[0]) * 111320 * math.Cos(a[1]*math.Pi/180)
	dy := (b[1] - a[1]) * 110540
	return math.Hypot(dx, dy)
}

func estimate(position domain.Point, uncertainty float64, integrity string, contributing, excluded, reasons []string, behavior string) domain.PntEstimateV1 {
	if contributing == nil {
		contributing = []string{}
	}
	if excluded == nil {
		excluded = []string{}
	}
	if reasons == nil {
		reasons = []string{}
	}
	return domain.PntEstimateV1{SchemaVersion: 1, Position: position, UncertaintyM: uncertainty, Integrity: integrity, ContributingSources: contributing, ExcludedSources: excluded, ReasonCodes: reasons, LeaseThresholdM: 45, Behavior: behavior}
}
