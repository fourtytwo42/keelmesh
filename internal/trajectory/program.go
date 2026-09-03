package trajectory

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const SegmentSeconds int64 = 10

func BuildRevision(mission domain.MissionWorkspaceV2, plan domain.FleetPlanV2, lease domain.FleetLeaseV2, revision int, createdTick, activationTick int64, key []byte) domain.TrajectoryRevisionV1 {
	result := domain.TrajectoryRevisionV1{Revision: revision, PlanID: plan.ID, PlanHash: plan.ContentHash, LeaseID: lease.ID, CreatedTick: createdTick, ActivationTick: activationTick, Segments: map[string][]domain.TrajectorySegmentV2{}}
	for _, assignment := range plan.Assignments {
		distanceM := routeDistanceM(assignment.Route)
		duration := int(math.Ceil(distanceM / math.Max(assignment.SpeedMPS, .1)))
		if duration < int(SegmentSeconds) {
			duration = int(SegmentSeconds)
		}
		count := int(math.Ceil(float64(duration) / float64(SegmentSeconds)))
		predecessor := ""
		segments := make([]domain.TrajectorySegmentV2, 0, count)
		for i := 0; i < count; i++ {
			startDistance := distanceM * float64(i) / float64(count)
			endDistance := distanceM * float64(i+1) / float64(count)
			segment := domain.TrajectorySegmentV2{
				SchemaVersion: 1, MissionID: mission.ID, PlanHash: plan.ContentHash, Revision: revision, VesselID: assignment.VesselID, Sequence: i,
				ActivationTick: activationTick + int64(i)*SegmentSeconds, ExpiryTick: activationTick + int64(i+1)*SegmentSeconds,
				Start: pointAtDistance(assignment.Route, startDistance), End: pointAtDistance(assignment.Route, endDistance),
				TargetSpeedMPS: assignment.SpeedMPS, MaximumSpeedMPS: mission.Constraints.MaximumSpeedMPS,
				CorridorRadiusM: math.Max(mission.Constraints.MinimumObjectSeparationM, 40), MaxLateralAdjustM: math.Min(math.Max(mission.Constraints.MinimumObjectSeparationM*.6, 20), 100),
				ScheduleToleranceS: 5, MinimumReserve: mission.Constraints.MinimumReserve, MinimumSeparationM: mission.Constraints.MinimumVesselSeparationM,
				MaximumUncertaintyM: mission.Constraints.MaximumPNTUncertaintyM, FailureBehavior: "safe_hold", PredecessorHash: predecessor,
			}
			segment.ContentHash = hash(segment)
			segment.Signature = sign(segment.ContentHash, key)
			segments = append(segments, segment)
			predecessor = segment.ContentHash
		}
		result.Segments[assignment.VesselID] = segments
		if count*int(SegmentSeconds) > result.DurationS {
			result.DurationS = count * int(SegmentSeconds)
		}
	}
	result.ContentHash = revisionHash(result)
	result.Signature = sign(result.ContentHash, key)
	return result
}

func NewProgram(missionID string, revision domain.TrajectoryRevisionV1, hotTapeSeconds int) domain.TrajectoryProgramV1 {
	if hotTapeSeconds < 60 {
		hotTapeSeconds = 60
	}
	program := domain.TrajectoryProgramV1{SchemaVersion: 1, MissionID: missionID, ActiveRevision: revision.Revision, MissionTickMS: revision.ActivationTick * 1000, HotTapeHorizonS: hotTapeSeconds, Revisions: map[int]domain.TrajectoryRevisionV1{revision.Revision: revision}, Cursors: map[string]domain.ExecutionCursorV1{}, LastAdjustments: map[string]domain.LocalAdjustmentV1{}}
	UpdateCursors(&program)
	program.ContentHash = programHash(program)
	return program
}

func AddPending(program *domain.TrajectoryProgramV1, revision domain.TrajectoryRevisionV1) {
	program.Revisions[revision.Revision] = revision
	program.PendingRevision = revision.Revision
	program.ActivationTick = revision.ActivationTick
	program.ContentHash = programHash(*program)
}

func Advance(program *domain.TrajectoryProgramV1, elapsedMS int64) (activated bool) {
	program.MissionTickMS += elapsedMS
	tick := program.MissionTickMS / 1000
	if program.PendingRevision > 0 && tick >= program.ActivationTick {
		program.ActiveRevision = program.PendingRevision
		program.PendingRevision = 0
		program.ActivationTick = 0
		activated = true
	}
	UpdateCursors(program)
	return activated
}

func CurrentSegment(program domain.TrajectoryProgramV1, vesselID string) (domain.TrajectorySegmentV2, bool) {
	revision, ok := program.Revisions[program.ActiveRevision]
	if !ok {
		return domain.TrajectorySegmentV2{}, false
	}
	segments := revision.Segments[vesselID]
	tick := program.MissionTickMS / 1000
	for _, segment := range segments {
		if tick >= segment.ActivationTick && tick < segment.ExpiryTick {
			return segment, true
		}
	}
	return domain.TrajectorySegmentV2{}, false
}

