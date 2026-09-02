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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
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
	Name      *string   `json:"name"`
	Color     *string   `json:"color"`
	Pattern   *string   `json:"pattern"`
	MemberIDs *[]string `json:"member_ids"`
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
	Name      string   `json:"name"`
	Objective string   `json:"objective"`
	TargetIDs []string `json:"target_ids"`
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
	IncludedAreas  [][][]float64         `json:"included_areas"`
	ExclusionAreas [][][]float64         `json:"exclusion_areas"`
	Waypoints      []domain.GeoPointV2   `json:"waypoints"`
	POIs           []domain.MissionPOIV2 `json:"pois"`
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
	mu           sync.RWMutex
	logger       *slog.Logger
	databaseURL  string
	secret       []byte
	fleetVersion int64
	vessels      map[string]domain.VesselProfileV2
	groups       map[string]domain.OperationalGroupV2
	collections  map[string]domain.SavedCollectionV2
	missions     map[string]domain.MissionWorkspaceV2
	drafts       map[string]domain.CommandDraftV2
	plans        map[string]domain.FleetPlanV2
	leases       map[string]domain.FleetLeaseV2
	idempotency  map[string]string
	startedPlans map[string]string
}

var callsigns = []string{"Gannet", "Osprey", "Tern", "Petrel", "Shearwater", "Cormorant", "Harrier", "Kite", "Merlin", "Plover", "Skua", "Fulmar", "Albatross", "Razorbill", "Puffin", "Heron", "Kittiwake", "Curlew", "Jaeger", "Avocet", "Sanderling", "Grebe", "Dunlin", "Egret", "Bittern", "Sandpiper", "Stormbird", "Kingfisher", "Loon", "Murre", "Nighthawk", "Pelican", "Rail", "Sparrowhawk", "Turnstone", "Whimbrel", "Auk", "Bunting", "Caspian", "Diver", "Eider", "Frigate", "Godwit", "Hobby", "Ibis", "Junco", "Lapwing", "Merganser"}
var groupNames = []string{"Watch Shoal", "Bay Lantern", "Sakonnet", "Block Guard", "Brenton", "Narragansett", "Point Judith", "Ocean State"}
var groupCodes = []string{"WS", "BL", "SK", "BG", "BR", "NG", "PJ", "OS"}
var groupColors = []string{"#e9a93f", "#62c5a8", "#d86f5f", "#b895d8", "#7eb4df", "#d2c05d", "#df8fb0", "#8fca72"}
var patterns = []string{"solid", "diagonal", "dots", "crosshatch", "vertical", "rings", "dash", "chevron"}

func New(databaseURL string, logger *slog.Logger) *Manager {
	m := &Manager{logger: logger, databaseURL: databaseURL, secret: []byte("keelmesh-m6-runtime-authority"), fleetVersion: 1, vessels: map[string]domain.VesselProfileV2{}, groups: map[string]domain.OperationalGroupV2{}, collections: map[string]domain.SavedCollectionV2{}, missions: map[string]domain.MissionWorkspaceV2{}, drafts: map[string]domain.CommandDraftV2{}, plans: map[string]domain.FleetPlanV2{}, leases: map[string]domain.FleetLeaseV2{}, idempotency: map[string]string{}, startedPlans: map[string]string{}}
	m.seed()
	return m
}

