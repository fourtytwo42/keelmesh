package fleetops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/trajectory"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Message }

type Mutation struct {
	RequestID       string `json:"request_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	ExpectedVersion int64  `json:"expected_version"`
}
type CreateGroupRequest struct {
	Mutation
	Name      string   `json:"name"`
	Color     string   `json:"color"`
	Pattern   string   `json:"pattern"`
	MemberIDs []string `json:"member_ids"`
}
type PatchGroupRequest struct {
	Mutation
	Name                   *string            `json:"name"`
	Color                  *string            `json:"color"`
	Pattern                *string            `json:"pattern"`
	MemberIDs              *[]string          `json:"member_ids"`
	Formation              *string            `json:"formation"`
	FormationSpacingM      *float64           `json:"formation_spacing_m"`
	AssemblyPoint          *domain.GeoPointV2 `json:"assembly_point"`
	ClearAssemblyPoint     bool               `json:"clear_assembly_point"`
	UseFirstMemberAssembly bool               `json:"use_first_member_assembly"`
}
type MoveGroupMemberRequest struct {
	Mutation
	VesselID string `json:"vessel_id"`
}
type PatchVesselRequest struct {
	Mutation
	Callsign string `json:"callsign"`
}
type CreateCollectionRequest struct {
	Mutation
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}
type PatchCollectionRequest struct {
	Mutation
	Name      *string   `json:"name"`
	MemberIDs *[]string `json:"member_ids"`
}
type CreateMissionRequest struct {
	Mutation
	Name       string   `json:"name"`
	NamingMode string   `json:"naming_mode"`
	Objective  string   `json:"objective"`
	TargetIDs  []string `json:"target_ids"`
}
type PatchMissionRequest struct {
	Mutation
	Name        *string                 `json:"name"`
	Objective   *string                 `json:"objective"`
	Status      *string                 `json:"status"`
	Formation   *string                 `json:"formation"`
	TargetIDs   *[]string               `json:"target_ids"`
	Constraints *domain.ConstraintSetV2 `json:"constraints"`
}
type GeometryRequest struct {
	Mutation
	IncludedAreas   [][][]float64              `json:"included_areas"`
	ExclusionAreas  [][][]float64              `json:"exclusion_areas"`
	Waypoints       []domain.GeoPointV2        `json:"waypoints"`
	WaypointDetails []domain.MissionWaypointV2 `json:"waypoint_details"`
	POIs            []domain.MissionPOIV2      `json:"pois"`
}
type CompileRequest struct {
	Mutation
	Text         string              `json:"text"`
	TargetIDs    []string            `json:"target_ids"`
	GuidanceKind string              `json:"guidance_kind"`
	Waypoints    []domain.GeoPointV2 `json:"waypoints"`
	Formation    string              `json:"formation"`
}
type PlansRequest struct {
	Mutation
	DraftID string `json:"draft_id"`
}
type PlanActionRequest struct {
	Mutation
	PlanHash   string `json:"plan_hash"`
	OperatorID string `json:"operator_id"`
	LeaseID    string `json:"lease_id"`
}

type Manager struct {
	mu                    sync.RWMutex
	persistenceMu         sync.Mutex
	persistenceGeneration atomic.Uint64
	logger                *slog.Logger
	databaseURL           string
	secret                []byte
	fleetVersion          int64
	vessels               map[string]domain.VesselProfileV2
	groups                map[string]domain.OperationalGroupV2
	collections           map[string]domain.SavedCollectionV2
	missions              map[string]domain.MissionWorkspaceV2
	drafts                map[string]domain.CommandDraftV2
	plans                 map[string]domain.FleetPlanV2
	leases                map[string]domain.FleetLeaseV2
	idempotency           map[string]string
	startedPlans          map[string]string
	programs              map[string]domain.TrajectoryProgramV1
	simTickMS             int64
}

var callsigns = []string{"Gannet", "Osprey", "Tern", "Petrel", "Shearwater", "Cormorant", "Harrier", "Kite", "Merlin", "Plover", "Skua", "Fulmar", "Albatross", "Razorbill", "Puffin", "Heron", "Kittiwake", "Curlew", "Jaeger", "Avocet", "Sanderling", "Grebe", "Dunlin", "Egret", "Bittern", "Sandpiper", "Stormbird", "Kingfisher", "Loon", "Murre", "Nighthawk", "Pelican", "Rail", "Sparrowhawk", "Turnstone", "Whimbrel", "Auk", "Bunting", "Caspian", "Diver", "Eider", "Frigate", "Godwit", "Hobby", "Ibis", "Junco", "Lapwing", "Merganser"}
var groupNames = []string{"Watch Shoal", "Bay Lantern", "Sakonnet", "Block Guard", "Brenton", "Narragansett", "Point Judith", "Ocean State"}
var groupCodes = []string{"WS", "BL", "SK", "BG", "BR", "NG", "PJ", "OS"}
var groupColors = []string{"#e9a93f", "#62c5a8", "#d86f5f", "#b895d8", "#7eb4df", "#d2c05d", "#df8fb0", "#8fca72"}
var missionNames = []string{"Harbor Lantern", "Silver Wake", "Tidal Compass", "Coastal Sentinel", "Sound Guardian", "Northstar Passage", "Sable Current", "Watch Meridian", "Seaward Beacon", "Iron Gull", "Quiet Horizon", "Amber Shoal"}
var patterns = []string{"solid", "diagonal", "dots", "crosshatch", "vertical", "rings", "dash", "chevron"}
var spawnCenters = []domain.GeoPointV2{{-71.385, 41.43}, {-71.30, 41.43}, {-71.24, 41.45}, {-71.42, 41.39}, {-71.33, 41.37}, {-71.23, 41.35}, {-71.47, 41.25}, {-71.30, 41.20}}
var legacySpawnCenters = []domain.GeoPointV2{{-71.375, 41.49}, {-71.315, 41.47}, {-71.24, 41.45}, {-71.43, 41.39}, {-71.33, 41.37}, {-71.23, 41.35}, {-71.47, 41.25}, {-71.30, 41.20}}
var reserveAfterLabel = regexp.MustCompile(`(?i)(?:reserve|battery)[^0-9]{0,24}([0-9]{1,3})\s*%`)
var reserveBeforeLabel = regexp.MustCompile(`(?i)([0-9]{1,3})\s*%\s*(?:reserve|battery)`)
var coastalDistance = regexp.MustCompile(`(?i)(?:within|inside|no more than|max(?:imum)?(?: distance)?(?: of)?)\s*([0-9]+(?:\.[0-9]+)?)\s*(?:nm|nmi|nautical\s*miles?)`)
var relativeTravelDistance = regexp.MustCompile(`(?i)\b(?:travel|go|proceed|sail|run|head|move)?\s*([0-9]+(?:\.[0-9]+)?)\s*(nm|nmi|nautical\s*miles?|km|kilometers?|mi|miles?)\b`)
var explicitHeading = regexp.MustCompile(`(?i)\b(?:heading|bearing|course)\s*(?:of|to|at)?\s*([0-9]{1,3}(?:\.[0-9]+)?)\s*(?:deg(?:rees?)?|°)?\b`)
var waypointColor = regexp.MustCompile(`(?i)\b(amber|red|green|cyan|blue|violet|purple|white)\s+(?:waypoints?|route|markers?)\b`)
var numberedMissionName = regexp.MustCompile(`(?i)^(mission|voyage)\s+([0-9]+)$`)

// coastalDepthRoutes are deterministic samples from the packaged NOAA ETOPO
// 5 m contour. They let natural-language coastal missions use the same local
// bathymetry shown on the map without requiring network or model availability.
var coastalDepthRoutes = [][]domain.GeoPointV2{
	{{-71.34792, 41.47334}, {-71.35312, 41.46875}, {-71.35185, 41.45208}, {-71.34792, 41.44572}, {-71.33169, 41.45208}, {-71.31458, 41.45531}, {-71.29792, 41.46140}},
	{{-71.35185, 41.45208}, {-71.34792, 41.44572}, {-71.33169, 41.45208}, {-71.31458, 41.45531}, {-71.29792, 41.46140}, {-71.28909, 41.46875}, {-71.28125, 41.48033}},
	{{-71.26636, 41.48542}, {-71.26458, 41.48565}, {-71.26348, 41.48542}, {-71.24792, 41.47723}, {-71.23742, 41.48542}, {-71.23125, 41.50600}, {-71.21972, 41.51875}},
	{{-71.48125, 41.36472}, {-71.46781, 41.38542}, {-71.46458, 41.38772}, {-71.45566, 41.40208}, {-71.45011, 41.43542}, {-71.44792, 41.43737}, {-71.43125, 41.44532}},
	{{-71.34792, 41.47334}, {-71.35312, 41.46875}, {-71.35185, 41.45208}, {-71.34792, 41.44572}, {-71.33169, 41.45208}, {-71.31458, 41.45531}, {-71.29792, 41.46140}},
	{{-71.19792, 41.48613}, {-71.19686, 41.48542}, {-71.19032, 41.46875}, {-71.18125, 41.46228}, {-71.16458, 41.46952}, {-71.14792, 41.47542}, {-71.13675, 41.48542}},
	{{-71.54941, 41.16875}, {-71.56285, 41.18542}, {-71.55508, 41.20208}, {-71.55360, 41.21875}, {-71.56458, 41.22894}},
	{{-71.56458, 41.13877}, {-71.54792, 41.14806}, {-71.54494, 41.15208}, {-71.54792, 41.16451}, {-71.54941, 41.16875}, {-71.56285, 41.18542}, {-71.55508, 41.20208}},
}
var coastalRouteNames = []string{
	"Newport West Passage",
	"Jamestown–Newport Reach",
	"Sakonnet North Reach",
	"Point Judith Approach",
	"Narragansett Coastal Watch",
	"Sakonnet East Reach",
	"Block Island North Watch",
	"Block Island West Watch",
}

func New(databaseURL string, logger *slog.Logger) *Manager {
	m := &Manager{logger: logger, databaseURL: databaseURL, secret: []byte("keelmesh-m6-runtime-authority"), fleetVersion: 1, vessels: map[string]domain.VesselProfileV2{}, groups: map[string]domain.OperationalGroupV2{}, collections: map[string]domain.SavedCollectionV2{}, missions: map[string]domain.MissionWorkspaceV2{}, drafts: map[string]domain.CommandDraftV2{}, plans: map[string]domain.FleetPlanV2{}, leases: map[string]domain.FleetLeaseV2{}, idempotency: map[string]string{}, startedPlans: map[string]string{}, programs: map[string]domain.TrajectoryProgramV1{}}
	m.seed()
	return m
}

func (m *Manager) Run(ctx context.Context) {
	m.loadPersistent(ctx)
	// Persist any deterministic spawn-layout migration after all retained fleet
	// identities, groups, collections, and mission workspaces have loaded.
	m.persistAsync()
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.tick()
		}
	}
}

func classFor(slot int) domain.VesselClassV2 {
	switch {
	case slot < 3:
		return domain.VesselClassV2{ID: "kestrel", Name: "Kestrel", Role: "agile scout", MaxSpeedMPS: 3.2, MinimumReserve: .22, EnduranceHours: 8}
	case slot < 5:
		return domain.VesselClassV2{ID: "mariner", Name: "Mariner", Role: "general-purpose platform", MaxSpeedMPS: 2.5, MinimumReserve: .25, EnduranceHours: 14}
	default:
		return domain.VesselClassV2{ID: "atlas", Name: "Atlas", Role: "endurance, support, and communications relay", MaxSpeedMPS: 1.9, MinimumReserve: .30, EnduranceHours: 30, CommunicationsRole: true}
	}
}

func (m *Manager) seed() {
	for g := 0; g < 8; g++ {
		gid := fmt.Sprintf("group-%02d", g+1)
		members := make([]string, 0, 6)
		for slot := 0; slot < 6; slot++ {
			idx := g*6 + slot
			id := fmt.Sprintf("00000000-0000-6000-8000-%012d", idx+1)
			members = append(members, id)
			class := classFor(slot)
			// Keep all six members legible at the default regional zoom. These are
			// real simulated positions, not screen-space offsets, so selection,
			// routing, and distance calculations all agree with the map.
			p := spawnPoint(spawnCenters, g, slot, .022, .028)
			env := environmentAt(p, float64(idx))
			m.vessels[id] = domain.VesselProfileV2{SchemaVersion: 2, ID: id, Designation: fmt.Sprintf("KM-%03d", 214+idx), Callsign: callsigns[idx], DisplayName: fmt.Sprintf("%s (KM-%03d)", callsigns[idx], 214+idx), Class: class, GroupID: gid, GroupCode: groupCodes[g], GroupColor: groupColors[g], GroupPattern: patterns[g], Available: true, DecisionCapable: true, Telemetry: domain.VesselTelemetryV2{Position: p, HeadingDeg: float64((idx * 37) % 360), SpeedMPS: .4 + float64(idx%5)*.11, Reserve: .96 - float64(idx%9)*.025, ProjectedReserve: .89 - float64(idx%9)*.025, Mode: "patrol", Health: "nominal", PNTIntegrity: "trusted", UncertaintyM: 4 + float64(idx%5), TapeDepthSeconds: 60, Environment: env}}
		}
		assembly := m.vessels[members[0]].Telemetry.Position
		m.groups[gid] = domain.OperationalGroupV2{SchemaVersion: 2, ID: gid, Code: groupCodes[g], Name: groupNames[g], Color: groupColors[g], Pattern: patterns[g], MemberIDs: members, Formation: "column", FormationSpacingM: 60, AssemblyPoint: &assembly, AssemblySource: "first-member", DecisionPolicy: "lowest_reachable_capable_id", DecisionNodeID: members[0], DecisionEpoch: 1, FallbackPolicy: "safe_hold_then_signal_seek_then_return_home_if_authorized", Revision: 1}
	}
	all := make([]string, 0, 48)
	for id := range m.vessels {
		all = append(all, id)
	}
	sort.Strings(all)
	m.collections["collection-relays"] = domain.SavedCollectionV2{SchemaVersion: 2, ID: "collection-relays", Name: "Atlas relay watch", MemberIDs: filterIDs(all, func(v domain.VesselProfileV2) bool { return v.Class.ID == "atlas" }, m.vessels), Revision: 1}
}

func spawnPoint(centers []domain.GeoPointV2, group, slot int, longitudeStep, latitudeStep float64) domain.GeoPointV2 {
	return domain.GeoPointV2{centers[group][0] + float64(slot%3-1)*longitudeStep, centers[group][1] + float64(slot/3)*latitudeStep}
}

func migrateLegacySpawn(v domain.VesselProfileV2, seeded domain.VesselProfileV2) domain.VesselProfileV2 {
	var designation int
	if _, err := fmt.Sscanf(v.Designation, "KM-%d", &designation); err != nil {
		return v
	}
	idx := designation - 214
	if idx < 0 || idx >= 48 {
		return v
	}
	group, slot := idx/6, idx%6
	legacyCompact := spawnPoint(legacySpawnCenters, group, slot, .008, .007)
	legacyReadable := spawnPoint(legacySpawnCenters, group, slot, .022, .028)
	if samePoint(v.Telemetry.Position, legacyCompact) || samePoint(v.Telemetry.Position, legacyReadable) {
		v.Telemetry.Position = seeded.Telemetry.Position
		v.Telemetry.Environment = seeded.Telemetry.Environment
	}
	return v
}

func samePoint(a, b domain.GeoPointV2) bool {
	// The simulator may advance a legacy seed by a few metres before retained
	// state loads. Keep the migration neighborhood tight enough that a vessel
	// which has actually departed its initial cell is never repositioned.
	return math.Abs(a[0]-b[0]) < .003 && math.Abs(a[1]-b[1]) < .003
}

func clearVesselGroup(vessel *domain.VesselProfileV2) {
	vessel.GroupID = ""
	vessel.GroupCode = ""
	vessel.GroupColor = "#737973"
	vessel.GroupPattern = "unassigned"
}

func withinMapBounds(point domain.GeoPointV2) bool {
	return point[0] >= -71.62 && point[0] <= -71.08 && point[1] >= 41.08 && point[1] <= 41.62
}

func validFormation(value string) bool {
	switch value {
	case "column", "line_abreast", "wedge", "echelon_left", "echelon_right", "parallel_columns", "dispersed_screen", "ring", "search_grid":
		return true
	default:
		return false
	}
}

func (m *Manager) nextGroupCodeLocked() string {
	for ordinal := 1; ; ordinal++ {
		candidate := fmt.Sprintf("C%02d", ordinal)
		used := false
		for _, group := range m.groups {
			if group.Code == candidate {
				used = true
				break
			}
		}
		if !used {
			return candidate
		}
	}
}

func (m *Manager) defaultAssemblyPointLocked(color string, members []string) (*domain.GeoPointV2, string, string) {
	wanted := nearestWaypointColor(color)
	missions := make([]domain.MissionWorkspaceV2, 0, len(m.missions))
	for _, mission := range m.missions {
		missions = append(missions, mission)
	}
	sort.Slice(missions, func(i, j int) bool { return missions[i].UpdatedAt.After(missions[j].UpdatedAt) })
	for _, mission := range missions {
		for _, waypoint := range mission.Geometry.WaypointDetails {
			if normalizeWaypointColor(waypoint.Color) == wanted {
				point := waypoint.Position
				return &point, "mission:" + mission.ID, waypoint.ID
			}
		}
	}
	if len(members) > 0 {
		if vessel, ok := m.vessels[members[0]]; ok {
			point := vessel.Telemetry.Position
			return &point, "first-member", ""
		}
	}
	return nil, "", ""
}

func nearestWaypointColor(value string) string {
	palette := map[string][3]int{
		"amber": {230, 166, 59}, "red": {227, 110, 98}, "green": {98, 197, 142},
		"cyan": {89, 189, 209}, "violet": {184, 149, 216}, "white": {236, 232, 220},
	}
	var r, g, b int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "#%02x%02x%02x", &r, &g, &b); err != nil {
		return "amber"
	}
	best, score := "amber", math.MaxFloat64
	for name, rgb := range palette {
		distance := math.Pow(float64(r-rgb[0]), 2) + math.Pow(float64(g-rgb[1]), 2) + math.Pow(float64(b-rgb[2]), 2)
		if distance < score {
			best, score = name, distance
		}
	}
	return best
}

func filterIDs(ids []string, pred func(domain.VesselProfileV2) bool, vs map[string]domain.VesselProfileV2) []string {
	out := []string{}
	for _, id := range ids {
		if pred(vs[id]) {
			out = append(out, id)
		}
	}
	return out
}
func environmentAt(p domain.GeoPointV2, phase float64) domain.EnvironmentV2 {
	return domain.EnvironmentV2{WindSpeedMPS: 5.4 + math.Sin(p[0]*18+phase)*1.3, WindDirectionDeg: 218 + math.Mod(phase*7, 34), CurrentSpeedMPS: .42 + math.Cos(p[1]*21+phase)*.12, CurrentDirectionDeg: 66 + math.Mod(phase*3, 22), WaveHeightM: .72 + math.Sin(phase*.4)*.14, WaterTemperatureC: 18.1 + math.Cos(phase*.2)*.5, FixtureAt: "2026-08-28T12:00:00Z", Label: "NOAA-derived simulation fixture", SourceIDs: []string{"NDBC-44097", "NDBC-NAQR1", "NOAA-COOPS-NARRAGANSETT"}}
}

func (m *Manager) Snapshot() domain.FleetSnapshotV2 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotLocked()
}
func (m *Manager) snapshotLocked() domain.FleetSnapshotV2 {
	vs := make([]domain.VesselProfileV2, 0, len(m.vessels))
	for _, v := range m.vessels {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].Designation < vs[j].Designation })
	gs := make([]domain.OperationalGroupV2, 0, len(m.groups))
	for _, g := range m.groups {
		g = m.groupDecisionSnapshotLocked(g)
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i].Code < gs[j].Code })
	cs := make([]domain.SavedCollectionV2, 0, len(m.collections))
	for _, c := range m.collections {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	ms := make([]domain.MissionWorkspaceV2, 0, len(m.missions))
	for _, v := range m.missions {
		if program, ok := m.programs[v.ID]; ok {
			summary := trajectory.Summary(program)
			v.Trajectory = &summary
		}
		ms = append(ms, v)
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].UpdatedAt.After(ms[j].UpdatedAt) })
	return domain.FleetSnapshotV2{SchemaVersion: 2, FleetVersion: m.fleetVersion, GeneratedAt: time.Now().UTC(), Vessels: vs, Groups: gs, Collections: cs, Missions: ms, Environment: environmentAt(domain.GeoPointV2{-71.34, 41.32}, 0), Map: map[string]any{"name": "Narragansett Bay & Rhode Island Sound", "center": domain.GeoPointV2{-71.34, 41.34}, "bounds": [][]float64{{-71.62, 41.08}, {-71.08, 41.62}}, "fixture": true, "navigation_warning": "Simulation only — not for navigation"}}
}

func (m *Manager) Vessel(id string) (domain.VesselProfileV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vessels[id]
	if !ok {
		return v, &Error{"VESSEL_NOT_FOUND", "Vessel not found."}
	}
	return v, nil
}

func (m *Manager) TrajectoryProgram(id string) (domain.TrajectoryProgramViewV1, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	program, ok := m.programs[id]
	if !ok {
		return domain.TrajectoryProgramViewV1{}, &Error{"TRAJECTORY_NOT_FOUND", "Mission does not have an active trajectory program."}
	}
	return trajectory.View(program), nil
}
func (m *Manager) PatchVessel(id string, req PatchVesselRequest) (domain.VesselProfileV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.vessels[id]
	if !ok {
		return v, &Error{"VESSEL_NOT_FOUND", "Vessel not found."}
	}
	if err := m.check(req.Mutation, m.fleetVersion); err != nil {
		return v, err
	}
	name := strings.TrimSpace(req.Callsign)
	if len(name) < 2 || len(name) > 32 {
		return v, &Error{"INVALID_VESSEL_NAME", "Vessel callsign must be between 2 and 32 characters."}
	}
	for otherID, other := range m.vessels {
		if otherID != id && strings.EqualFold(other.Callsign, name) {
			return v, &Error{"VESSEL_NAME_CONFLICT", "That vessel callsign is already in use."}
		}
	}
	v.Callsign = name
	v.DisplayName = fmt.Sprintf("%s (%s)", name, v.Designation)
	m.vessels[id] = v
	m.fleetVersion++
	m.persistAsync()
	return v, nil
}
func (m *Manager) Reachability(id string) (domain.ReachabilityV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vessels[id]
	if !ok {
		return domain.ReachabilityV2{}, &Error{"VESSEL_NOT_FOUND", "Vessel not found."}
	}
	r := domain.ReachabilityV2{SchemaVersion: 2, VesselID: id, Authority: "mission-scoped authority"}
	g, grouped := m.groups[v.GroupID]
	if !grouped {
		r.Authority = "unassigned · no group movement authority"
		return r, nil
	}
	for i, peer := range g.MemberIDs {
		if peer == id {
			continue
		}
		p := domain.ReachabilityPathV2{VesselID: peer, State: "reachable", LatencyMS: 18 + float64(i)*3}
		if i%3 == 0 {
			p.Hops = []string{id, atlasID(g, m.vessels), peer}
			p.Underlay = []string{"halow", "halow"}
			r.RelayedPeers = append(r.RelayedPeers, p)
		} else {
			p.Hops = []string{id, peer}
			p.Underlay = []string{"starlink"}
			r.DirectPeers = append(r.DirectPeers, p)
		}
	}
	for _, og := range m.groups {
		if og.ID != g.ID && len(og.MemberIDs) > 0 && len(r.ExternalPeers) < 3 {
			peer := og.MemberIDs[len(og.MemberIDs)-1]
			r.ExternalPeers = append(r.ExternalPeers, domain.ReachabilityPathV2{VesselID: peer, State: "reachable", Hops: []string{id, atlasID(g, m.vessels), peer}, Underlay: []string{"halow", "peer-starlink"}, LatencyMS: 54})
		}
	}
	return r, nil
}
func atlasID(g domain.OperationalGroupV2, vs map[string]domain.VesselProfileV2) string {
	for _, id := range g.MemberIDs {
		if vs[id].Class.ID == "atlas" {
			return id
		}
	}
	return g.MemberIDs[0]
}

// groupDecisionSnapshotLocked elects a predictable advisory decision node.
// It grants no additional mission authority: every adjustment still has to fit
// the signed trajectory envelope. A future deployment can mark only GPU-bearing
// vessels DecisionCapable without changing election semantics.
func (m *Manager) groupDecisionSnapshotLocked(g domain.OperationalGroupV2) domain.OperationalGroupV2 {
	if g.DecisionPolicy == "" {
		g.DecisionPolicy = "lowest_reachable_capable_id"
	}
	if g.FallbackPolicy == "" {
		g.FallbackPolicy = "safe_hold_then_signal_seek_then_return_home_if_authorized"
	}
	candidates := make([]string, 0, len(g.MemberIDs))
	for _, id := range g.MemberIDs {
		if vessel, ok := m.vessels[id]; ok && vessel.Available && vessel.DecisionCapable {
			candidates = append(candidates, id)
		}
	}
	sort.Strings(candidates)
	previous := g.DecisionNodeID
	if len(candidates) == 0 {
		g.DecisionNodeID = ""
	} else {
		g.DecisionNodeID = candidates[0]
	}
	if g.DecisionEpoch == 0 {
		g.DecisionEpoch = 1
	} else if previous != "" && previous != g.DecisionNodeID {
		g.DecisionEpoch++
	}
	return g
}

func (m *Manager) CreateGroup(req CreateGroupRequest) (domain.OperationalGroupV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.Mutation, m.fleetVersion); err != nil {
		return domain.OperationalGroupV2{}, err
	}
	if err := m.validTargets(req.MemberIDs); err != nil {
		return domain.OperationalGroupV2{}, err
	}
	if conflicts := m.conflicts(req.MemberIDs, ""); len(conflicts) > 0 {
		return domain.OperationalGroupV2{}, &Error{"ACTIVE_MISSION_REPLAN_REQUIRED", "End or re-plan active movement authority before changing operational groups."}
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 48 {
		return domain.OperationalGroupV2{}, &Error{"INVALID_GROUP_NAME", "Group name must be between 2 and 48 characters."}
	}
	for _, existing := range m.groups {
		if strings.EqualFold(existing.Name, name) {
			return domain.OperationalGroupV2{}, &Error{"GROUP_NAME_CONFLICT", "That operational group name is already in use."}
		}
	}
	id := "group-custom-" + shortHash(req.IdempotencyKey)
	members := unique(req.MemberIDs)
	assembly, source, waypointID := m.defaultAssemblyPointLocked(req.Color, members)
	g := domain.OperationalGroupV2{SchemaVersion: 2, ID: id, Code: m.nextGroupCodeLocked(), Name: name, Color: req.Color, Pattern: req.Pattern, MemberIDs: members, Formation: "column", FormationSpacingM: 60, AssemblyPoint: assembly, AssemblySource: source, AssemblyWaypointID: waypointID, DecisionPolicy: "lowest_reachable_capable_id", DecisionEpoch: 1, FallbackPolicy: "safe_hold_then_signal_seek_then_return_home_if_authorized", Revision: 1}
	g = m.groupDecisionSnapshotLocked(g)
	for _, vid := range g.MemberIDs {
		v := m.vessels[vid]
		if old, ok := m.groups[v.GroupID]; ok {
			old.MemberIDs = remove(old.MemberIDs, vid)
			old.Revision++
			m.groups[old.ID] = old
		}
		v.GroupID = id
		v.GroupCode = g.Code
		v.GroupColor = g.Color
		v.GroupPattern = g.Pattern
		m.vessels[vid] = v
	}
	m.groups[id] = g
	m.fleetVersion++
	m.persistAsync()
	return g, nil
}
func (m *Manager) PatchGroup(id string, req PatchGroupRequest) (domain.OperationalGroupV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[id]
	if !ok {
		return g, &Error{"GROUP_NOT_FOUND", "Group not found."}
	}
	if err := m.check(req.Mutation, g.Revision); err != nil {
		return g, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len(name) < 2 || len(name) > 48 {
			return g, &Error{"INVALID_GROUP_NAME", "Group name must be between 2 and 48 characters."}
		}
		for otherID, existing := range m.groups {
			if otherID != id && strings.EqualFold(existing.Name, name) {
				return g, &Error{"GROUP_NAME_CONFLICT", "That operational group name is already in use."}
			}
		}
		g.Name = name
	}
	if req.Color != nil {
		g.Color = *req.Color
	}
	if req.Pattern != nil {
		g.Pattern = *req.Pattern
	}
	if req.MemberIDs != nil {
		if err := m.validTargets(*req.MemberIDs); err != nil {
			return g, err
		}
		nextMembers := unique(*req.MemberIDs)
		if !sameMembers(g.MemberIDs, nextMembers) {
			if conflicts := m.conflicts(append(cloneStrings(g.MemberIDs), nextMembers...), ""); len(conflicts) > 0 {
				return g, &Error{"ACTIVE_MISSION_REPLAN_REQUIRED", "End or re-plan active movement authority before changing operational groups."}
			}
			for _, prior := range g.MemberIDs {
				if !contains(nextMembers, prior) {
					vessel := m.vessels[prior]
					clearVesselGroup(&vessel)
					m.vessels[prior] = vessel
				}
			}
			for _, vid := range nextMembers {
				vessel := m.vessels[vid]
				if vessel.GroupID != id {
					if old, exists := m.groups[vessel.GroupID]; exists {
						old.MemberIDs = remove(old.MemberIDs, vid)
						old.Revision++
						m.groups[old.ID] = old
					}
				}
			}
			g.MemberIDs = nextMembers
		}
	}
	if req.Formation != nil {
		if !validFormation(*req.Formation) {
			return g, &Error{"INVALID_FORMATION", "Unsupported station-keeping formation."}
		}
		g.Formation = *req.Formation
	}
	if req.FormationSpacingM != nil {
		if *req.FormationSpacingM < 15 || *req.FormationSpacingM > 1000 {
			return g, &Error{"INVALID_FORMATION_SPACING", "Formation spacing must be between 15 and 1,000 metres."}
		}
		g.FormationSpacingM = *req.FormationSpacingM
	}
	if req.ClearAssemblyPoint {
		g.AssemblyPoint = nil
		g.AssemblySource = ""
		g.AssemblyWaypointID = ""
	}
	if req.AssemblyPoint != nil {
		point := *req.AssemblyPoint
		if !withinMapBounds(point) {
			return g, &Error{"INVALID_ASSEMBLY_POINT", "Assembly point must be inside the local water-simulation bounds."}
		}
		g.AssemblyPoint = &point
		g.AssemblySource = "operator"
		g.AssemblyWaypointID = ""
	}
	if req.UseFirstMemberAssembly {
		if len(g.MemberIDs) == 0 {
			return g, &Error{"GROUP_EMPTY", "Add a vessel before using its position as the assembly point."}
		}
		point := m.vessels[g.MemberIDs[0]].Telemetry.Position
		g.AssemblyPoint = &point
		g.AssemblySource = "first-member"
		g.AssemblyWaypointID = ""
	}
	g.Revision++
	m.groups[id] = g
	for _, vid := range g.MemberIDs {
		v := m.vessels[vid]
		v.GroupID = id
		v.GroupCode = g.Code
		v.GroupColor = g.Color
		v.GroupPattern = g.Pattern
		m.vessels[vid] = v
	}
	m.fleetVersion++
	m.persistAsync()
	return g, nil
}
func (m *Manager) MoveGroupMember(id string, req MoveGroupMemberRequest) (domain.OperationalGroupV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "unassigned" {
		vessel, ok := m.vessels[req.VesselID]
		if !ok {
			return domain.OperationalGroupV2{}, &Error{"VESSEL_NOT_FOUND", "Vessel not found: " + req.VesselID}
		}
		if err := m.check(req.Mutation, m.fleetVersion); err != nil {
			return domain.OperationalGroupV2{}, err
		}
		if conflicts := m.conflicts([]string{req.VesselID}, ""); len(conflicts) > 0 {
			return domain.OperationalGroupV2{}, &Error{"ACTIVE_MISSION_REPLAN_REQUIRED", "End or re-plan active movement authority before unassigning this vessel."}
		}
		if source, exists := m.groups[vessel.GroupID]; exists {
			source.MemberIDs = remove(source.MemberIDs, vessel.ID)
			source.Revision++
			m.groups[source.ID] = source
		}
		clearVesselGroup(&vessel)
		m.vessels[vessel.ID] = vessel
		m.fleetVersion++
		m.persistAsync()
		return domain.OperationalGroupV2{SchemaVersion: 2, ID: "unassigned", Name: "Unassigned", MemberIDs: []string{vessel.ID}, Revision: m.fleetVersion}, nil
	}
	destination, ok := m.groups[id]
	if !ok {
		return destination, &Error{"GROUP_NOT_FOUND", "Destination group not found."}
	}
	if err := m.check(req.Mutation, destination.Revision); err != nil {
		return destination, err
	}
	vessel, ok := m.vessels[req.VesselID]
	if !ok {
		return destination, &Error{"VESSEL_NOT_FOUND", "Vessel not found: " + req.VesselID}
	}
	if vessel.GroupID == id {
		return destination, nil
	}
	if conflicts := m.conflicts([]string{req.VesselID}, ""); len(conflicts) > 0 {
		return destination, &Error{"ACTIVE_MISSION_REPLAN_REQUIRED", "End or re-plan active movement authority before changing this vessel's operational group."}
	}
	source, sourceExists := m.groups[vessel.GroupID]
	if sourceExists {
		source.MemberIDs = remove(source.MemberIDs, req.VesselID)
		source.Revision++
		m.groups[source.ID] = source
	}
	destination.MemberIDs = unique(append(destination.MemberIDs, req.VesselID))
	destination.Revision++
	vessel.GroupID = destination.ID
	vessel.GroupCode = destination.Code
	vessel.GroupColor = destination.Color
	vessel.GroupPattern = destination.Pattern
	m.groups[destination.ID] = destination
	m.vessels[vessel.ID] = vessel
	m.fleetVersion++
	m.persistAsync()
	return destination, nil
}
func (m *Manager) DeleteGroup(id string, req Mutation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[id]
	if !ok {
		return &Error{"GROUP_NOT_FOUND", "Group not found."}
	}
	if err := m.check(req, g.Revision); err != nil {
		return err
	}
	if conflicts := m.conflicts(g.MemberIDs, ""); len(conflicts) > 0 {
		return &Error{"ACTIVE_MISSION_REPLAN_REQUIRED", "End or re-plan active movement authority before dissolving this group."}
	}
	m.persistenceGeneration.Add(1)
	if err := m.deleteGroupPersistence(id); err != nil {
		return &Error{"GROUP_PERSISTENCE_FAILED", "Group deletion could not be committed. Nothing was deleted."}
	}
	for _, vesselID := range g.MemberIDs {
		vessel := m.vessels[vesselID]
		clearVesselGroup(&vessel)
		m.vessels[vesselID] = vessel
	}
	delete(m.groups, id)
	m.fleetVersion++
	m.persistAsync()
	return nil
}
func (m *Manager) CreateCollection(req CreateCollectionRequest) (domain.SavedCollectionV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.Mutation, m.fleetVersion); err != nil {
		return domain.SavedCollectionV2{}, err
	}
	if err := m.validTargets(req.MemberIDs); err != nil {
		return domain.SavedCollectionV2{}, err
	}
	c := domain.SavedCollectionV2{SchemaVersion: 2, ID: "collection-" + shortHash(req.IdempotencyKey), Name: req.Name, MemberIDs: unique(req.MemberIDs), Revision: 1}
	m.collections[c.ID] = c
	m.fleetVersion++
	m.persistAsync()
	return c, nil
}

func (m *Manager) PatchCollection(id string, req PatchCollectionRequest) (domain.SavedCollectionV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.collections[id]
	if !ok {
		return c, &Error{"COLLECTION_NOT_FOUND", "Saved collection not found."}
	}
	if err := m.check(req.Mutation, c.Revision); err != nil {
		return c, err
	}
	if req.Name != nil {
		c.Name = strings.TrimSpace(*req.Name)
	}
	if req.MemberIDs != nil {
		if err := m.validTargets(*req.MemberIDs); err != nil {
			return c, err
		}
		c.MemberIDs = unique(*req.MemberIDs)
	}
	if c.Name == "" {
		return c, &Error{"COLLECTION_NAME_REQUIRED", "Saved collection name is required."}
	}
	c.Revision++
	m.collections[id] = c
	m.fleetVersion++
	m.persistAsync()
	return c, nil
}

func (m *Manager) CreateMission(req CreateMissionRequest) (domain.MissionWorkspaceV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req.Mutation, m.fleetVersion); err != nil {
		return domain.MissionWorkspaceV2{}, err
	}
	if active := m.activeCount(); active >= 8 {
		return domain.MissionWorkspaceV2{}, &Error{"MISSION_LIMIT", "Eight active mission workspaces are already open."}
	}
	if err := m.validTargets(req.TargetIDs); err != nil {
		return domain.MissionWorkspaceV2{}, err
	}
	if conflicts := m.conflicts(req.TargetIDs, ""); len(conflicts) > 0 {
		return domain.MissionWorkspaceV2{}, &Error{"MOVEMENT_AUTHORITY_CONFLICT", "Targets already belong to an active mission: " + strings.Join(conflicts, ", ")}
	}
	now := time.Now().UTC()
	targets := unique(req.TargetIDs)
	nameSource := "operator"
	name := strings.TrimSpace(req.Name)
	if name == "" && req.NamingMode == "ai" {
		nameSource = "ai"
		name = "Operation Pending"
	} else if name == "" {
		nameSource = "generated"
		name = m.nextMissionNameLocked()
	}
	mission := domain.MissionWorkspaceV2{SchemaVersion: 2, ID: "mission-" + shortHash(req.IdempotencyKey), Name: m.uniqueMissionNameLocked(name), NameSource: nameSource, Objective: req.Objective, Status: "draft", TargetIDs: targets, TargetSnapshotHash: hashAny(targets), FleetVersion: m.fleetVersion, Version: 1, Geometry: domain.MissionGeometryV2{Revision: 1, IncludedAreas: [][][]float64{}, ExclusionAreas: [][][]float64{}, Waypoints: []domain.GeoPointV2{}, POIs: []domain.MissionPOIV2{}}, Constraints: defaultConstraints(), Formation: "column", Conversation: []domain.MissionChatMessageV2{}, CreatedAt: now, UpdatedAt: now}
	m.missions[mission.ID] = mission
	m.persistAsync()
	return mission, nil
}

// ResetOperations clears transient command authority while preserving persistent
// vessel identities, primary groups, and saved collections.
func (m *Manager) ResetOperations(req Mutation) (domain.FleetSnapshotV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.check(req, m.fleetVersion); err != nil {
		return domain.FleetSnapshotV2{}, err
	}
	m.missions = map[string]domain.MissionWorkspaceV2{}
	m.drafts = map[string]domain.CommandDraftV2{}
	m.plans = map[string]domain.FleetPlanV2{}
	m.leases = map[string]domain.FleetLeaseV2{}
	m.startedPlans = map[string]string{}
	m.programs = map[string]domain.TrajectoryProgramV1{}
	for id, vessel := range m.vessels {
		vessel.Telemetry.MissionID = ""
		vessel.Telemetry.Route = nil
		vessel.Telemetry.Mode = "patrol"
		vessel.Telemetry.SpeedMPS = 0.4
		m.vessels[id] = vessel
	}
	m.fleetVersion++
	m.persistAsync()
	m.clearMissionPersistenceAsync()
	return m.snapshotLocked(), nil
}
func (m *Manager) PatchMission(id string, req PatchMissionRequest) (domain.MissionWorkspaceV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.missions[id]
	if !ok {
		return v, &Error{"MISSION_NOT_FOUND", "Mission not found."}
	}
	if err := m.check(req.Mutation, v.Version); err != nil {
		return v, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len(name) < 2 || len(name) > 64 {
			return v, &Error{"INVALID_MISSION_NAME", "Mission name must be between 2 and 64 characters."}
		}
		v.Name = m.uniqueMissionNameExceptLocked(name, id)
		v.NameSource = "operator"
	}
	if req.Objective != nil {
		v.Objective = *req.Objective
	}
	if req.Status != nil {
		if (*req.Status == "paused" && v.Status != "executing") || (*req.Status == "executing" && v.Status != "paused") {
			return v, &Error{"INVALID_MISSION_TRANSITION", "Only an executing mission may pause, and only a paused mission may resume."}
		}
		if *req.Status != "paused" && *req.Status != "executing" {
			return v, &Error{"INVALID_MISSION_STATUS", "Mission status must be paused or executing."}
		}
		v.Status = *req.Status
	}
	if req.Formation != nil {
		v.Formation = *req.Formation
	}
	if req.Constraints != nil {
		v.Constraints = conservative(*req.Constraints, v.Constraints)
	}
	if req.TargetIDs != nil {
		if err := m.validTargets(*req.TargetIDs); err != nil {
			return v, err
		}
		if c := m.conflicts(*req.TargetIDs, id); len(c) > 0 {
			return v, &Error{"MOVEMENT_AUTHORITY_CONFLICT", "Targets already belong to another active mission."}
		}
		v.TargetIDs = unique(*req.TargetIDs)
		v.TargetSnapshotHash = hashAny(v.TargetIDs)
	}
	v.Version++
	v.UpdatedAt = time.Now().UTC()
	if req.Status == nil {
		v.AuthorizedPlanID = ""
	}
	m.missions[id] = v
	m.persistAsync()
	return v, nil
}

// DeleteMission ends any active movement, releases its vessels, and removes
// transient plans and authority associated with the exact mission version.
func (m *Manager) DeleteMission(id string, req Mutation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.missions[id]
	if !ok {
		return &Error{"MISSION_NOT_FOUND", "Mission not found."}
	}
	if err := m.check(req, v.Version); err != nil {
		return err
	}
	// Invalidate every older asynchronous persistence snapshot before the
	// durable delete. The persistence mutex then guarantees that a snapshot
	// already in flight finishes before the row is removed, while snapshots
	// still waiting observe the newer generation and cannot resurrect it.
	m.persistenceGeneration.Add(1)
	if err := m.deleteMissionPersistence(id); err != nil {
		return &Error{"MISSION_PERSISTENCE_FAILED", "Mission deletion could not be committed. Nothing was deleted."}
	}
	for vesselID, vessel := range m.vessels {
		if vessel.Telemetry.MissionID != id {
			continue
		}
		vessel.Telemetry.MissionID = ""
		vessel.Telemetry.Route = nil
		vessel.Telemetry.Mode = "patrol"
		vessel.Telemetry.SpeedMPS = 0
		m.vessels[vesselID] = vessel
	}
	for key, draft := range m.drafts {
		if draft.MissionID == id {
			delete(m.drafts, key)
		}
	}
	planIDs := map[string]bool{}
	for key, plan := range m.plans {
		if plan.MissionID == id {
			planIDs[key] = true
			delete(m.plans, key)
		}
	}
	for key, lease := range m.leases {
		if lease.MissionID == id {
			delete(m.leases, key)
		}
	}
	for key, planID := range m.startedPlans {
		if planIDs[planID] {
			delete(m.startedPlans, key)
		}
	}
	delete(m.missions, id)
	delete(m.programs, id)
	m.fleetVersion++
	m.persistAsync()
	return nil
}
func (m *Manager) SetGeometry(id string, req GeometryRequest) (domain.MissionWorkspaceV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.missions[id]
	if !ok {
		return v, &Error{"MISSION_NOT_FOUND", "Mission not found."}
	}
	if err := m.check(req.Mutation, v.Version); err != nil {
		return v, err
	}
	if len(req.IncludedAreas) > 0 && len(req.IncludedAreas[0]) < 3 {
		return v, &Error{"INVALID_GEOMETRY", "Included polygon requires at least three vertices."}
	}
	details, err := normalizeWaypointDetails(req.Waypoints, req.WaypointDetails)
	if err != nil {
		return v, err
	}
	v.Geometry = domain.MissionGeometryV2{Revision: v.Geometry.Revision + 1, IncludedAreas: req.IncludedAreas, ExclusionAreas: req.ExclusionAreas, Waypoints: req.Waypoints, WaypointDetails: details, POIs: req.POIs}
	m.reconcileGroupAssemblyWaypointsLocked(id, details)
	v.Version++
	v.UpdatedAt = time.Now().UTC()
	v.AuthorizedPlanID = ""
	m.missions[id] = v
	m.persistAsync()
	return v, nil
}

func (m *Manager) reconcileGroupAssemblyWaypointsLocked(missionID string, waypoints []domain.MissionWaypointV2) {
	byID := make(map[string]domain.MissionWaypointV2, len(waypoints))
	for _, waypoint := range waypoints {
		byID[waypoint.ID] = waypoint
	}
	for id, group := range m.groups {
		changed := false
		if group.AssemblySource == "mission:"+missionID && group.AssemblyWaypointID != "" {
			if waypoint, ok := byID[group.AssemblyWaypointID]; ok {
				point := waypoint.Position
				group.AssemblyPoint = &point
			} else {
				group.AssemblyPoint = nil
				group.AssemblySource = ""
				group.AssemblyWaypointID = ""
			}
			changed = true
		}
		if group.AssemblyPoint == nil {
			wanted := nearestWaypointColor(group.Color)
			for _, waypoint := range waypoints {
				if normalizeWaypointColor(waypoint.Color) != wanted {
					continue
				}
				point := waypoint.Position
				group.AssemblyPoint = &point
				group.AssemblySource = "mission:" + missionID
				group.AssemblyWaypointID = waypoint.ID
				changed = true
				break
			}
		}
		if changed {
			group.Revision++
			m.groups[id] = group
		}
	}
}

func (m *Manager) Compile(id string, req CompileRequest) (domain.CommandDraftV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mission, ok := m.missions[id]
	if !ok {
		return domain.CommandDraftV2{}, &Error{"MISSION_NOT_FOUND", "Mission not found."}
	}
	if err := m.check(req.Mutation, mission.Version); err != nil {
		return domain.CommandDraftV2{}, err
	}
	targets := mission.TargetIDs
	if len(req.TargetIDs) > 0 {
		targets = unique(req.TargetIDs)
	}
	if len(targets) == 0 {
		return domain.CommandDraftV2{}, &Error{"TARGETS_REQUIRED", "Select at least one vessel or group."}
	}
	if err := m.validTargets(targets); err != nil {
		return domain.CommandDraftV2{}, err
	}
	formation := req.Formation
	if formation == "" {
		formation = inferFormation(req.Text, mission.Formation)
	}
	kind := req.GuidanceKind
	if kind == "" {
		kind = inferGuidance(req.Text)
	}
	wps := req.Waypoints
	if len(wps) == 0 {
		wps = mission.Geometry.Waypoints
	}
	geometrySource := ""
	notes := []string{}
	selectedWaypointColor := requestedWaypointColor(req.Text)
	if selectedWaypointColor != "" {
		wps = nil
		for _, waypoint := range mission.Geometry.WaypointDetails {
			if waypoint.Color == selectedWaypointColor {
				wps = append(wps, waypoint.Position)
			}
		}
		if len(wps) > 0 {
			geometrySource = "intent:waypoint-color:" + selectedWaypointColor
			notes = append(notes, fmt.Sprintf("Selected %d %s waypoints in numbered order from the active mission.", len(wps), selectedWaypointColor))
		}
	}
	missionChanged := false
	if len(wps) == 0 && len(mission.Geometry.IncludedAreas) == 0 {
		if polygon, route, source, ok := m.resolveNamedGeometry(req.Text, targets); ok {
			mission.Geometry.Revision++
			mission.Geometry.IncludedAreas = [][][]float64{polygon}
			mission.Geometry.Waypoints = route
			mission.UpdatedAt = time.Now().UTC()
			wps = route
			geometrySource = source
			notes = append(notes, "Operating corridor and patrol route resolved from target positions and the map's local 5 m depth contour; review the exact route before authorization.")
			missionChanged = true
		}
	}
	if len(wps) == 0 && len(mission.Geometry.IncludedAreas) == 0 {
		if polygon, route, source, description, ok := m.resolveRelativeGeometry(req.Text, targets); ok {
			mission.Geometry.Revision++
			mission.Geometry.IncludedAreas = [][][]float64{polygon}
			mission.Geometry.Waypoints = route
			mission.UpdatedAt = time.Now().UTC()
			wps = route
			geometrySource = source
			notes = append(notes, description)
			missionChanged = true
		}
	}
	constraints := mission.Constraints
	if distance, ok := requestedCoastalDistance(req.Text); ok {
		constraints.MaximumShoreDistanceM = distance
		mission.Constraints.MaximumShoreDistanceM = distance
		missionChanged = true
		notes = append(notes, fmt.Sprintf("Coastal offset limited to %.2f nautical miles (%.0f m); shallow or land-intersecting portions are excluded by the depth-aware corridor.", distance/1852, distance))
	}
	if requested, ok := requestedReserve(req.Text); ok {
		if requested > constraints.MinimumReserve {
			constraints.MinimumReserve = requested
			mission.Constraints.MinimumReserve = requested
			missionChanged = true
			notes = append(notes, fmt.Sprintf("Minimum reserve set to %.0f%% from intent.", requested*100))
		} else {
			notes = append(notes, fmt.Sprintf("Requested %.0f%% reserve; standing policy keeps the effective minimum at %.0f%%.", requested*100, constraints.MinimumReserve*100))
		}
	}
	if missionChanged {
		mission.Version++
		mission.AuthorizedPlanID = ""
		m.missions[id] = mission
		m.persistAsync()
	}
	amb := []string{}
	if selectedWaypointColor != "" && len(wps) == 0 {
		amb = append(amb, fmt.Sprintf("No %s waypoints exist in the active mission.", selectedWaypointColor))
	}
	if len(wps) == 0 && len(mission.Geometry.IncludedAreas) == 0 {
		amb = append(amb, "Choose an area or waypoint before route generation.")
	}
	draft := domain.CommandDraftV2{SchemaVersion: 2, ID: "draft-" + shortHash(req.IdempotencyKey), MissionID: id, SourceText: req.Text, Objective: nonempty(req.Text, mission.Objective), TargetIDs: targets, TargetSnapshotHash: hashAny(targets), GeometryRevision: mission.Geometry.Revision, FleetVersion: m.fleetVersion, Constraints: constraints, FormationPreference: formation, GuidanceKind: kind, Waypoints: wps, GeometrySource: geometrySource, ResolutionNotes: notes, Ambiguities: amb}
	draft.ContentHash = hashWithout(draft)
	m.drafts[draft.ID] = draft
	messageID := "message-" + shortHash(req.IdempotencyKey)
	if !hasChatMessage(mission.Conversation, messageID) {
		mission.Conversation = append(mission.Conversation, domain.MissionChatMessageV2{ID: messageID, Role: "operator", Markdown: req.Text, State: "complete", CreatedAt: time.Now().UTC()})
	}
	mission.UpdatedAt = time.Now().UTC()
	m.missions[id] = mission
	m.persistAsync()
	return draft, nil
}

// PlanningContext returns a bounded, read-only projection for the advisory
// model. Hidden authority material, signing keys, leases, and unrelated fleet
// state never cross this boundary.
func (m *Manager) PlanningContext(draftID string) (domain.MissionPlanningContextV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	draft, ok := m.drafts[draftID]
	if !ok {
		return domain.MissionPlanningContextV2{}, &Error{"DRAFT_NOT_FOUND", "Command draft not found."}
	}
	mission := m.missions[draft.MissionID]
	targets := make([]domain.MissionPlanningVesselV2, 0, len(draft.TargetIDs))
	for _, id := range draft.TargetIDs {
		v := m.vessels[id]
		targets = append(targets, domain.MissionPlanningVesselV2{ID: v.ID, Name: v.DisplayName, Class: v.Class.Name, Position: v.Telemetry.Position, Reserve: v.Telemetry.Reserve, MaxSpeedMPS: v.Class.MaxSpeedMPS, PNTIntegrity: v.Telemetry.PNTIntegrity, UncertaintyM: v.Telemetry.UncertaintyM, GroupCode: v.GroupCode, Communications: "authenticated mesh reachable"})
	}
	geometryOptions := []domain.MissionGeometryOptionV2{}
	if advisorGeometryEligible(draft, mission) {
		geometryOptions = m.geometryOptionsLocked(draft.TargetIDs, draft.SourceText)
	}
	conversation := mission.Conversation
	if len(conversation) > 12 {
		conversation = conversation[len(conversation)-12:]
	}
	return domain.MissionPlanningContextV2{SchemaVersion: 2, MissionID: mission.ID, Intent: draft.SourceText, GuidanceKind: draft.GuidanceKind, TargetCount: len(targets), Targets: targets, Constraints: draft.Constraints, Environment: environmentAt(domain.GeoPointV2{-71.34, 41.32}, 0), OperatingAreas: len(mission.Geometry.IncludedAreas), ExclusionAreas: len(mission.Geometry.ExclusionAreas), WaypointCount: len(draft.Waypoints), GeometrySource: draft.GeometrySource, GeometryOptions: geometryOptions, MapBounds: [][]float64{{-71.62, 41.08}, {-71.08, 41.62}}, FormationCurrent: mission.Formation, Conversation: append([]domain.MissionChatMessageV2(nil), conversation...)}, nil
}

// ApplyAdvisor validates and freezes advisory strategies into the immutable
// command draft. Invalid model output is replaced with a deterministic,
// target-aware fallback; it is never allowed to reach route generation.
func (m *Manager) ApplyAdvisor(draftID string, advisor domain.MissionAdvisorV2) (domain.CommandDraftV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	draft, ok := m.drafts[draftID]
	if !ok {
		return domain.CommandDraftV2{}, &Error{"DRAFT_NOT_FOUND", "Command draft not found."}
	}
	if !validAdvisor(advisor, len(draft.TargetIDs)) {
		advisor = deterministicAdvisor(len(draft.TargetIDs), draft.GuidanceKind, "model response failed validation")
	}
	missionChanged := false
	missionForGeometry := m.missions[draft.MissionID]
	if advisorGeometryEligible(draft, missionForGeometry) {
		options := m.geometryOptionsLocked(draft.TargetIDs, draft.SourceText)
		if len(options) > 0 {
			chosen := options[0]
			for _, option := range options {
				if option.ID == advisor.GeometryOptionID {
					chosen = option
					break
				}
			}
			advisor.GeometryOptionID = chosen.ID
			mission := m.missions[draft.MissionID]
			currentID := strings.TrimPrefix(draft.GeometrySource, "intent:map-depth-")
			if currentID != chosen.ID {
				mission.Geometry.Revision++
				mission.Geometry.IncludedAreas = [][][]float64{pointsToCoordinates(chosen.Boundary)}
				mission.Geometry.Waypoints = clonePoints(chosen.Waypoints)
				mission.Geometry.WaypointDetails = nil
				mission.AuthorizedPlanID = ""
				mission.UpdatedAt = time.Now().UTC()
				mission.Version++
				m.missions[mission.ID] = mission
				draft.GeometryRevision = mission.Geometry.Revision
				draft.Waypoints = clonePoints(chosen.Waypoints)
				draft.GeometrySource = "advisor:" + advisor.Provider + ":" + chosen.ID
				draft.ResolutionNotes = append(draft.ResolutionNotes, fmt.Sprintf("%s selected %s from %d depth-validated map sectors; deterministic geometry and policy checks remain authoritative.", strings.ToUpper(advisor.Provider), chosen.Name, len(options)))
				draft.Ambiguities = nil
				missionChanged = true
			}
		}
	}
	draft.Advisor = advisor
	if mission, exists := m.missions[draft.MissionID]; exists {
		markdown := strings.TrimSpace(advisor.Summary)
		if markdown == "" {
			lines := []string{fmt.Sprintf("I prepared %d bounded options for deterministic validation.", len(advisor.Strategies))}
			for _, strategy := range advisor.Strategies {
				lines = append(lines, fmt.Sprintf("- **%s** — %s", strategy.Name, strategy.Description))
			}
			markdown = strings.Join(lines, "\n")
		}
		messageID := "message-advisor-" + shortHash(draft.ID+advisor.Model)
		if !hasChatMessage(mission.Conversation, messageID) {
			mission.Conversation = append(mission.Conversation, domain.MissionChatMessageV2{ID: messageID, Role: "assistant", Markdown: markdown, State: advisor.State, CreatedAt: time.Now().UTC()})
		}
		mission.UpdatedAt = time.Now().UTC()
		m.missions[mission.ID] = mission
		missionChanged = true
	}
	if mission, exists := m.missions[draft.MissionID]; exists && mission.NameSource == "ai" && strings.TrimSpace(advisor.MissionName) != "" {
		mission.Name = m.uniqueMissionNameExceptLocked(strings.TrimSpace(advisor.MissionName), mission.ID)
		mission.NameSource = advisor.Provider
		if !missionChanged {
			mission.Version++
		}
		mission.UpdatedAt = time.Now().UTC()
		m.missions[mission.ID] = mission
		missionChanged = true
	}
	if missionChanged {
		m.persistAsync()
	}
	draft.ContentHash = hashWithout(struct {
		DraftID      string
		MissionID    string
		SourceText   string
		Targets      []string
		Geometry     int64
		FleetVersion int64
		Constraints  domain.ConstraintSetV2
		Strategies   []domain.MissionStrategyV2
	}{draft.ID, draft.MissionID, draft.SourceText, draft.TargetIDs, draft.GeometryRevision, draft.FleetVersion, draft.Constraints, advisor.Strategies})
	m.drafts[draftID] = draft
	return draft, nil
}

func pointsToCoordinates(points []domain.GeoPointV2) [][]float64 {
	coordinates := make([][]float64, len(points))
	for i, point := range points {
		coordinates[i] = []float64{point[0], point[1]}
	}
	return coordinates
}

func advisorGeometryEligible(draft domain.CommandDraftV2, mission domain.MissionWorkspaceV2) bool {
	if requestedWaypointColor(draft.SourceText) != "" {
		return false
	}
	if strings.HasPrefix(draft.GeometrySource, "intent:map-depth-coastal-corridor-") {
		return true
	}
	return (draft.GuidanceKind == "patrol" || draft.GuidanceKind == "search") && len(mission.Geometry.IncludedAreas) == 0 && len(draft.Waypoints) == 0
}

func DeterministicAdvisor(targetCount int, guidance, reason string) domain.MissionAdvisorV2 {
	return deterministicAdvisor(targetCount, guidance, reason)
}

func normalizeWaypointDetails(points []domain.GeoPointV2, details []domain.MissionWaypointV2) ([]domain.MissionWaypointV2, error) {
	if len(details) > 0 && len(details) != len(points) {
		return nil, &Error{"INVALID_GEOMETRY", "Waypoint metadata must match the waypoint list."}
	}
	out := make([]domain.MissionWaypointV2, len(points))
	for i, point := range points {
		waypoint := domain.MissionWaypointV2{ID: "waypoint-" + shortHash(fmt.Sprintf("%.7f:%.7f:%d", point[0], point[1], i+1)), Position: point, Color: "amber", Sequence: i + 1}
		if len(details) > 0 {
			waypoint = details[i]
			waypoint.Position = point
			waypoint.Sequence = i + 1
			if waypoint.ID == "" {
				waypoint.ID = "waypoint-" + shortHash(fmt.Sprintf("%.7f:%.7f:%d", point[0], point[1], i+1))
			}
		}
		waypoint.Color = normalizeWaypointColor(waypoint.Color)
		if waypoint.Color == "" {
			return nil, &Error{"INVALID_GEOMETRY", "Waypoint color must be amber, red, green, cyan, violet, or white."}
		}
		out[i] = waypoint
	}
	return out, nil
}

func requestedWaypointColor(text string) string {
	match := waypointColor.FindStringSubmatch(text)
	if len(match) != 2 {
		return ""
	}
	return normalizeWaypointColor(match[1])
}

func normalizeWaypointColor(color string) string {
	switch strings.ToLower(strings.TrimSpace(color)) {
	case "amber":
		return "amber"
	case "red":
		return "red"
	case "green":
		return "green"
	case "cyan", "blue":
		return "cyan"
	case "violet", "purple":
		return "violet"
	case "white":
		return "white"
	default:
		return ""
	}
}

func (m *Manager) resolveNamedGeometry(text string, targets []string) ([][]float64, []domain.GeoPointV2, string, bool) {
	lower := strings.ToLower(text)
	coastal := []string{"shoreline", "coast", "coastal", "shore patrol", "beach", "beaches", "nearshore", "near shore", "littoral"}
	matched := false
	for _, term := range coastal {
		if strings.Contains(lower, term) {
			matched = true
			break
		}
	}
	if !matched {
		return nil, nil, "", false
	}
	centroid := domain.GeoPointV2{}
	for _, id := range targets {
		position := m.vessels[id].Telemetry.Position
		centroid[0] += position[0]
		centroid[1] += position[1]
	}
	centroid[0] /= float64(len(targets))
	centroid[1] /= float64(len(targets))
	nearest := 0
	minimum := math.Inf(1)
	for i, center := range spawnCenters {
		distance := math.Hypot((centroid[0]-center[0])*.75, centroid[1]-center[1])
		if distance < minimum {
			minimum, nearest = distance, i
		}
	}
	route := clonePoints(coastalDepthRoutes[nearest])
	// Patrol back over the same depth-validated corridor rather than closing a
	// polygon with an unvalidated cross-land leg.
	for i := len(coastalDepthRoutes[nearest]) - 2; i >= 0; i-- {
		route = append(route, coastalDepthRoutes[nearest][i])
	}
	distance, ok := requestedCoastalDistance(text)
	if !ok {
		distance = 1852
	}
	polygon := coastalBounds(route, distance)
	return polygon, route, fmt.Sprintf("intent:map-depth-coastal-corridor-%02d", nearest+1), true
}

func (m *Manager) geometryOptionsLocked(targets []string, text string) []domain.MissionGeometryOptionV2 {
	if len(targets) == 0 {
		return nil
	}
	targetCenter := domain.GeoPointV2{}
	for _, id := range targets {
		position := m.vessels[id].Telemetry.Position
		targetCenter[0] += position[0]
		targetCenter[1] += position[1]
	}
	targetCenter[0] /= float64(len(targets))
	targetCenter[1] /= float64(len(targets))
	distance, ok := requestedCoastalDistance(text)
	if !ok {
		distance = 1852
	}
	options := make([]domain.MissionGeometryOptionV2, 0, len(coastalDepthRoutes))
	seen := map[string]bool{}
	for index, base := range coastalDepthRoutes {
		fingerprint := hashAny(base)
		if seen[fingerprint] {
			continue
		}
		seen[fingerprint] = true
		route := clonePoints(base)
		for i := len(base) - 2; i >= 0; i-- {
			route = append(route, base[i])
		}
		center := domain.GeoPointV2{}
		for _, point := range base {
			center[0] += point[0]
			center[1] += point[1]
		}
		center[0] /= float64(len(base))
		center[1] /= float64(len(base))
		name := fmt.Sprintf("Coastal Sector %02d", index+1)
		if index < len(coastalRouteNames) {
			name = coastalRouteNames[index]
		}
		option := domain.MissionGeometryOptionV2{
			ID:                  fmt.Sprintf("coastal-corridor-%02d", index+1),
			Name:                name,
			Description:         fmt.Sprintf("Depth-aware shoreline patrol corridor with %.2f nm maximum coastal offset.", distance/1852),
			Center:              center,
			Boundary:            coordinatesToPoints(coastalBounds(route, distance)),
			Waypoints:           route,
			DistanceToTargetsKM: routeDistance([]domain.GeoPointV2{targetCenter, center}),
			DepthValidated:      true,
		}
		options = append(options, option)
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].DistanceToTargetsKM == options[j].DistanceToTargetsKM {
			return options[i].ID < options[j].ID
		}
		return options[i].DistanceToTargetsKM < options[j].DistanceToTargetsKM
	})
	return options
}

func coordinatesToPoints(coordinates [][]float64) []domain.GeoPointV2 {
	points := make([]domain.GeoPointV2, 0, len(coordinates))
	for _, coordinate := range coordinates {
		if len(coordinate) >= 2 {
			points = append(points, domain.GeoPointV2{coordinate[0], coordinate[1]})
		}
	}
	return points
}

// resolveRelativeGeometry converts a bounded relative navigation instruction
// into reviewable map geometry. The language model may suggest strategy, but
// deterministic code owns coordinates and preserves the exact requested
// distance for policy evaluation.
func (m *Manager) resolveRelativeGeometry(text string, targets []string) ([][]float64, []domain.GeoPointV2, string, string, bool) {
	match := relativeTravelDistance.FindStringSubmatch(text)
	if len(match) != 3 {
		return nil, nil, "", "", false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil || value <= 0 {
		return nil, nil, "", "", false
	}
	unit := strings.ToLower(match[2])
	distanceM := value * 1852
	unitLabel := "nm"
	switch {
	case strings.HasPrefix(unit, "km"), strings.HasPrefix(unit, "kilometer"):
		distanceM, unitLabel = value*1000, "km"
	case unit == "mi", strings.HasPrefix(unit, "mile"):
		distanceM, unitLabel = value*1609.344, "mi"
	}
	// Keep natural-language resolution bounded. Larger requests require
	// explicit geometry so an accidental transcription cannot create a huge
	// route before the operator sees it.
	if distanceM > 50*1852 {
		return nil, nil, "", "", false
	}

	lower := strings.ToLower(text)
	bearing, direction, ok := requestedBearing(lower)
	if !ok {
		return nil, nil, "", "", false
	}
	centroid := domain.GeoPointV2{}
	for _, id := range targets {
		position := m.vessels[id].Telemetry.Position
		centroid[0] += position[0]
		centroid[1] += position[1]
	}
	centroid[0] /= float64(len(targets))
	centroid[1] /= float64(len(targets))
	outbound := destinationPoint(centroid, bearing, distanceM)
	route := []domain.GeoPointV2{outbound}
	returning := strings.Contains(lower, "return") || strings.Contains(lower, "come back") || strings.Contains(lower, "round trip") || strings.Contains(lower, "round-trip")
	if returning {
		route = append(route, centroid)
	}
	all := append([]domain.GeoPointV2{centroid}, route...)
	polygon := routeBounds(all, 926) // half-nautical-mile review corridor
	source := fmt.Sprintf("intent:relative-%s:%.1f%s", direction, value, unitLabel)
	description := fmt.Sprintf("Resolved a %.1f %s %s leg from the selected asset position%s; exact coordinates and the return leg are shown for review before authorization.", value, unitLabel, direction, map[bool]string{true: " with a return to the starting position", false: ""}[returning])
	return polygon, route, source, description, true
}

func requestedBearing(lower string) (float64, string, bool) {
	if match := explicitHeading.FindStringSubmatch(lower); len(match) == 2 {
		value, err := strconv.ParseFloat(match[1], 64)
		if err == nil && value >= 0 && value <= 360 {
			return math.Mod(value, 360), fmt.Sprintf("heading-%03.0f", math.Mod(value, 360)), true
		}
	}
	// In this packaged Narragansett fixture, open Atlantic water is south of
	// every seeded operating group. This is map knowledge, not model inference.
	for _, phrase := range []string{"out to sea", "out to the sea", "offshore", "open ocean", "open water", "seaward"} {
		if strings.Contains(lower, phrase) {
			return 180, "seaward", true
		}
	}
	cardinals := []struct {
		words   []string
		bearing float64
		name    string
	}{
		{[]string{"north", "northward"}, 0, "north"},
		{[]string{"northeast", "north-east"}, 45, "northeast"},
		{[]string{"east", "eastward"}, 90, "east"},
		{[]string{"southeast", "south-east"}, 135, "southeast"},
		{[]string{"south", "southward"}, 180, "south"},
		{[]string{"southwest", "south-west"}, 225, "southwest"},
		{[]string{"west", "westward"}, 270, "west"},
		{[]string{"northwest", "north-west"}, 315, "northwest"},
	}
	// Check compound directions before their component words.
	order := []int{1, 3, 5, 7, 0, 2, 4, 6}
	for _, index := range order {
		candidate := cardinals[index]
		for _, word := range candidate.words {
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`).MatchString(lower) {
				return candidate.bearing, candidate.name, true
			}
		}
	}
	return 0, "", false
}

