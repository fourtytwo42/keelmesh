package fleetops

import (
	"log/slog"
	"testing"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestNavigationMoveRoutesAroundRenderedLand(t *testing.T) {
	land := blockIslandInteriorPoint(t)
	west := nearestSafeOnLongitude(t, land, -1)
	east := nearestSafeOnLongitude(t, land, 1)
	if navigationSegmentSafe(west, east) {
		t.Fatalf("test chord unexpectedly avoids land: west=%v east=%v", west, east)
	}

	position := west
	for step := 0; step < 2400 && combatDistanceM(position, east) > 25; step++ {
		previous := position
		position, _, _ = navigationMove(position, east, 40, 1)
		if !navigationPointSafe(position) || !navigationSegmentSafe(previous, position) {
			t.Fatalf("navigation entered land on step %d: %v -> %v", step, previous, position)
		}
	}
	if distance := combatDistanceM(position, east); distance > 25 {
		t.Fatalf("navigation failed to route around land; %.1f m remain", distance)
	}
}

func TestNavigationReconcilesPersistedLandPositions(t *testing.T) {
	m := New("", slog.Default())
	land := blockIslandInteriorPoint(t)
	vesselID := m.Snapshot().Vessels[0].ID
	vessel := m.vessels[vesselID]
	vessel.Telemetry.Position = land
	m.vessels[vesselID] = vessel
	raider := m.combatEntities[blackwakeID]
	raider.Position = land
	m.combatEntities[blackwakeID] = raider

	m.reconcileNavigationPositionsLocked()
	if got := m.vessels[vesselID].Telemetry.Position; !navigationPointSafe(got) {
		t.Fatalf("controlled vessel was not moved to safe water: %v", got)
	}
	if got := m.combatEntities[blackwakeID].Position; !navigationPointSafe(got) {
		t.Fatalf("Blackwake was not moved to safe water: %v", got)
	}
}

func TestEveryFleetMovementStepRejectsLand(t *testing.T) {
	m := New("", slog.Default())
	land := blockIslandInteriorPoint(t)
	west := nearestSafeOnLongitude(t, land, -1)
	east := nearestSafeOnLongitude(t, land, 1)

	raider := m.combatEntities[blackwakeID]
	raider.Position = west
	for step := 0; step < 400; step++ {
		moveCombatEntity(&raider, east, blackwakeMaximumSpeedMPS, .2)
		if !navigationPointSafe(raider.Position) {
			t.Fatalf("combat movement entered land on step %d at %v", step, raider.Position)
		}
	}

	vessel := m.vessels[m.Snapshot().Vessels[0].ID]
	vessel.Telemetry.Position = west
	for step := 0; step < 400; step++ {
		vessel.Telemetry.Position, _, _ = navigationMove(vessel.Telemetry.Position, east, vessel.Class.MaxSpeedMPS, .2)
		if !navigationPointSafe(vessel.Telemetry.Position) {
			t.Fatalf("mission movement entered land on step %d at %v", step, vessel.Telemetry.Position)
		}
	}
}

func blockIslandInteriorPoint(t *testing.T) domain.GeoPointV2 {
	t.Helper()
	for latitude := 41.14; latitude <= 41.22; latitude += .002 {
		for longitude := -71.64; longitude <= -71.52; longitude += .002 {
			point := domain.GeoPointV2{longitude, latitude}
			if navigationPointOnLand(point) {
				return point
			}
		}
	}
	t.Fatal("Block Island land point not found in embedded chart")
	return domain.GeoPointV2{}
}

func nearestSafeOnLongitude(t *testing.T, origin domain.GeoPointV2, direction float64) domain.GeoPointV2 {
	t.Helper()
	for offset := .002; offset <= .20; offset += .002 {
		point := domain.GeoPointV2{origin[0] + direction*offset, origin[1]}
		if navigationPointSafe(point) {
			return point
		}
	}
	t.Fatalf("safe water not found from %v direction %.0f", origin, direction)
	return domain.GeoPointV2{}
}
