package scenario

import "github.com/fourtytwo42/keelmesh/internal/domain"

type Scenario struct {
	ID, Name                                    string
	Boundary, SuggestedArea, Exclusion, Holding domain.ZoneV1
	Vessels                                     []domain.VesselV1
	SensorRadiusKM                              float64
	SimulationRate                              float64
}

func rect(id, name, kind string, minX, minY, maxX, maxY float64) domain.ZoneV1 {
	ring := []domain.Point{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}, {minX, minY}}
	return domain.ZoneV1{ID: id, Name: name, Kind: kind, Geometry: domain.NewPolygon(ring)}
}

func Golden() Scenario {
	return Scenario{
		ID: "golden-basin-v1", Name: "Keel Basin — simulated waters",
		Boundary:       rect("operational-1", "Authorized operating boundary", "boundary", -70.012, 39.992, -69.952, 40.042),
		SuggestedArea:  rect("search-suggested", "Suggested search area", "search", -70.000, 40.008, -69.978, 40.020),
		Exclusion:      rect("exclusion-2", "Protected exclusion zone", "exclusion", -69.992, 40.011, -69.988, 40.015),
		Holding:        rect("holding-1", "Safe holding area", "holding", -70.006, 39.998, -70.001, 40.003),
		SensorRadiusKM: 0.24, SimulationRate: 180,
		Vessels: []domain.VesselV1{
			{ID: "vessel-01", Name: "Vessel 1", Position: domain.Point{-70.005, 40.001}, HeadingDeg: 45, Reserve: .82, Available: true},
			{ID: "vessel-02", Name: "Vessel 2", Position: domain.Point{-70.002, 40.006}, HeadingDeg: 60, Reserve: .79, Available: true},
			{ID: "vessel-03", Name: "Vessel 3", Position: domain.Point{-70.006, 40.012}, HeadingDeg: 75, Reserve: .86, Available: true},
			{ID: "vessel-04", Name: "Vessel 4", Position: domain.Point{-70.004, 40.020}, HeadingDeg: 90, Reserve: .76, Available: true},
			{ID: "vessel-05", Name: "Vessel 5", Position: domain.Point{-69.998, 39.998}, HeadingDeg: 30, Reserve: .88, Available: true},
			{ID: "vessel-06", Name: "Vessel 6", Position: domain.Point{-69.991, 39.997}, HeadingDeg: 20, Reserve: .81, Available: true},
		},
	}
}