func destinationPoint(start domain.GeoPointV2, bearingDeg, distanceM float64) domain.GeoPointV2 {
	radians := bearingDeg * math.Pi / 180
	lat := start[1] + math.Cos(radians)*distanceM/111_000
	lonScale := 111_000 * math.Cos(start[1]*math.Pi/180)
	return domain.GeoPointV2{start[0] + math.Sin(radians)*distanceM/lonScale, lat}
}

func routeBounds(route []domain.GeoPointV2, paddingM float64) [][]float64 {
	minX, maxX, minY, maxY := route[0][0], route[0][0], route[0][1], route[0][1]
	for _, point := range route[1:] {
		minX, maxX = math.Min(minX, point[0]), math.Max(maxX, point[0])
		minY, maxY = math.Min(minY, point[1]), math.Max(maxY, point[1])
	}
	latPad := paddingM / 111_000
	lonPad := paddingM / (111_000 * math.Cos(((minY+maxY)/2)*math.Pi/180))
	return [][]float64{{minX - lonPad, minY - latPad}, {maxX + lonPad, minY - latPad}, {maxX + lonPad, maxY + latPad}, {minX - lonPad, maxY + latPad}, {minX - lonPad, minY - latPad}}
}

func requestedCoastalDistance(text string) (float64, bool) {
	match := coastalDistance.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil || value <= 0 || value > 10 {
		return 0, false
	}
	return value * 1852, true
}