func (m *Manager) Run(ctx context.Context) {
	m.loadPersistent(ctx)
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
	centers := []domain.GeoPointV2{{-71.375, 41.49}, {-71.315, 41.47}, {-71.24, 41.45}, {-71.43, 41.39}, {-71.33, 41.37}, {-71.23, 41.35}, {-71.47, 41.25}, {-71.30, 41.20}}
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
			p := domain.GeoPointV2{centers[g][0] + float64(slot%3-1)*.022, centers[g][1] + float64(slot/3)*.028}
			env := environmentAt(p, float64(idx))
			m.vessels[id] = domain.VesselProfileV2{SchemaVersion: 2, ID: id, Designation: fmt.Sprintf("KM-%03d", 214+idx), Callsign: callsigns[idx], DisplayName: fmt.Sprintf("%s (KM-%03d)", callsigns[idx], 214+idx), Class: class, GroupID: gid, GroupCode: groupCodes[g], GroupColor: groupColors[g], GroupPattern: patterns[g], Available: true, Telemetry: domain.VesselTelemetryV2{Position: p, HeadingDeg: float64((idx * 37) % 360), SpeedMPS: .4 + float64(idx%5)*.11, Reserve: .96 - float64(idx%9)*.025, ProjectedReserve: .89 - float64(idx%9)*.025, Mode: "patrol", Health: "nominal", PNTIntegrity: "trusted", UncertaintyM: 4 + float64(idx%5), TapeDepthSeconds: 60, Environment: env}}
		}
		m.groups[gid] = domain.OperationalGroupV2{SchemaVersion: 2, ID: gid, Code: groupCodes[g], Name: groupNames[g], Color: groupColors[g], Pattern: patterns[g], MemberIDs: members, Revision: 1}
	}
	all := make([]string, 0, 48)
	for id := range m.vessels {
		all = append(all, id)
	}
	sort.Strings(all)
	m.collections["collection-relays"] = domain.SavedCollectionV2{SchemaVersion: 2, ID: "collection-relays", Name: "Atlas relay watch", MemberIDs: filterIDs(all, func(v domain.VesselProfileV2) bool { return v.Class.ID == "atlas" }, m.vessels), Revision: 1}
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
func (m *Manager) Reachability(id string) (domain.ReachabilityV2, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vessels[id]
	if !ok {
		return domain.ReachabilityV2{}, &Error{"VESSEL_NOT_FOUND", "Vessel not found."}
	}
	r := domain.ReachabilityV2{SchemaVersion: 2, VesselID: id, Authority: "mission-scoped authority"}
	g := m.groups[v.GroupID]
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
		if og.ID != g.ID && len(r.ExternalPeers) < 3 {
			peer := og.MemberIDs[5]
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
	id := "group-custom-" + shortHash(req.IdempotencyKey)
	g := domain.OperationalGroupV2{SchemaVersion: 2, ID: id, Code: fmt.Sprintf("C%02d", len(m.groups)-7), Name: req.Name, Color: req.Color, Pattern: req.Pattern, MemberIDs: unique(req.MemberIDs), Revision: 1}
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
		g.Name = *req.Name
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
					return g, &Error{"PRIMARY_GROUP_REQUIRED", "Move vessels by creating or selecting their destination group; a vessel cannot be left without a primary group."}
				}
			}
			for _, vid := range nextMembers {
				vessel := m.vessels[vid]
				if vessel.GroupID != id {
					old := m.groups[vessel.GroupID]
					old.MemberIDs = remove(old.MemberIDs, vid)
					old.Revision++
					m.groups[old.ID] = old
				}
			}
			g.MemberIDs = nextMembers
		}
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
func (m *Manager) DeleteGroup(id string, req Mutation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[id]
	if !ok {
		return &Error{"GROUP_NOT_FOUND", "Group not found."}
	}
	if strings.HasPrefix(id, "group-0") {
		return &Error{"HARDWARE_POLICY", "Seed groups cannot be dissolved; move members instead."}
	}
	if err := m.check(req, g.Revision); err != nil {
		return err
	}
	if len(g.MemberIDs) > 0 {
		return &Error{"GROUP_NOT_EMPTY", "Move members before dissolving the group."}
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
	mission := domain.MissionWorkspaceV2{SchemaVersion: 2, ID: "mission-" + shortHash(req.IdempotencyKey), Name: req.Name, Objective: req.Objective, Status: "draft", TargetIDs: targets, TargetSnapshotHash: hashAny(targets), FleetVersion: m.fleetVersion, Version: 1, Geometry: domain.MissionGeometryV2{Revision: 1, IncludedAreas: [][][]float64{}, ExclusionAreas: [][][]float64{}, Waypoints: []domain.GeoPointV2{}, POIs: []domain.MissionPOIV2{}}, Constraints: defaultConstraints(), Formation: "column", CreatedAt: now, UpdatedAt: now}
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
		v.Name = *req.Name
	}
	if req.Objective != nil {
		v.Objective = *req.Objective
	}
	if req.Status != nil {
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
	v.AuthorizedPlanID = ""
	m.missions[id] = v
	m.persistAsync()
	return v, nil
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
	v.Geometry = domain.MissionGeometryV2{Revision: v.Geometry.Revision + 1, IncludedAreas: req.IncludedAreas, ExclusionAreas: req.ExclusionAreas, Waypoints: req.Waypoints, POIs: req.POIs}
	v.Version++
	v.UpdatedAt = time.Now().UTC()
	v.AuthorizedPlanID = ""
	m.missions[id] = v
	m.persistAsync()
	return v, nil
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
	amb := []string{}
	if len(wps) == 0 && len(mission.Geometry.IncludedAreas) == 0 {
		amb = append(amb, "Choose an area or waypoint before route generation.")
	}
	draft := domain.CommandDraftV2{SchemaVersion: 2, ID: "draft-" + shortHash(req.IdempotencyKey), MissionID: id, SourceText: req.Text, Objective: nonempty(req.Text, mission.Objective), TargetIDs: targets, TargetSnapshotHash: hashAny(targets), GeometryRevision: mission.Geometry.Revision, FleetVersion: m.fleetVersion, Constraints: mission.Constraints, FormationPreference: formation, GuidanceKind: kind, Waypoints: wps, Ambiguities: amb}
	draft.ContentHash = hashWithout(draft)
	m.drafts[draft.ID] = draft
	return draft, nil
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
	formations := uniqueStrings([]string{draft.FormationPreference, "wedge", "line_abreast", "dispersed_screen"})
	if len(formations) > 3 {
		formations = formations[:3]
	}
	out := make([]domain.FleetPlanV2, 0, len(formations))
	for i, f := range formations {
		p := m.makePlan(mission, draft, f, i)
		m.plans[p.ID] = p
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return planScore(out[i]) > planScore(out[j]) })
	for i := range out {
		out[i].Recommended = i == 0
		p := out[i]
		m.plans[p.ID] = p
	}
	mission.PlanIDs = nil
	for _, p := range out {
		mission.PlanIDs = append(mission.PlanIDs, p.ID)
	}
	mission.Status = "planned"
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
	mission.Status = "authorized"
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
	mission.Status = "executing"
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	m.missions[mid] = mission
	for _, a := range p.Assignments {
		v := m.vessels[a.VesselID]
		v.Telemetry.Route = clonePoints(a.Route)
		v.Telemetry.MissionID = mid
		v.Telemetry.Mode = "mission"
		v.Telemetry.SpeedMPS = a.SpeedMPS
		v.Telemetry.ProjectedReserve = p.MinimumReserve
		m.vessels[v.ID] = v
	}
	m.fleetVersion++
	return mission, nil
}

