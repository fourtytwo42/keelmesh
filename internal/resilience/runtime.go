package resilience

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	missionclock "github.com/fourtytwo42/keelmesh/internal/clock"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/mesh"
	"github.com/fourtytwo42/keelmesh/internal/pnt"
	"github.com/fourtytwo42/keelmesh/internal/rejoin"
	"github.com/fourtytwo42/keelmesh/internal/tape"
)

const (
	FaultFailStarlink = "fail_starlink"
	FaultPartition    = "partition_vessel4"
	FaultGNSSSpoof    = "inject_gnss_spoof"
	FaultRestore      = "restore_contact"
)

type Runtime struct {
	clock      *missionclock.MissionClock
	snapshot   domain.ResilienceSnapshotV1
	key        []byte
	lease      domain.MissionLeaseV1
	plan       domain.PlanCandidateV1
	segments   map[string][]domain.MissionTapeSegmentV1
	positions  map[string]domain.Point
	watermarks map[string]int
}

func New(lease domain.MissionLeaseV1, plan domain.PlanCandidateV1, vessels []domain.VesselV1, stateVersion int64) *Runtime {
	seed := sha256.Sum256([]byte(lease.PlanHash + "|keelmesh-m2-segment-authority"))
	r := &Runtime{clock: missionclock.New(0), key: seed[:], lease: lease, plan: plan, segments: map[string][]domain.MissionTapeSegmentV1{}, positions: map[string]domain.Point{}, watermarks: map[string]int{}}
	assignments := map[string]domain.AssignmentV1{}
	for _, a := range plan.Assignments {
		assignments[a.VesselID] = a
	}
	for _, vessel := range vessels {
		r.positions[vessel.ID] = vessel.Position
		a := assignments[vessel.ID]
		r.segments[vessel.ID] = tape.BuildSix(lease.MissionID, lease.ID, plan.ID, plan.ContentHash, 0, 0, a.Route, a.SpeedMPS, lease.MinReserve, r.key)
		r.watermarks[vessel.ID] = -1
	}
	r.snapshot = domain.ResilienceSnapshotV1{SchemaVersion: domain.SchemaVersion, StateVersion: stateVersion, ScenarioID: "resilient-edge-v1", Phase: "ready", IncidentNodeID: "vessel-04", RelayNodeID: "vessel-03", Links: mesh.Healthy(), Advertisements: []domain.EgressAdvertisementV1{}, ActivePath: []string{"operator", "vessel-04"}, HopReceipts: []domain.HopReceiptV1{}, DiscardedSequences: []int{}, AutoRunAvailable: true, Summary: "All nodes have direct reachability and a validated 60-second mission tape.", NextAction: FaultFailStarlink}
	r.snapshot.PNTTransitions = []domain.PntEstimateV1{pnt.Trusted(r.positions["vessel-04"])}
	r.refreshNodes()
	return r
}

func (r *Runtime) Snapshot() domain.ResilienceSnapshotV1 {
	b, _ := json.Marshal(r.snapshot)
	var out domain.ResilienceSnapshotV1
	_ = json.Unmarshal(b, &out)
	return out
}

func (r *Runtime) SetStateVersion(version int64) { r.snapshot.StateVersion = version }

func (r *Runtime) SetPosition(nodeID string, position domain.Point) {
	r.positions[nodeID] = position
	for i := range r.snapshot.Nodes {
		if r.snapshot.Nodes[i].ID == nodeID {
			r.snapshot.Nodes[i].Position = position
			r.snapshot.Nodes[i].PNT.Position = position
		}
	}
}

func (r *Runtime) CanMove(nodeID string) bool {
	if nodeID != r.snapshot.IncidentNodeID {
		return true
	}
	return r.snapshot.Phase != "safe_hold"
}