func coastalBounds(route []domain.GeoPointV2, distanceM float64) [][]float64 {
	minX, maxX, minY, maxY := route[0][0], route[0][0], route[0][1], route[0][1]
	for _, point := range route[1:] {
		minX, maxX = math.Min(minX, point[0]), math.Max(maxX, point[0])
		minY, maxY = math.Min(minY, point[1]), math.Max(maxY, point[1])
	}
	latPad := distanceM / 111_000
	lonPad := distanceM / (111_000 * math.Cos(((minY+maxY)/2)*math.Pi/180))
	return [][]float64{{minX - lonPad, minY - latPad}, {maxX + lonPad, minY - latPad}, {maxX + lonPad, maxY + latPad}, {minX - lonPad, maxY + latPad}, {minX - lonPad, minY - latPad}}
}

func requestedReserve(text string) (float64, bool) {
	for _, pattern := range []*regexp.Regexp{reserveAfterLabel, reserveBeforeLabel} {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 2 {
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err == nil && value >= 0 && value <= 100 {
			return float64(value) / 100, true
		}
	}
	return 0, false
}
func (m *Manager) GeneratePlans(id string, req PlansRequest) ([]domain.FleetPlanV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mission, ok := m.missions[id]
	if !ok {
		return nil, &Error{"MISSION_NOT_FOUND", "Mission not found."}
	}
	if err := m.check(req.Mutation, mission.Version); err != nil {
		return nil, err
	}
	draft, ok := m.drafts[req.DraftID]
	if !ok || draft.MissionID != id {
		return nil, &Error{"DRAFT_NOT_FOUND", "Command draft not found."}
	}
	if len(draft.Ambiguities) > 0 {
		return nil, &Error{"COMMAND_AMBIGUOUS", strings.Join(draft.Ambiguities, " ")}
	}
	strategies := draft.Advisor.Strategies
	if len(strategies) == 0 {
		strategies = deterministicAdvisor(len(draft.TargetIDs), draft.GuidanceKind, "advisor not available").Strategies
	}
	out := make([]domain.FleetPlanV2, 0, len(strategies))
	for i, strategy := range strategies {
		p := m.makePlan(mission, draft, strategy, i)
		m.plans[p.ID] = p
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return planScore(out[i]) > planScore(out[j]) })
	recommended := false
	for i := range out {
		out[i].Recommended = !recommended && out[i].PolicyStatus != "prohibited"
		if out[i].Recommended {
			recommended = true
		}
		p := out[i]
		m.plans[p.ID] = p
	}
	mission.PlanIDs = nil
	for _, p := range out {
		mission.PlanIDs = append(mission.PlanIDs, p.ID)
	}
	if _, executing := m.programs[id]; !executing {
		mission.Status = "planned"
	}
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	m.missions[id] = mission
	return out, nil
}
func (m *Manager) Preview(mid, pid string, req PlanActionRequest) (domain.FleetPreviewV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return domain.FleetPreviewV2{}, &Error{"MUTATION_METADATA_REQUIRED", "request_id and idempotency_key are required."}
	}
	p, err := m.plan(mid, pid)
	if err != nil {
		return domain.FleetPreviewV2{}, err
	}
	mission := m.missions[mid]
	if req.ExpectedVersion != mission.Version {
		return domain.FleetPreviewV2{}, stale(mission.Version)
	}
	routes := map[string][]domain.GeoPointV2{}
	duration := 0
	for _, a := range p.Assignments {
		routes[a.VesselID] = a.Route
		d := int(a.DistanceKM * 1000 / a.SpeedMPS)
		if d > duration {
			duration = d
		}
	}
	return domain.FleetPreviewV2{PlanID: pid, PlanHash: p.ContentHash, DurationSeconds: duration, Routes: routes, NothingSent: true}, nil
}
func (m *Manager) Authorize(mid, pid string, req PlanActionRequest) (domain.FleetLeaseV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, err := m.plan(mid, pid)
	if err != nil {
		return domain.FleetLeaseV2{}, err
	}
	mission := m.missions[mid]
	leaseID := "lease-" + shortHash(req.IdempotencyKey)
	if existing, ok := m.leases[leaseID]; ok {
		if existing.MissionID == mid && existing.PlanID == pid && existing.PlanHash == req.PlanHash && existing.OperatorID == req.OperatorID {
			return existing, nil
		}
		return domain.FleetLeaseV2{}, &Error{"IDEMPOTENCY_CONFLICT", "Authorization key was already used for a different request."}
	}
	if err := m.check(req.Mutation, mission.Version); err != nil {
		return domain.FleetLeaseV2{}, err
	}
	if p.SourceMissionVersion+1 != mission.Version {
		return domain.FleetLeaseV2{}, &Error{"STALE_PLAN", "Mission targets, geometry, formation, or constraints changed after this plan was generated."}
	}
	if req.PlanHash != p.ContentHash {
		return domain.FleetLeaseV2{}, &Error{"PLAN_HASH_MISMATCH", "Exact plan hash does not match."}
	}
	if p.PolicyStatus != "approval_required" {
		return domain.FleetLeaseV2{}, &Error{"POLICY_REJECTED", "Plan does not satisfy policy."}
	}
	if req.OperatorID != "demo-operator" {
		return domain.FleetLeaseV2{}, &Error{"OPERATOR_APPROVAL_REQUIRED", "demo-operator approval is required."}
	}
	now := time.Now().UTC()
	lease := domain.FleetLeaseV2{ID: leaseID, MissionID: mid, PlanID: pid, PlanHash: p.ContentHash, OperatorID: req.OperatorID, TargetIDs: mission.TargetIDs, IssuedAt: now, ExpiresAt: now.Add(90 * time.Second)}
	lease.Signature = m.sign(lease)
	m.leases[lease.ID] = lease
	mission.AuthorizedPlanID = pid
	if _, executing := m.programs[mid]; !executing {
		mission.Status = "authorized"
	}
	mission.Version++
	mission.UpdatedAt = now
	m.missions[mid] = mission
	return lease, nil
}
func (m *Manager) Start(mid, pid string, req PlanActionRequest) (domain.MissionWorkspaceV2, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, err := m.plan(mid, pid)
	if err != nil {
		return domain.MissionWorkspaceV2{}, err
	}
	mission := m.missions[mid]
	if old, ok := m.startedPlans[req.IdempotencyKey]; ok {
		if old == pid {
			return mission, nil
		}
		return mission, &Error{"IDEMPOTENCY_CONFLICT", "Idempotency key already used for another plan."}
	}
	if err := m.check(req.Mutation, mission.Version); err != nil {
		return mission, err
	}
	lease, ok := m.leases[req.LeaseID]
	if !ok || lease.MissionID != mid || lease.PlanID != pid {
		return mission, &Error{"LEASE_REQUIRED", "Matching movement lease required."}
	}
	if time.Now().UTC().After(lease.ExpiresAt) {
		return mission, &Error{"LEASE_EXPIRED", "Movement lease expired."}
	}
	if req.PlanHash != p.ContentHash || lease.PlanHash != p.ContentHash {
		return mission, &Error{"PLAN_HASH_MISMATCH", "Plan changed after approval."}
	}
	m.startedPlans[req.IdempotencyKey] = pid
	program, revising := m.programs[mid]
	revisionNumber, createdTick, activationTick := 1, int64(0), int64(0)
	if revising {
		revisionNumber = program.ActiveRevision + 1
		if program.PendingRevision >= revisionNumber {
			revisionNumber = program.PendingRevision + 1
		}
		currentTick := program.MissionTickMS / 1000
		createdTick = currentTick
		activationTick = ((currentTick + 29) / 10) * 10
	}
	revision := trajectory.BuildRevision(mission, p, lease, revisionNumber, createdTick, activationTick, m.secret)
	if !trajectory.ValidateRevision(revision, m.secret) {
		return mission, &Error{"TRAJECTORY_SIGNATURE_INVALID", "Generated trajectory revision failed signature validation."}
	}
	if revising {
		trajectory.AddPending(&program, revision)
	} else {
		program = trajectory.NewProgram(mid, revision, 60)
	}
	m.programs[mid] = program
	mission.Status = "executing"
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	m.missions[mid] = mission
	for _, a := range p.Assignments {
		v := m.vessels[a.VesselID]
		if !revising {
			v.Telemetry.Route = clonePoints(a.Route)
		}
		v.Telemetry.MissionID = mid
		v.Telemetry.Mode = "mission"
		v.Telemetry.SpeedMPS = a.SpeedMPS
		v.Telemetry.ProjectedReserve = p.MinimumReserve
		m.vessels[v.ID] = v
	}
	m.fleetVersion++
	m.persistAsync()
	m.persistProgramAsync(mid, program)
	return mission, nil
}