func (m *Manager) makePlan(mission domain.MissionWorkspaceV2, draft domain.CommandDraftV2, formation string, index int) domain.FleetPlanV2 {
	targets := cloneStrings(draft.TargetIDs)
	sort.Strings(targets)
	speed := math.Min(draft.Constraints.MaximumSpeedMPS, 1.45+float64(index)*.15)
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
			off := formationOffset(formation, i, len(targets), draft.Constraints.FormationSpacingM)
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
	duration := maxDistance * 1000 / speed / 60
	coverage := math.Min(99, 72+float64(len(targets))*2.4-float64(index)*1.6)
	minSep := draft.Constraints.FormationSpacingM
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
	p := domain.FleetPlanV2{SchemaVersion: 2, ID: fmt.Sprintf("plan-%s-%d", shortHash(draft.ContentHash), index+1), MissionID: mission.ID, DraftID: draft.ID, Name: formationName(formation), Formation: formation, Maneuvers: maneuvers(draft.GuidanceKind, formation), Assignments: assignments, CoveragePercent: coverage, MinimumReserve: minReserve, DurationMinutes: duration, EnergyKWH: total * (1.8 + speed*.4), LinkExposureSeconds: duration * 60 * .08 * float64(index+1), MinimumSeparationM: minSep, PolicyStatus: status, ReasonCodes: reasons, SourceMissionVersion: mission.Version}
	p.ContentHash = hashWithout(p)
	return p
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
	changed := false
	for mid, mission := range m.missions {
		if mission.Status != "executing" {
			continue
		}
		active := false
		for id, v := range m.vessels {
			if v.Telemetry.MissionID != mid || len(v.Telemetry.Route) < 2 {
				continue
			}
			active = true
			target := v.Telemetry.Route[1]
			step := v.Telemetry.SpeedMPS * .2 / 111_000
			dx, dy := target[0]-v.Telemetry.Position[0], target[1]-v.Telemetry.Position[1]
			d := math.Hypot(dx, dy)
			if d <= step {
				v.Telemetry.Position = target
				v.Telemetry.Route = v.Telemetry.Route[1:]
			} else {
				v.Telemetry.Position[0] += dx / d * step
				v.Telemetry.Position[1] += dy / d * step
			}
			v.Telemetry.HeadingDeg = math.Mod(math.Atan2(dx, dy)*180/math.Pi+360, 360)
			v.Telemetry.Reserve = math.Max(0, v.Telemetry.Reserve-.000006)
			v.Telemetry.Environment = environmentAt(v.Telemetry.Position, float64(time.Now().Unix()%100))
			m.vessels[id] = v
			changed = true
		}
		if !active {
			mission.Status = "completed"
			mission.Version++
			mission.UpdatedAt = time.Now().UTC()
			m.missions[mid] = mission
		}
	}
	if changed {
		m.fleetVersion++
	}
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
	return a
}
func formationOffset(f string, i, n int, spacing float64) domain.GeoPointV2 {
	d := spacing / 111_000
	center := float64(n-1) / 2
	switch f {
	case "line_abreast":
		return domain.GeoPointV2{(float64(i) - center) * d, 0}
	case "wedge":
		side := 1.
		if i%2 == 0 {
			side = -1
		}
		rank := float64((i + 1) / 2)
		return domain.GeoPointV2{side * rank * d, -rank * d * .7}
	case "echelon_left":
		return domain.GeoPointV2{-float64(i) * d, -float64(i) * d * .7}
	case "echelon_right":
		return domain.GeoPointV2{float64(i) * d, -float64(i) * d * .7}
	case "parallel_columns":
		return domain.GeoPointV2{float64(i%2)*d - float64(1)*d/2, -float64(i/2) * d}
	case "dispersed_screen":
		return domain.GeoPointV2{(float64(i%3) - 1) * d * 1.7, -float64(i/3) * d * 1.7}
	case "ring", "orbit":
		a := 2 * math.Pi * float64(i) / float64(n)
		return domain.GeoPointV2{math.Cos(a) * d, math.Sin(a) * d}
	default:
		return domain.GeoPointV2{0, -float64(i) * d}
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
	for _, k := range []string{"rendezvous", "regroup", "orbit", "return", "hold", "split", "merge", "heading", "search"} {
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
func nonempty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
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
	snapshot := m.snapshotLocked()
	go func() {
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
func (m *Manager) clearMissionPersistenceAsync() {
	if m.databaseURL == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, m.databaseURL)
		if err != nil {
			return
		}
		defer pool.Close()
		_, _ = pool.Exec(ctx, `TRUNCATE mission_command_drafts, fleet_plans, mission_workspaces`)
	}()
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
	m.persistAsync()
}

var _ = errors.New