func (r *Runtime) Apply(kind string) (string, error) {
	switch kind {
	case FaultFailStarlink:
		if r.snapshot.Phase != "ready" {
			return "", fmt.Errorf("INVALID_FAULT_SEQUENCE")
		}
		r.snapshot.Links = mesh.FailDirect(r.snapshot.Links, r.clock.Tick())
		r.snapshot.Advertisements = mesh.Advertisements(r.clock.Tick())
		r.snapshot.ActivePath = mesh.RelayPath(r.snapshot.Links)
		bundle := mesh.NewBundle("bundle-segment-v4-06", "segment-v4-06", r.lease.MissionID, r.plan.ContentHash, r.segments["vessel-04"][5].ContentHash, r.clock.Tick(), r.key)
		if err := mesh.ValidateBundle(bundle, r.clock.Tick(), 2, r.key); err != nil {
			return "", err
		}
		dedup := mesh.NewDeduplicator()
		_, _ = dedup.Deliver(bundle)
		duplicateAccepted, _ := dedup.Deliver(bundle)
		r.snapshot.HopReceipts = []domain.HopReceiptV1{
			{SchemaVersion: 1, BundleID: "bundle-segment-v4-06", RelayID: "vessel-03", IngressLinkID: "operator-v3-starlink", EgressLinkID: "v3-v4-halow", ObservedTick: 0, Result: "forwarded"},
			{SchemaVersion: 1, BundleID: "bundle-segment-v4-06", RelayID: "vessel-04", IngressLinkID: "v3-v4-halow", ObservedTick: 0, Result: "deduplicated"},
		}
		if !duplicateAccepted {
			r.snapshot.DuplicateDeliveries = 1
		}
		r.snapshot.Phase, r.snapshot.Summary, r.snapshot.NextAction = "relayed", "Vessel 4 direct Starlink failed. Authenticated traffic switched to Vessel 3 peer egress after route hysteresis.", FaultPartition
		return "resilience.link.relay_selected", nil
	case FaultPartition:
		if r.snapshot.Phase != "relayed" {
			return "", fmt.Errorf("INVALID_FAULT_SEQUENCE")
		}
		r.clock.Advance(30)
		r.snapshot.Links = mesh.PartitionV4(r.snapshot.Links, r.clock.Tick())
		r.snapshot.ActivePath = []string{}
		r.snapshot.Advertisements = nil
		r.snapshot.QueuedBundles = 3
		r.snapshot.Phase, r.snapshot.Summary, r.snapshot.NextAction = "partitioned", "Vessel 4 is fully partitioned. It continues only validated onboard authority; bulk telemetry is suppressed.", FaultGNSSSpoof
		r.advanceTapes(true)
		r.refreshNodes()
		r.setIncidentPNT(pnt.Partitioned(r.positions["vessel-04"]), "dead_reckoning", 7)
		r.snapshot.PNTTransitions = append(r.snapshot.PNTTransitions, pnt.Partitioned(r.positions["vessel-04"]))
		return "resilience.partition.entered", nil
	case FaultGNSSSpoof:
		if r.snapshot.Phase != "partitioned" {
			return "", fmt.Errorf("INVALID_FAULT_SEQUENCE")
		}
		r.clock.Advance(30)
		r.advanceTapes(true)
		estimate, ghost := pnt.Spoof(r.positions["vessel-04"])
		r.snapshot.RawGNSSPosition = &ghost
		r.snapshot.Phase, r.snapshot.Summary, r.snapshot.NextAction = "safe_hold", "GNSS jump was excluded before fusion. With no corroboration and an empty tape, Vessel 4 entered bounded zero-speed safe hold.", FaultRestore
		r.refreshNodes()
		r.setIncidentPNT(estimate, "safe_hold", 14)
		denied := estimate
		denied.UncertaintyM, denied.Integrity, denied.Behavior = 41, "denied", "reduced_speed"
		denied.ReasonCodes = []string{"GNSS_EXCLUDED", "CORROBORATION_DEGRADED"}
		r.snapshot.PNTTransitions = append(r.snapshot.PNTTransitions, denied, estimate)
		return "resilience.pnt.safe_hold", nil
	case FaultRestore:
		if r.snapshot.Phase != "safe_hold" {
			return "", fmt.Errorf("INVALID_FAULT_SEQUENCE")
		}
		r.clock.Advance(15)
		r.snapshot.Links = mesh.RestoreHaLow(r.snapshot.Links, r.clock.Tick())
		r.snapshot.ActivePath = mesh.RelayPath(r.snapshot.Links)
		r.snapshot.Advertisements = mesh.Advertisements(r.clock.Tick())
		r.snapshot.DiscardedSequences = []int{6, 7, 8}
		r.snapshot.QueuedBundles = 0
		actual := r.positions["vessel-04"]
		target := targetPoint(r.plan, "vessel-04")
		bridge := rejoin.Build(actual, target, r.watermarks["vessel-04"], r.snapshot.DiscardedSequences, 9, 90)
		r.snapshot.Bridge = &bridge
		a := assignment(r.plan, "vessel-04")
		r.segments["vessel-04"] = tape.BuildSix(r.lease.MissionID, r.lease.ID, r.plan.ID, r.plan.ContentHash, 9, 90, append([]domain.Point{actual}, a.Route...), a.SpeedMPS, r.lease.MinReserve, r.key)
		r.snapshot.Phase, r.snapshot.Summary, r.snapshot.NextAction = "rejoined", "High-water marks reconciled, stale work expired, and a policy-valid bridge targets future segment 9 without replay or position jump.", ""
		r.refreshNodes()
		r.setIncidentPNT(pnt.Recovered(actual), "rejoined", 16)
		r.snapshot.PNTTransitions = append(r.snapshot.PNTTransitions, pnt.Recovered(actual))
		return "resilience.bridge.activated", nil
	default:
		return "", fmt.Errorf("INVALID_FAULT")
	}
}