func (m *Manager) makePlan(mission domain.MissionWorkspaceV2, draft domain.CommandDraftV2, strategy domain.MissionStrategyV2, index int) domain.FleetPlanV2 {
	targets := cloneStrings(draft.TargetIDs)
	sort.Strings(targets)
	formation := strategy.Formation
	speed := math.Max(.35, math.Min(draft.Constraints.MaximumSpeedMPS, draft.Constraints.MaximumSpeedMPS*strategy.SpeedFactor))
	assignments := make([]domain.FleetAssignmentV2, 0, len(targets))
	total := 0.
	maxDistance := 0.
	minReserve := 1.
	for i, id := range targets {
		v := m.vessels[id]
		route := []domain.GeoPointV2{v.Telemetry.Position}
		reference := draft.Waypoints
		if len(reference) == 0 && len(mission.Geometry.IncludedAreas) > 0 {
			reference = sweepLane(mission.Geometry.IncludedAreas[0], i, len(targets))
		}
		for _, p := range reference {
			off := formationOffset(formation, i, len(targets), draft.Constraints.FormationSpacingM, p[1])
			route = append(route, domain.GeoPointV2{p[0] + off[0], p[1] + off[1]})
		}
		dist := routeDistance(route)
		reserve := v.Telemetry.Reserve - dist*(.018+speed*.004)
		if reserve < minReserve {
			minReserve = reserve
		}
		total += dist
		maxDistance = math.Max(maxDistance, dist)
		assignments = append(assignments, domain.FleetAssignmentV2{VesselID: id, Route: route, SpeedMPS: math.Min(speed, v.Class.MaxSpeedMPS), DistanceKM: dist})
	}
	// The model chooses a planning posture, not final kinematics. For balanced
	// or coverage-oriented strategies, deterministically raise speed only as
	// much as needed to satisfy the existing duration limit. Conservative
	// strategies remain visible even when policy rejects their longer runtime.
	if strategy.ReserveBias <= .65 && draft.Constraints.MaximumDurationMinutes > 0 {
		required := maxDistance * 1000 / (draft.Constraints.MaximumDurationMinutes * 60) * 1.01
		if required > speed {
			speed = math.Min(draft.Constraints.MaximumSpeedMPS, required)
			minReserve = 1
			for i := range assignments {
				v := m.vessels[assignments[i].VesselID]
				assignments[i].SpeedMPS = math.Min(speed, v.Class.MaxSpeedMPS)
				reserve := v.Telemetry.Reserve - assignments[i].DistanceKM*(.018+assignments[i].SpeedMPS*.004)
				minReserve = math.Min(minReserve, reserve)
			}
		}
	}
	duration := maxDistance * 1000 / speed / 60
	coverage := math.Min(99, 72+float64(len(targets))*2.4-float64(index)*1.6)
	minSep := draft.Constraints.FormationSpacingM
	if len(targets) == 1 {
		minSep = 0
	}
	status := "approval_required"
	reasons := []string{"ROUTES_WITHIN_FIXTURE_OPERATING_AREA", "EXACT_HASH_APPROVAL_REQUIRED"}
	if minReserve < draft.Constraints.MinimumReserve {
		status = "prohibited"
		reasons = append(reasons, "MINIMUM_RESERVE_VIOLATION")
	}
	if maxDistance > draft.Constraints.MaximumRouteDistanceKM {
		status = "prohibited"
		reasons = append(reasons, "MAXIMUM_ROUTE_DISTANCE_VIOLATION")
	}
	if duration > draft.Constraints.MaximumDurationMinutes {
		status = "prohibited"
		reasons = append(reasons, "MAXIMUM_DURATION_VIOLATION")
	}
	p := domain.FleetPlanV2{SchemaVersion: 2, ID: fmt.Sprintf("plan-%s-%d", shortHash(draft.ContentHash), index+1), MissionID: mission.ID, DraftID: draft.ID, Name: strategy.Name, Description: strategy.Description, Formation: formation, AdvisorSource: draft.Advisor.Provider, AdvisorModel: draft.Advisor.Model, Maneuvers: cloneStrings(strategy.Maneuvers), Assignments: assignments, CoveragePercent: coverage, MinimumReserve: minReserve, DurationMinutes: duration, EnergyKWH: total * (1.8 + speed*.4), LinkExposureSeconds: duration * 60 * .08 * float64(index+1), MinimumSeparationM: minSep, PolicyStatus: status, ReasonCodes: reasons, SourceMissionVersion: mission.Version}
	p.ContentHash = hashWithout(p)
	return p
}

