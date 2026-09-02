package mesh

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type Deduplicator struct{ seen map[string]string }

func NewDeduplicator() *Deduplicator { return &Deduplicator{seen: map[string]string{}} }

func NewBundle(id, idempotencyKey, missionID, planHash, payloadHash string, tick int64, key []byte) domain.PeerBundleV1 {
	b := domain.PeerBundleV1{SchemaVersion: 1, ID: id, IdempotencyKey: idempotencyKey, OriginID: "operator", DestinationID: "vessel-04", MessageClass: "mission_segment", Priority: 100, AuthorityEpoch: 1, MissionID: missionID, PlanHash: planHash, CreatedTick: tick, ExpiresTick: tick + 30, HopLimit: 3, PayloadSchema: "MissionTapeSegmentV1", PayloadHash: payloadHash}
	b.ContentHash = bundleHash(b)
	b.OriginSignature = bundleSign(b, key)
	return b
}

func ValidateBundle(bundle domain.PeerBundleV1, tick int64, hops int, key []byte) error {
	if bundle.ContentHash != bundleHash(bundle) || !hmac.Equal([]byte(bundle.OriginSignature), []byte(bundleSign(bundle, key))) {
		return fmt.Errorf("BUNDLE_HASH_MISMATCH")
	}
	if tick >= bundle.ExpiresTick {
		return fmt.Errorf("BUNDLE_EXPIRED")
	}
	if hops > bundle.HopLimit || hops > 3 {
		return fmt.Errorf("HOP_LIMIT_EXCEEDED")
	}
	if bundle.OriginID != "operator" || bundle.DestinationID != "vessel-04" {
		return fmt.Errorf("MUTATED_RELAY")
	}
	return nil
}

func (d *Deduplicator) Deliver(bundle domain.PeerBundleV1) (bool, error) {
	if previous, ok := d.seen[bundle.IdempotencyKey]; ok {
		if previous != bundle.ContentHash {
			return false, fmt.Errorf("IDEMPOTENCY_CONFLICT")
		}
		return false, nil
	}
	d.seen[bundle.IdempotencyKey] = bundle.ContentHash
	return true, nil
}

func bundleHash(bundle domain.PeerBundleV1) string {
	bundle.ContentHash, bundle.OriginSignature = "", ""
	b, _ := json.Marshal(bundle)
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
func bundleSign(bundle domain.PeerBundleV1, key []byte) string {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write([]byte(bundleHash(bundle)))
	return "hmac-sha256:" + hex.EncodeToString(m.Sum(nil))
}

func Healthy() []domain.LinkStateV1 {
	links := []domain.LinkStateV1{
		link("operator-v4-starlink", "operator", "vessel-04", "starlink", true, 58, 0.4, "high", 1),
		link("operator-v3-starlink", "operator", "vessel-03", "starlink", true, 52, 0.3, "high", 1),
		link("v3-v4-halow", "vessel-03", "vessel-04", "halow", true, 18, 0.8, "medium", .3),
		link("v4-v3-halow", "vessel-04", "vessel-03", "halow", true, 18, 0.8, "medium", .3),
	}
	return links
}

func FailDirect(links []domain.LinkStateV1, tick int64) []domain.LinkStateV1 {
	out := clone(links)
	for i := range out {
		if out[i].ID == "operator-v4-starlink" {
			out[i].Reachable, out[i].LastTransitionTick = false, tick
		}
	}
	return out
}

func PartitionV4(links []domain.LinkStateV1, tick int64) []domain.LinkStateV1 {
	out := clone(links)
	for i := range out {
		if out[i].DestinationID == "vessel-04" || out[i].SourceID == "vessel-04" {
			out[i].Reachable, out[i].LastTransitionTick = false, tick
		}
	}
	return out
}

func RestoreHaLow(links []domain.LinkStateV1, tick int64) []domain.LinkStateV1 {
	out := clone(links)
	for i := range out {
		if out[i].Underlay == "halow" {
			out[i].Reachable, out[i].LastTransitionTick = true, tick
		}
	}
	return out
}

func RelayPath(links []domain.LinkStateV1) []string {
	if reachable(links, "operator-v3-starlink") && reachable(links, "v3-v4-halow") {
		return []string{"operator", "vessel-03", "vessel-04"}
	}
	return nil
}

func Advertisements(tick int64) []domain.EgressAdvertisementV1 {
	return []domain.EgressAdvertisementV1{{SchemaVersion: 1, NodeID: "vessel-03", Internet: true, ExpiresTick: tick + 15, CapacityClass: "high", Sequence: tick/15 + 1, Signature: "sim-authenticated:vessel-03"}}
}

func link(id, from, to, underlay string, up bool, latency int, loss float64, capacity string, energy float64) domain.LinkStateV1 {
	return domain.LinkStateV1{SchemaVersion: 1, ID: id, SourceID: from, DestinationID: to, Underlay: underlay, Reachable: up, LatencyMS: latency, LossPercent: loss, CapacityClass: capacity, Trusted: true, EnergyCost: energy}
}
func reachable(links []domain.LinkStateV1, id string) bool {
	for _, l := range links {
		if l.ID == id {
			return l.Reachable
		}
	}
	return false
}
func clone(in []domain.LinkStateV1) []domain.LinkStateV1 {
	return append([]domain.LinkStateV1(nil), in...)
}