func (r *Runtime) Advance(seconds int64) {
	r.clock.Advance(seconds)
	r.advanceTapes(r.snapshot.Phase != "safe_hold")
	r.refreshNodes()
}

func (r *Runtime) advanceTapes(executing bool) {
	for nodeID, segments := range r.segments {
		updated, watermark := tape.Advance(segments, r.clock.Tick(), executing)
		r.segments[nodeID] = updated
		if watermark > r.watermarks[nodeID] {
			r.watermarks[nodeID] = watermark
		}
	}
	r.snapshot.MissionTick = r.clock.Tick()
}

func (r *Runtime) refreshNodes() {
	nodes := make([]domain.NodeSnapshotV1, 0, 7)
	nodes = append(nodes, domain.NodeSnapshotV1{SchemaVersion: 1, ID: "operator", Name: "Operator Station", Behavior: "command", ActiveRoute: append([]string(nil), r.snapshot.ActivePath...), PNT: pnt.Trusted(domain.Point{})})
	for i := 1; i <= 6; i++ {
		id := fmt.Sprintf("vessel-%02d", i)
		behavior := "mission"
		estimate := pnt.Trusted(r.positions[id])
		bufferedEvents := 0
		if id == "vessel-04" {
			switch r.snapshot.Phase {
			case "partitioned":
				behavior, estimate, bufferedEvents = "dead_reckoning", pnt.Partitioned(r.positions[id]), 7
			case "safe_hold":
				estimate, _ = pnt.Spoof(r.positions[id])
				behavior, bufferedEvents = "safe_hold", 14
			case "rejoined":
				behavior, estimate, bufferedEvents = "rejoined", pnt.Recovered(r.positions[id]), 0
			}
		}
		nodes = append(nodes, domain.NodeSnapshotV1{SchemaVersion: 1, ID: id, Name: fmt.Sprintf("Vessel %d", i), Position: r.positions[id], Behavior: behavior, ActiveLeaseID: r.lease.ID, Tape: tape.Summary(r.segments[id], r.clock.Tick()), ActiveRoute: append([]string(nil), r.snapshot.ActivePath...), BufferedBundles: r.snapshot.QueuedBundles, BufferedEvents: bufferedEvents, ExecutionWatermark: r.watermarks[id], PNT: estimate, PNTObservations: pnt.Observations(r.positions[id], r.snapshot.Phase, r.clock.Tick()), LocalSequence: r.clock.Tick()})
	}
	r.snapshot.MissionTick = r.clock.Tick()
	r.snapshot.Nodes = nodes
}

func (r *Runtime) setIncidentPNT(estimate domain.PntEstimateV1, behavior string, events int) {
	for i := range r.snapshot.Nodes {
		if r.snapshot.Nodes[i].ID == "vessel-04" {
			r.snapshot.Nodes[i].PNT = estimate
			r.snapshot.Nodes[i].Behavior = behavior
			r.snapshot.Nodes[i].BufferedEvents = events
		}
	}
}

func assignment(plan domain.PlanCandidateV1, vesselID string) domain.AssignmentV1 {
	for _, a := range plan.Assignments {
		if a.VesselID == vesselID {
			return a
		}
	}
	return domain.AssignmentV1{}
}
func targetPoint(plan domain.PlanCandidateV1, vesselID string) domain.Point {
	a := assignment(plan, vesselID)
	if len(a.Route) == 0 {
		return domain.Point{}
	}
	return a.Route[len(a.Route)*2/3]
}