func validAdvisor(advisor domain.MissionAdvisorV2, targetCount int) bool {
	if len(advisor.Strategies) < 2 || len(advisor.Strategies) > 4 || advisor.Provider == "" || advisor.State == "" {
		return false
	}
	allowed := map[string]bool{"independent": true, "column": true, "line_abreast": true, "wedge": true, "echelon_left": true, "echelon_right": true, "parallel_columns": true, "dispersed_screen": true, "ring": true, "search_grid": true}
	seen := map[string]bool{}
	for _, strategy := range advisor.Strategies {
		if strategy.ID == "" || strategy.Name == "" || strategy.Description == "" || seen[strategy.ID] || !allowed[strategy.Formation] || strategy.SpeedFactor < .25 || strategy.SpeedFactor > 1 || strategy.ReserveBias < 0 || strategy.ReserveBias > 1 || len(strategy.Maneuvers) < 2 || len(strategy.Maneuvers) > 6 {
			return false
		}
		if targetCount == 1 && strategy.Formation != "independent" {
			return false
		}
		if targetCount == 1 {
			semantic := strings.ToLower(strategy.Name + " " + strategy.Description + " " + strings.Join(strategy.Maneuvers, " "))
			for _, fleetOnly := range []string{"formation", "regroup", "fleet", "other vessel", "separation exceeds"} {
				if strings.Contains(semantic, fleetOnly) {
					return false
				}
			}
		}
		if targetCount > 1 && strategy.Formation == "independent" {
			return false
		}
		seen[strategy.ID] = true
	}
	return true
}