func Summary(program domain.TrajectoryProgramV1) domain.TrajectoryProgramSummaryV1 {
	revision := program.Revisions[program.ActiveRevision]
	total := 0
	for _, segments := range revision.Segments {
		total += len(segments)
	}
	return domain.TrajectoryProgramSummaryV1{MissionID: program.MissionID, ActiveRevision: program.ActiveRevision, PendingRevision: program.PendingRevision, ActivationTick: program.ActivationTick, MissionTick: program.MissionTickMS / 1000, DurationS: revision.DurationS, TotalSegments: total, HotTapeHorizonS: program.HotTapeHorizonS, Execution: program.Cursors, LastAdjustments: program.LastAdjustments, ContentHash: program.ContentHash}
}

func View(program domain.TrajectoryProgramV1) domain.TrajectoryProgramViewV1 {
	tick := program.MissionTickMS / 1000
	limit := tick + int64(program.HotTapeHorizonS)
	hot := map[string][]domain.TrajectorySegmentV2{}
	appendWindow := func(revision domain.TrajectoryRevisionV1, startsBefore, startsAtOrAfter int64) {
		for vesselID, segments := range revision.Segments {
			for _, segment := range segments {
				if segment.ExpiryTick <= tick || segment.ActivationTick >= limit {
					continue
				}
				if startsBefore > 0 && segment.ActivationTick >= startsBefore {
					continue
				}
				if startsAtOrAfter > 0 && segment.ActivationTick < startsAtOrAfter {
					continue
				}
				hot[vesselID] = append(hot[vesselID], segment)
			}
		}
	}
	active := program.Revisions[program.ActiveRevision]
	if program.PendingRevision > 0 {
		appendWindow(active, program.ActivationTick, 0)
		appendWindow(program.Revisions[program.PendingRevision], 0, program.ActivationTick)
	} else {
		appendWindow(active, 0, 0)
	}
	for vesselID := range hot {
		segments := hot[vesselID]
		// A node receives one monotonically ordered materialized tape even when
		// its tail crosses an armed revision boundary.
		sort.SliceStable(segments, func(i, j int) bool {
			return segments[i].ActivationTick < segments[j].ActivationTick
		})
		hot[vesselID] = segments
	}
	return domain.TrajectoryProgramViewV1{Summary: Summary(program), HotTape: hot}
}

func UpdateCursors(program *domain.TrajectoryProgramV1) {
	revision, ok := program.Revisions[program.ActiveRevision]
	if !ok {
		return
	}
	tick := program.MissionTickMS / 1000
	for vesselID, segments := range revision.Segments {
		sequence := len(segments)
		lifecycle := "completed"
		for i, segment := range segments {
			if tick < segment.ExpiryTick {
				sequence = i
				lifecycle = "executing"
				break
			}
		}
		remaining := max(0, revision.DurationS-int(tick-revision.ActivationTick))
		program.Cursors[vesselID] = domain.ExecutionCursorV1{VesselID: vesselID, Revision: revision.Revision, Sequence: sequence, MissionTick: tick, HotTapeDepthS: min(program.HotTapeHorizonS, remaining), ProgramRemainingS: remaining, Lifecycle: lifecycle}
	}
}

func ValidateRevision(revision domain.TrajectoryRevisionV1, key []byte) bool {
	if revision.ContentHash != revisionHash(revision) || !hmac.Equal([]byte(revision.Signature), []byte(sign(revision.ContentHash, key))) {
		return false
	}
	for _, segments := range revision.Segments {
		predecessor := ""
		for _, segment := range segments {
			if segment.PredecessorHash != predecessor || segment.ContentHash != hash(segment) || !hmac.Equal([]byte(segment.Signature), []byte(sign(segment.ContentHash, key))) {
				return false
			}
			predecessor = segment.ContentHash
		}
	}
	return true
}

func pointAtDistance(route []domain.GeoPointV2, targetM float64) domain.GeoPointV2 {
	if len(route) == 0 {
		return domain.GeoPointV2{}
	}
	remaining := targetM
	for i := 1; i < len(route); i++ {
		leg := distanceM(route[i-1], route[i])
		if remaining <= leg || i == len(route)-1 {
			fraction := 0.0
			if leg > 0 {
				fraction = math.Min(1, remaining/leg)
			}
			return domain.GeoPointV2{route[i-1][0] + (route[i][0]-route[i-1][0])*fraction, route[i-1][1] + (route[i][1]-route[i-1][1])*fraction}
		}
		remaining -= leg
	}
	return route[len(route)-1]
}

func routeDistanceM(route []domain.GeoPointV2) float64 {
	total := 0.0
	for i := 1; i < len(route); i++ {
		total += distanceM(route[i-1], route[i])
	}
	return total
}

func distanceM(a, b domain.GeoPointV2) float64 {
	lat := (a[1] + b[1]) * math.Pi / 360
	dx := (b[0] - a[0]) * 111_000 * math.Cos(lat)
	dy := (b[1] - a[1]) * 111_000
	return math.Hypot(dx, dy)
}

func hash(value domain.TrajectorySegmentV2) string {
	value.ContentHash, value.Signature = "", ""
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func revisionHash(value domain.TrajectoryRevisionV1) string {
	value.ContentHash, value.Signature = "", ""
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func programHash(value domain.TrajectoryProgramV1) string {
	value.ContentHash = ""
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sign(hash string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(hash))
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}