func deterministicAdvisor(targetCount int, guidance, reason string) domain.MissionAdvisorV2 {
	if guidance == "" {
		guidance = "patrol"
	}
	if targetCount == 1 {
		return domain.MissionAdvisorV2{State: "fallback", Provider: "deterministic", Model: "keelmesh-target-aware-v2", Summary: "Target-aware fallback used: " + reason, MissionName: "Operation Coastal Watch", Strategies: []domain.MissionStrategyV2{
			{ID: "close-track", Name: "Close Shoreline Patrol", Description: "Stay close to the validated shoreline corridor while preserving the configured depth and object margins.", Formation: "independent", GuidanceKind: guidance, SpeedFactor: .92, ReserveBias: .25, Maneuvers: []string{"join validated coastal corridor", "track shoreline at bounded offset", "safe hold on completion"}},
			{ID: "reserve-first", Name: "Reserve-Conserving Patrol", Description: "Reduce propulsion demand and prioritize projected battery reserve over completion time.", Formation: "independent", GuidanceKind: guidance, SpeedFactor: .52, ReserveBias: .75, Maneuvers: []string{"enter corridor at economy speed", "patrol with reserve checks", "return to safe hold before reserve floor"}},
			{ID: "current-aware", Name: "Current-Assisted Patrol", Description: "Use the simulated current direction to reduce station-keeping and propulsion demand where policy permits.", Formation: "independent", GuidanceKind: guidance, SpeedFactor: .63, ReserveBias: .5, Maneuvers: []string{"intercept favorable current leg", "patrol depth-safe corridor", "counter-drift and safe hold"}},
		}}
	}
	return domain.MissionAdvisorV2{State: "fallback", Provider: "deterministic", Model: "keelmesh-target-aware-v2", Summary: "Target-aware fallback used: " + reason, MissionName: "Operation Fleet Watch", Strategies: []domain.MissionStrategyV2{
		{ID: "parallel-screen", Name: "Parallel Shoreline Screen", Description: "Distribute the selected vessels across parallel coastal lanes for fast coverage.", Formation: "line_abreast", GuidanceKind: guidance, SpeedFactor: .92, ReserveBias: .25, Maneuvers: []string{"rendezvous at corridor entry", "establish parallel screen", "patrol assigned lanes", "regroup at safe hold"}},
		{ID: "staggered-sweep", Name: "Staggered Coastal Sweep", Description: "Use an echelon to preserve sensor overlap while reducing simultaneous turns near the shoreline.", Formation: "echelon_right", GuidanceKind: guidance, SpeedFactor: .67, ReserveBias: .45, Maneuvers: []string{"form echelon at safe separation", "sweep coastal corridor", "rotate lead on reserve threshold", "regroup on completion"}},
		{ID: "reserve-trail", Name: "Reserve-First Trail", Description: "Follow one depth-validated reference path with conservative speed and spacing.", Formation: "column", GuidanceKind: guidance, SpeedFactor: .5, ReserveBias: .8, Maneuvers: []string{"form trail at validated entry", "patrol at economy speed", "maintain communications spacing", "safe hold on completion"}},
	}}
}

// sweepLane converts an area polygon into one deterministic lane per vessel.
// Alternate lane direction to avoid a shared perimeter traversal and large
// synchronized turns at the edge of the operating area.
func sweepLane(area [][]float64, index, count int) []domain.GeoPointV2 {
	if len(area) == 0 {
		return nil
	}
	minX, maxX, minY, maxY := area[0][0], area[0][0], area[0][1], area[0][1]
	for _, point := range area[1:] {
		if len(point) < 2 {
			continue
		}
		minX, maxX = math.Min(minX, point[0]), math.Max(maxX, point[0])
		minY, maxY = math.Min(minY, point[1]), math.Max(maxY, point[1])
	}
	fraction := float64(index+1) / float64(count+1)
	y := minY + (maxY-minY)*fraction
	left, right := domain.GeoPointV2{minX + (maxX-minX)*.06, y}, domain.GeoPointV2{maxX - (maxX-minX)*.06, y}
	if index%2 == 1 {
		return []domain.GeoPointV2{right, left}
	}
	return []domain.GeoPointV2{left, right}
}

func (m *Manager) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simTickMS += 200
	for mid, program := range m.programs {
		mission, exists := m.missions[mid]
		if !exists || mission.Status == "paused" || mission.Status == "ended" || mission.Status == "completed" {
			continue
		}
		activated := trajectory.Advance(&program, 200)
		m.programs[mid] = program
		active := false
		for vesselID, cursor := range program.Cursors {
			v, vesselExists := m.vessels[vesselID]
			if !vesselExists {
				continue
			}
			segment, segmentActive := trajectory.CurrentSegment(program, vesselID)
			if !segmentActive {
				if cursor.Lifecycle == "completed" {
					v.Telemetry.Route = nil
					v.Telemetry.MissionID = ""
					v.Telemetry.Mode = "station_keep"
					v.Telemetry.SpeedMPS = 0
					v.Telemetry.TapeDepthSeconds = 0
					m.vessels[vesselID] = v
				}
				continue
			}
			active = true
			target := segment.End
			adjustment := m.localAdjustmentLocked(v, segment)
			if !adjustment.InsideEnvelope {
				v.Telemetry.SpeedMPS = 0
				v.Telemetry.TapeDepthSeconds = cursor.HotTapeDepthS
				v.Telemetry.Mode = "safe_hold · instruction requested"
				v.Telemetry.Environment = environmentAt(v.Telemetry.Position, float64(program.MissionTickMS/1000))
				v.Telemetry.Route = nil
				program.LastAdjustments[vesselID] = adjustment
				m.vessels[vesselID] = v
				continue
			}
			if adjustment.InsideEnvelope && adjustment.LateralOffsetM != 0 {
				target = lateralPoint(segment.Start, segment.End, adjustment.LateralOffsetM)
			}
			speed := math.Min(segment.MaximumSpeedMPS, segment.TargetSpeedMPS*adjustment.SpeedFactor)
			step := speed * .2 / 111_000
			dx, dy := target[0]-v.Telemetry.Position[0], target[1]-v.Telemetry.Position[1]
			d := math.Hypot(dx, dy)
			if d <= step {
				v.Telemetry.Position = target
			} else {
				v.Telemetry.Position[0] += dx / d * step
				v.Telemetry.Position[1] += dy / d * step
			}
			v.Telemetry.HeadingDeg = math.Mod(math.Atan2(dx, dy)*180/math.Pi+360+adjustment.HeadingDelta, 360)
			v.Telemetry.SpeedMPS = speed
			v.Telemetry.TapeDepthSeconds = cursor.HotTapeDepthS
			v.Telemetry.Mode = "mission"
			if adjustment.Kind != "nominal" {
				v.Telemetry.Mode = "mission · adaptive"
				program.LastAdjustments[vesselID] = adjustment
			}
			v.Telemetry.Reserve = m.advanceEnergy(v, speed, program.MissionTickMS/1000, .2)
			v.Telemetry.ProjectedReserve = math.Max(segment.MinimumReserve, v.Telemetry.Reserve-float64(cursor.ProgramRemainingS)*.000006)
			v.Telemetry.Environment = environmentAt(v.Telemetry.Position, float64(program.MissionTickMS/1000))
			v.Telemetry.Route = []domain.GeoPointV2{v.Telemetry.Position, segment.End}
			m.vessels[vesselID] = v
		}
		m.programs[mid] = program
		if program.MissionTickMS%10_000 == 0 {
			m.persistProgramAsync(mid, program)
		}
		if !active {
			mission.Status = "completed"
			mission.Version++
			mission.UpdatedAt = time.Now().UTC()
			m.missions[mid] = mission
			m.persistProgramAsync(mid, program)
		} else if activated {
			mission.UpdatedAt = time.Now().UTC()
			m.missions[mid] = mission
			m.persistProgramAsync(mid, program)
		}
	}
	// Telemetry, battery, and simulation-clock motion do not mutate fleet
	// configuration. Keeping their 5 Hz cadence out of fleetVersion prevents
	// ordinary group/mission writes from racing continuous station keeping.
	m.tickIdleGroupsLocked()
}

func (m *Manager) localAdjustmentLocked(vessel domain.VesselProfileV2, segment domain.TrajectorySegmentV2) domain.LocalAdjustmentV1 {
	decisionNode, scope := vessel.ID, "local"
	if group, ok := m.groups[vessel.GroupID]; ok {
		group = m.groupDecisionSnapshotLocked(group)
		if group.DecisionNodeID != "" {
			decisionNode, scope = group.DecisionNodeID, "group"
		}
	}
	adjustment := domain.LocalAdjustmentV1{VesselID: vessel.ID, Tick: segment.ActivationTick, Kind: "nominal", Reason: "inside signed trajectory envelope", SpeedFactor: 1, InsideEnvelope: true, DecisionNodeID: decisionNode, DecisionScope: scope, Escalation: "none"}
	if vessel.Telemetry.UncertaintyM > segment.MaximumUncertaintyM || vessel.Telemetry.PNTIntegrity == "unsafe" {
		adjustment.Kind = "guardrail_stop"
		adjustment.Reason = "PNT evidence is outside the signed trajectory envelope"
		adjustment.SpeedFactor = 0
		adjustment.InsideEnvelope = false
		adjustment.Escalation = "instruction_requested"
		adjustment.Contingency = "safe_hold"
		return adjustment
	}
	if vessel.Telemetry.Reserve <= segment.MinimumReserve {
		adjustment.Kind = "guardrail_stop"
		adjustment.Reason = "reserve is at or below the signed mission floor"
		adjustment.SpeedFactor = 0
		adjustment.InsideEnvelope = false
		adjustment.Escalation = "instruction_requested"
		adjustment.Contingency = "safe_hold"
		return adjustment
	}
	if vessel.Telemetry.Reserve-segment.MinimumReserve < .08 {
		adjustment.Kind = "energy_compensation"
		adjustment.Reason = "projected reserve is near the signed floor"
		adjustment.SpeedFactor = .72
	}
	closestID, closestM := "", math.MaxFloat64
	for otherID, other := range m.vessels {
		if otherID == vessel.ID {
			continue
		}
		distance := geoDistanceM(vessel.Telemetry.Position, other.Telemetry.Position)
		if distance < closestM {
			closestID, closestM = otherID, distance
		}
	}
	if closestM < segment.MinimumSeparationM*1.5 {
		adjustment.Kind = "collision_avoidance"
		adjustment.Reason = "bounded closest-approach correction around " + m.vessels[closestID].Callsign
		adjustment.LateralOffsetM = math.Min(segment.MaxLateralAdjustM, segment.MinimumSeparationM)
		adjustment.SpeedFactor = math.Min(adjustment.SpeedFactor, .65)
		adjustment.HeadingDelta = 8
	}
	return adjustment
}

func lateralPoint(start, end domain.GeoPointV2, offsetM float64) domain.GeoPointV2 {
	dx, dy := end[0]-start[0], end[1]-start[1]
	length := math.Hypot(dx, dy)
	if length == 0 {
		return end
	}
	offset := offsetM / 111_000
	return domain.GeoPointV2{end[0] - dy/length*offset, end[1] + dx/length*offset}
}

func geoDistanceM(a, b domain.GeoPointV2) float64 {
	latitude := (a[1] + b[1]) * math.Pi / 360
	return math.Hypot((b[0]-a[0])*111_000*math.Cos(latitude), (b[1]-a[1])*111_000)
}

func (m *Manager) advanceEnergy(vessel domain.VesselProfileV2, speed float64, missionTick int64, seconds float64) float64 {
	base, propulsion, batteryKWH, solarKW := 250., 160., 40., 2.
	switch vessel.Class.ID {
	case "kestrel":
		base, propulsion, batteryKWH, solarKW = 150, 85, 18, 1.2
	case "atlas":
		base, propulsion, batteryKWH, solarKW = 450, 320, 90, 4
	}
	dayPhase := 2 * math.Pi * float64(missionTick%86400) / 86400
	solar := solarKW * 1000 * math.Max(0, math.Sin(dayPhase-math.Pi/2))
	load := base + propulsion*math.Pow(speed, 3)
	delta := (solar - load) * seconds / 3_600_000 / batteryKWH
	return math.Max(0, math.Min(1, vessel.Telemetry.Reserve+delta))
}

func (m *Manager) tickIdleGroupsLocked() bool {
	changed := false
	for _, group := range m.groups {
		if group.AssemblyPoint == nil || len(group.MemberIDs) == 0 {
			continue
		}
		for index, vesselID := range group.MemberIDs {
			vessel := m.vessels[vesselID]
			if vessel.Telemetry.MissionID != "" {
				continue
			}
			offset := formationOffset(group.Formation, index, len(group.MemberIDs), group.FormationSpacingM, (*group.AssemblyPoint)[1])
			target := domain.GeoPointV2{(*group.AssemblyPoint)[0] + offset[0], (*group.AssemblyPoint)[1] + offset[1]}
			distance := geoDistanceM(vessel.Telemetry.Position, target)
			if distance > 2 {
				dx, dy := target[0]-vessel.Telemetry.Position[0], target[1]-vessel.Telemetry.Position[1]
				length := math.Hypot(dx, dy)
				speed := math.Min(.8, distance/20)
				step := speed * .2 / 111_000
				vessel.Telemetry.Position[0] += dx / length * math.Min(step, length)
				vessel.Telemetry.Position[1] += dy / length * math.Min(step, length)
				vessel.Telemetry.HeadingDeg = math.Mod(math.Atan2(dx, dy)*180/math.Pi+360, 360)
				vessel.Telemetry.SpeedMPS = speed
				vessel.Telemetry.Mode = "forming · " + strings.ReplaceAll(group.Formation, "_", " ")
			} else {
				vessel.Telemetry.SpeedMPS = 0
				vessel.Telemetry.Mode = "station_keep · " + strings.ReplaceAll(group.Formation, "_", " ")
			}
			vessel.Telemetry.Reserve = m.advanceEnergy(vessel, vessel.Telemetry.SpeedMPS, m.simTickMS/1000, .2)
			vessel.Telemetry.Environment = environmentAt(vessel.Telemetry.Position, float64(m.simTickMS/1000))
			m.vessels[vesselID] = vessel
			changed = true
		}
	}
	return changed
}

func (m *Manager) check(req Mutation, current int64) error {
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return &Error{"MUTATION_METADATA_REQUIRED", "request_id and idempotency_key are required."}
	}
	if req.ExpectedVersion != current {
		return stale(current)
	}
	if prior, ok := m.idempotency[req.IdempotencyKey]; ok && prior != req.RequestID {
		return &Error{"IDEMPOTENCY_CONFLICT", "Idempotency key was used by another request."}
	}
	m.idempotency[req.IdempotencyKey] = req.RequestID
	return nil
}
func stale(current int64) error {
	return &Error{"STALE_STATE", fmt.Sprintf("Expected state version does not match current version %d.", current)}
}
func (m *Manager) validTargets(ids []string) error {
	for _, id := range unique(ids) {
		if _, ok := m.vessels[id]; !ok {
			return &Error{"VESSEL_NOT_FOUND", "Target vessel not found: " + id}
		}
	}
	return nil
}
func (m *Manager) activeCount() int {
	n := 0
	for _, v := range m.missions {
		if v.Status != "completed" && v.Status != "ended" {
			n++
		}
	}
	return n
}
func (m *Manager) conflicts(ids []string, except string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	out := []string{}
	for _, x := range m.missions {
		if x.ID == except || x.Status == "completed" || x.Status == "ended" {
			continue
		}
		for _, id := range x.TargetIDs {
			if set[id] {
				out = append(out, m.vessels[id].DisplayName)
			}
		}
	}
	return uniqueStrings(out)
}
func (m *Manager) plan(mid, pid string) (domain.FleetPlanV2, error) {
	p, ok := m.plans[pid]
	if !ok || p.MissionID != mid {
		return p, &Error{"PLAN_NOT_FOUND", "Fleet plan not found."}
	}
	return p, nil
}
func (m *Manager) sign(v any) string {
	mac := hmac.New(sha256.New, m.secret)
	b, _ := json.Marshal(v)
	mac.Write(b)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func defaultConstraints() domain.ConstraintSetV2 {
	return domain.ConstraintSetV2{MinimumReserve: .30, MaximumSpeedMPS: 1.8, MinimumVesselSeparationM: 35, MinimumObjectSeparationM: 50, MaximumWaveHeightM: 2.2, MaximumWindMPS: 16, MaximumPNTUncertaintyM: 25, MaximumDurationMinutes: 240, MaximumRouteDistanceKM: 25, MinimumTapeWatermarkSeconds: 20, Formation: "column", FormationSpacingM: 60, LeaderPolicy: "best_link_and_reserve", RegroupThresholdM: 120}
}
func conservative(a, b domain.ConstraintSetV2) domain.ConstraintSetV2 {
	if a.MinimumReserve < b.MinimumReserve {
		a.MinimumReserve = b.MinimumReserve
	}
	if a.MinimumVesselSeparationM < b.MinimumVesselSeparationM {
		a.MinimumVesselSeparationM = b.MinimumVesselSeparationM
	}
	if a.MinimumObjectSeparationM < b.MinimumObjectSeparationM {
		a.MinimumObjectSeparationM = b.MinimumObjectSeparationM
	}
	if a.MinimumTapeWatermarkSeconds < b.MinimumTapeWatermarkSeconds {
		a.MinimumTapeWatermarkSeconds = b.MinimumTapeWatermarkSeconds
	}
	if a.MaximumSpeedMPS == 0 || a.MaximumSpeedMPS > b.MaximumSpeedMPS {
		a.MaximumSpeedMPS = b.MaximumSpeedMPS
	}
	if a.MaximumWaveHeightM == 0 || a.MaximumWaveHeightM > b.MaximumWaveHeightM {
		a.MaximumWaveHeightM = b.MaximumWaveHeightM
	}
	if a.MaximumWindMPS == 0 || a.MaximumWindMPS > b.MaximumWindMPS {
		a.MaximumWindMPS = b.MaximumWindMPS
	}
	if a.MaximumPNTUncertaintyM == 0 || a.MaximumPNTUncertaintyM > b.MaximumPNTUncertaintyM {
		a.MaximumPNTUncertaintyM = b.MaximumPNTUncertaintyM
	}
	if b.MaximumShoreDistanceM > 0 && (a.MaximumShoreDistanceM == 0 || a.MaximumShoreDistanceM > b.MaximumShoreDistanceM) {
		a.MaximumShoreDistanceM = b.MaximumShoreDistanceM
	}
	return a
}
func formationOffset(f string, i, n int, spacing, latitude float64) domain.GeoPointV2 {
	latD := spacing / 111_000
	lonD := latD / math.Max(.2, math.Cos(latitude*math.Pi/180))
	center := float64(n-1) / 2
	switch f {
	case "line_abreast":
		return domain.GeoPointV2{(float64(i) - center) * lonD, 0}
	case "wedge":
		side := 1.
		if i%2 == 0 {
			side = -1
		}
		rank := float64((i + 1) / 2)
		return domain.GeoPointV2{side * rank * lonD, -rank * latD * .7}
	case "echelon_left":
		return domain.GeoPointV2{-float64(i) * lonD, -float64(i) * latD * .7}
	case "echelon_right":
		return domain.GeoPointV2{float64(i) * lonD, -float64(i) * latD * .7}
	case "parallel_columns":
		return domain.GeoPointV2{float64(i%2)*lonD - lonD/2, -float64(i/2) * latD}
	case "dispersed_screen":
		return domain.GeoPointV2{(float64(i%3) - 1) * lonD * 1.7, -float64(i/3) * latD * 1.7}
	case "ring", "orbit":
		a := 2 * math.Pi * float64(i) / float64(n)
		return domain.GeoPointV2{math.Cos(a) * lonD, math.Sin(a) * latD}
	default:
		return domain.GeoPointV2{0, -float64(i) * latD}
	}
}
func inferFormation(s, def string) string {
	v := strings.ToLower(s)
	for _, f := range []string{"line_abreast", "wedge", "echelon_left", "echelon_right", "parallel_columns", "dispersed_screen", "ring", "search_grid", "column"} {
		if strings.Contains(strings.ReplaceAll(v, " ", "_"), f) {
			return f
		}
	}
	return def
}
func inferGuidance(s string) string {
	v := strings.ToLower(s)
	for _, k := range []string{"rendezvous", "regroup", "orbit", "return", "hold", "split", "merge", "heading", "search", "patrol"} {
		if strings.Contains(v, k) {
			return k
		}
	}
	return "waypoints"
}
func formationName(f string) string {
	names := map[string]string{"column": "Trail Economy", "line_abreast": "Line Abreast", "wedge": "Adaptive Wedge", "echelon_left": "Echelon Left", "echelon_right": "Echelon Right", "parallel_columns": "Parallel Columns", "dispersed_screen": "Dispersed Screen", "ring": "Protective Ring", "search_grid": "Search Grid"}
	if n := names[f]; n != "" {
		return n
	}
	return strings.ReplaceAll(strings.Title(strings.ReplaceAll(f, "_", " ")), "  ", " ")
}
func maneuvers(kind, formation string) []string {
	return []string{"regroup on reference leader", "transition to " + strings.ReplaceAll(formation, "_", " "), kind, "safe hold on completion"}
}
func routeDistance(route []domain.GeoPointV2) float64 {
	d := 0.
	for i := 1; i < len(route); i++ {
		lat := route[i-1][1] * math.Pi / 180
		dx := (route[i][0] - route[i-1][0]) * 111.32 * math.Cos(lat)
		dy := (route[i][1] - route[i-1][1]) * 110.57
		d += math.Hypot(dx, dy)
	}
	return d
}
func planScore(p domain.FleetPlanV2) float64 {
	if p.PolicyStatus == "prohibited" {
		return -1_000_000
	}
	return p.CoveragePercent*.5 + p.MinimumReserve*100*.3 + math.Max(0, 100-p.DurationMinutes)*.2
}
func shortHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:])[:12] }
func hashAny(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
func hashWithout(v any) string   { return hashAny(v) }
func unique(v []string) []string { return uniqueStrings(v) }
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func sameMembers(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, value := range a {
		if !contains(b, value) {
			return false
		}
	}
	return true
}
func uniqueStrings(v []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range v {
		if x != "" && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
func remove(v []string, x string) []string {
	out := []string{}
	for _, s := range v {
		if s != x {
			out = append(out, s)
		}
	}
	return out
}
func cloneStrings(v []string) []string { return append([]string(nil), v...) }
func clonePoints(v []domain.GeoPointV2) []domain.GeoPointV2 {
	return append([]domain.GeoPointV2(nil), v...)
}

func hasChatMessage(messages []domain.MissionChatMessageV2, id string) bool {
	for _, message := range messages {
		if message.ID == id {
			return true
		}
	}
	return false
}
func nonempty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
}

func (m *Manager) uniqueMissionNameLocked(requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = "Mission 1"
	}
	used := map[string]bool{}
	maxOrdinal := 0
	for _, mission := range m.missions {
		used[strings.ToLower(strings.TrimSpace(mission.Name))] = true
		if match := numberedMissionName.FindStringSubmatch(strings.TrimSpace(mission.Name)); len(match) == 3 {
			if ordinal, err := strconv.Atoi(match[2]); err == nil && ordinal > maxOrdinal {
				maxOrdinal = ordinal
			}
		}
	}
	if !used[strings.ToLower(requested)] {
		return requested
	}
	if match := numberedMissionName.FindStringSubmatch(requested); len(match) == 3 {
		prefix := strings.ToUpper(match[1][:1]) + strings.ToLower(match[1][1:])
		for ordinal := maxOrdinal + 1; ; ordinal++ {
			candidate := fmt.Sprintf("%s %d", prefix, ordinal)
			if !used[strings.ToLower(candidate)] {
				return candidate
			}
		}
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s %d", requested, suffix)
		if !used[strings.ToLower(candidate)] {
			return candidate
		}
	}
}

func (m *Manager) uniqueMissionNameExceptLocked(requested, exceptID string) string {
	requested = strings.TrimSpace(requested)
	used := map[string]bool{}
	for id, mission := range m.missions {
		if id != exceptID {
			used[strings.ToLower(strings.TrimSpace(mission.Name))] = true
		}
	}
	if !used[strings.ToLower(requested)] {
		return requested
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s %d", requested, suffix)
		if !used[strings.ToLower(candidate)] {
			return candidate
		}
	}
}

func (m *Manager) nextMissionNameLocked() string {
	used := map[string]bool{}
	for _, mission := range m.missions {
		used[strings.ToLower(strings.TrimSpace(mission.Name))] = true
	}
	for _, name := range missionNames {
		if !used[strings.ToLower(name)] {
			return name
		}
	}
	return fmt.Sprintf("Harbor Lantern %d", len(m.missions)+1)
}

func (m *Manager) deduplicateMissionNamesLocked() {
	missions := make([]domain.MissionWorkspaceV2, 0, len(m.missions))
	for _, mission := range m.missions {
		missions = append(missions, mission)
	}
	sort.Slice(missions, func(i, j int) bool {
		if missions[i].CreatedAt.Equal(missions[j].CreatedAt) {
			return missions[i].ID < missions[j].ID
		}
		return missions[i].CreatedAt.Before(missions[j].CreatedAt)
	})
	seen := map[string]bool{}
	maxOrdinal := 0
	for _, mission := range missions {
		if match := numberedMissionName.FindStringSubmatch(strings.TrimSpace(mission.Name)); len(match) == 3 {
			if ordinal, err := strconv.Atoi(match[2]); err == nil && ordinal > maxOrdinal {
				maxOrdinal = ordinal
			}
		}
	}
	for _, mission := range missions {
		name := strings.TrimSpace(mission.Name)
		key := strings.ToLower(name)
		if !seen[key] {
			seen[key] = true
			continue
		}
		if match := numberedMissionName.FindStringSubmatch(name); len(match) == 3 {
			prefix := strings.ToUpper(match[1][:1]) + strings.ToLower(match[1][1:])
			for {
				maxOrdinal++
				name = fmt.Sprintf("%s %d", prefix, maxOrdinal)
				if !seen[strings.ToLower(name)] {
					break
				}
			}
		} else {
			for suffix := 2; ; suffix++ {
				candidate := fmt.Sprintf("%s %d", name, suffix)
				if !seen[strings.ToLower(candidate)] {
					name = candidate
					break
				}
			}
		}
		mission.Name = name
		mission.Version++
		mission.UpdatedAt = time.Now().UTC()
		m.missions[mission.ID] = mission
		seen[strings.ToLower(name)] = true
	}
}

func (m *Manager) Voices() []domain.VoiceV2 {
	names := []string{"Anna", "Vera", "Charles", "Paul", "Jeff", "Patrick", "James", "Morgan", "Movie Trailer Voice", "Ian", "Sam", "David"}
	out := make([]domain.VoiceV2, 0, len(names))
	for _, n := range names {
		out = append(out, domain.VoiceV2{ID: strings.ToLower(strings.ReplaceAll(n, " ", "-")), Name: n, Default: n == "Morgan", Available: n == "Morgan"})
	}
	return out
}
func (m *Manager) SpeechCapabilities() domain.SpeechCapabilitiesV2 {
	return domain.SpeechCapabilitiesV2{TTSNode: "vm-214", TTSEngine: "Pocket TTS", TTSVersion: "2.1.0", DefaultVoice: "Morgan", Streaming: true, BargeIn: true, TranscriptionRoutes: []string{"browser-webgpu", "browser-wasm", "colocated-node", "trusted-peer", "typed-input"}, HTTPSRequired: true, DemoLimitations: []string{"TTS service wiring is deployment-optional; visible text is authoritative.", "Browser microphone and WebGPU require HTTPS.", "Logical node isolation is simulated on VM 214."}}
}

func (m *Manager) persistAsync() {
	if m.databaseURL == "" {
		return
	}
	generation := m.persistenceGeneration.Add(1)
	snapshot := m.snapshotLocked()
	go func() {
		m.persistenceMu.Lock()
		defer m.persistenceMu.Unlock()
		if generation != m.persistenceGeneration.Load() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, m.databaseURL)
		if err != nil {
			return
		}
		defer pool.Close()
		for _, v := range snapshot.Vessels {
			b, _ := json.Marshal(v)
			_, _ = pool.Exec(ctx, `INSERT INTO fleet_vessels(id,designation,callsign,class_id,profile) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET designation=EXCLUDED.designation,callsign=EXCLUDED.callsign,class_id=EXCLUDED.class_id,profile=EXCLUDED.profile`, v.ID, v.Designation, v.Callsign, v.Class.ID, b)
		}
		for _, g := range snapshot.Groups {
			b, _ := json.Marshal(g)
			_, _ = pool.Exec(ctx, `INSERT INTO operational_groups(id,revision,payload) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload`, g.ID, g.Revision, b)
		}
		for _, c := range snapshot.Collections {
			b, _ := json.Marshal(c)
			_, _ = pool.Exec(ctx, `INSERT INTO saved_collections(id,revision,payload) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload,updated_at=now()`, c.ID, c.Revision, b)
		}
		for _, v := range snapshot.Missions {
			b, _ := json.Marshal(v)
			_, _ = pool.Exec(ctx, `INSERT INTO mission_workspaces(id,version,status,payload,updated_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET version=EXCLUDED.version,status=EXCLUDED.status,payload=EXCLUDED.payload,updated_at=EXCLUDED.updated_at`, v.ID, v.Version, v.Status, b, v.UpdatedAt)
		}
	}()
}

func (m *Manager) persistProgramAsync(missionID string, program domain.TrajectoryProgramV1) {
	if m.databaseURL == "" {
		return
	}
	go func() {
		m.persistenceMu.Lock()
		defer m.persistenceMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, m.databaseURL)
		if err != nil {
			return
		}
		defer pool.Close()
		payload, _ := json.Marshal(program)
		_, _ = pool.Exec(ctx, `INSERT INTO trajectory_programs(mission_id,active_revision,content_hash,payload,updated_at) VALUES($1,$2,$3,$4,now()) ON CONFLICT(mission_id) DO UPDATE SET active_revision=EXCLUDED.active_revision,content_hash=EXCLUDED.content_hash,payload=EXCLUDED.payload,updated_at=now()`, missionID, program.ActiveRevision, program.ContentHash, payload)
	}()
}
func (m *Manager) clearMissionPersistenceAsync() {
	if m.databaseURL == "" {
		return
	}
	generation := m.persistenceGeneration.Add(1)
	go func() {
		m.persistenceMu.Lock()
		defer m.persistenceMu.Unlock()
		if generation != m.persistenceGeneration.Load() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, m.databaseURL)
		if err != nil {
			return
		}
		defer pool.Close()
		_, _ = pool.Exec(ctx, `TRUNCATE trajectory_programs, mission_command_drafts, fleet_plans, mission_workspaces`)
	}()
}

// deleteMissionPersistence makes workspace deletion durable. Previously the
// in-memory mission disappeared but its PostgreSQL row survived and was loaded
// again after a core restart.
func (m *Manager) deleteMissionPersistence(id string) error {
	if m.databaseURL == "" {
		return nil
	}
	m.persistenceMu.Lock()
	defer m.persistenceMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, m.databaseURL)
	if err != nil {
		m.logger.Warn("mission persistence deletion unavailable", "mission_id", id, "error", err)
		return err
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		m.logger.Warn("mission persistence deletion could not start", "mission_id", id, "error", err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `DELETE FROM mission_command_drafts WHERE mission_id=$1`, id); err == nil {
		_, err = tx.Exec(ctx, `DELETE FROM fleet_plans WHERE mission_id=$1`, id)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `DELETE FROM mission_workspaces WHERE id=$1`, id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		m.logger.Warn("mission persistence deletion failed", "mission_id", id, "error", err)
	}
	return err
}

func (m *Manager) deleteGroupPersistence(id string) error {
	if m.databaseURL == "" {
		return nil
	}
	m.persistenceMu.Lock()
	defer m.persistenceMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, m.databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `DELETE FROM operational_groups WHERE id=$1`, id)
	if err != nil {
		m.logger.Warn("group persistence deletion failed", "group_id", id, "error", err)
	}
	return err
}
func (m *Manager) loadPersistent(ctx context.Context) {
	if m.databaseURL == "" {
		return
	}
	pool, err := pgxpool.New(ctx, m.databaseURL)
	if err != nil {
		m.logger.Warn("M6 persistence unavailable", "error", err)
		return
	}
	defer pool.Close()
	m.mu.Lock()
	defer m.mu.Unlock()
	load := func(query string, apply func([]byte)) {
		rows, queryErr := pool.Query(ctx, query)
		if queryErr != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var b []byte
			if rows.Scan(&b) == nil {
				apply(b)
			}
		}
	}
	load(`SELECT profile FROM fleet_vessels ORDER BY designation`, func(b []byte) {
		var v domain.VesselProfileV2
		if json.Unmarshal(b, &v) == nil && v.ID != "" {
			if seeded, ok := m.vessels[v.ID]; ok {
				v = migrateLegacySpawn(v, seeded)
			}
			// Current simulated nodes all expose an inference route. Future physical
			// deployments may persist false for nodes without a GPU/provider.
			v.DecisionCapable = true
			m.vessels[v.ID] = v
		}
	})
	load(`SELECT payload FROM operational_groups ORDER BY id`, func(b []byte) {
		var g domain.OperationalGroupV2
		if json.Unmarshal(b, &g) == nil && g.ID != "" {
			m.groups[g.ID] = g
		}
	})
	load(`SELECT payload FROM saved_collections ORDER BY id`, func(b []byte) {
		var c domain.SavedCollectionV2
		if json.Unmarshal(b, &c) == nil && c.ID != "" {
			m.collections[c.ID] = c
		}
	})
	load(`SELECT payload FROM mission_workspaces ORDER BY updated_at`, func(b []byte) {
		var v domain.MissionWorkspaceV2
		if json.Unmarshal(b, &v) == nil && v.ID != "" {
			m.missions[v.ID] = v
		}
	})
	load(`SELECT payload FROM trajectory_programs ORDER BY mission_id`, func(b []byte) {
		var program domain.TrajectoryProgramV1
		if json.Unmarshal(b, &program) == nil && program.MissionID != "" && trajectory.ValidateRevision(program.Revisions[program.ActiveRevision], m.secret) {
			m.programs[program.MissionID] = program
		}
	})
	for id, group := range m.groups {
		if !validFormation(group.Formation) {
			group.Formation = "column"
		}
		if group.FormationSpacingM < 15 {
			group.FormationSpacingM = 60
		}
		if group.AssemblyPoint == nil && len(group.MemberIDs) > 0 {
			if vessel, ok := m.vessels[group.MemberIDs[0]]; ok {
				point := vessel.Telemetry.Position
				group.AssemblyPoint = &point
				group.AssemblySource = "first-member"
			}
		}
		group = m.groupDecisionSnapshotLocked(group)
		m.groups[id] = group
	}
	for id, vessel := range m.vessels {
		if vessel.GroupID != "" {
			if _, ok := m.groups[vessel.GroupID]; !ok {
				clearVesselGroup(&vessel)
				m.vessels[id] = vessel
			}
		}
	}
	m.deduplicateMissionNamesLocked()
	m.persistAsync()
}

var _ = errors.New
