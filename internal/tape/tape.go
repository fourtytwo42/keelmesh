package tape

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

const SegmentSeconds int64 = 10

func BuildSix(missionID, leaseID, planID, planHash string, startSequence int, startTick int64, route []domain.Point, speed, reserve float64, key []byte) []domain.MissionTapeSegmentV1 {
	segments := make([]domain.MissionTapeSegmentV1, 0, 6)
	predecessor := ""
	for i := 0; i < 6; i++ {
		segment := domain.MissionTapeSegmentV1{
			SchemaVersion: domain.SchemaVersion, MissionID: missionID, LeaseID: leaseID,
			PlanID: planID, PlanHash: planHash, Revision: 1, Sequence: startSequence + i,
			ActivationTick: startTick + int64(i)*SegmentSeconds, ExpiryTick: startTick + int64(i+1)*SegmentSeconds,
			PredecessorHash: predecessor, RouteCorridor: sampleCorridor(route, i), SpeedMinMPS: 0,
			SpeedMaxMPS: speed, ExpectedPosition: samplePoint(route, i+1), MinimumReserve: reserve,
			MaximumUncertaintyM: 45, FailureBehavior: "safe_hold", Lifecycle: "armed",
		}
		segment.ContentHash = Hash(segment)
		segment.Signature = Sign(segment, key)
		segments = append(segments, segment)
		predecessor = segment.ContentHash
	}
	return segments
}

func Hash(segment domain.MissionTapeSegmentV1) string {
	segment.ContentHash = ""
	segment.Signature = ""
	b, _ := json.Marshal(segment)
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

func Sign(segment domain.MissionTapeSegmentV1, key []byte) string {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write([]byte(Hash(segment)))
	return "hmac-sha256:" + hex.EncodeToString(m.Sum(nil))
}

func Validate(segment domain.MissionTapeSegmentV1, predecessor string, tick int64, leaseID, planHash string, uncertainty float64, key []byte) error {
	if segment.ContentHash != Hash(segment) || !hmac.Equal([]byte(segment.Signature), []byte(Sign(segment, key))) {
		return fmt.Errorf("SEGMENT_HASH_MISMATCH")
	}
	if segment.PredecessorHash != predecessor {
		return fmt.Errorf("PREDECESSOR_MISMATCH")
	}
	if segment.LeaseID != leaseID || segment.PlanHash != planHash {
		return fmt.Errorf("INACTIVE_LEASE")
	}
	if tick >= segment.ExpiryTick {
		return fmt.Errorf("SEGMENT_EXPIRED")
	}
	if segment.ExpiryTick-segment.ActivationTick != SegmentSeconds || len(segment.RouteCorridor) < 2 || segment.SpeedMaxMPS <= 0 || segment.SpeedMinMPS < 0 || segment.SpeedMinMPS > segment.SpeedMaxMPS {
		return fmt.Errorf("SEGMENT_ENVELOPE_INVALID")
	}
	if uncertainty > segment.MaximumUncertaintyM {
		return fmt.Errorf("UNSAFE_PNT")
	}
	return nil
}

func Transition(current, next string) error {
	allowed := map[string]map[string]bool{
		"received":  {"validated": true, "rejected": true, "expired": true},
		"validated": {"armed": true, "rejected": true, "expired": true},
		"armed":     {"started": true, "expired": true, "preempted": true},
		"started":   {"completed": true, "skipped": true, "preempted": true},
	}
	if terminal(current) || !allowed[current][next] {
		return fmt.Errorf("INVALID_SEGMENT_TRANSITION")
	}
	return nil
}

func Advance(segments []domain.MissionTapeSegmentV1, tick int64, executing bool) (out []domain.MissionTapeSegmentV1, executionWatermark int) {
	out = append([]domain.MissionTapeSegmentV1(nil), segments...)
	executionWatermark = -1
	for i := range out {
		s := &out[i]
		if terminal(s.Lifecycle) {
			if s.Lifecycle == "completed" {
				executionWatermark = s.Sequence
			}
			continue
		}
		switch {
		case tick >= s.ExpiryTick:
			if executing {
				s.Lifecycle = "completed"
				executionWatermark = s.Sequence
			} else {
				s.Lifecycle = "expired"
			}
		case tick >= s.ActivationTick && executing:
			s.Lifecycle = "started"
		default:
			s.Lifecycle = "armed"
		}
	}
	return out, executionWatermark
}

func Summary(segments []domain.MissionTapeSegmentV1, tick int64) domain.TapeSummaryV1 {
	depth := int64(0)
	for _, s := range segments {
		if !terminal(s.Lifecycle) && s.ExpiryTick > tick {
			start := s.ActivationTick
			if start < tick {
				start = tick
			}
			depth += s.ExpiryTick - start
		}
	}
	watermark := "empty"
	switch {
	case depth >= 45:
		watermark = "full"
	case depth >= 15:
		watermark = "low"
	case depth > 0:
		watermark = "critical"
	}
	return domain.TapeSummaryV1{DepthSeconds: int(depth), Watermark: watermark, Segments: append([]domain.MissionTapeSegmentV1(nil), segments...)}
}

func terminal(state string) bool {
	return state == "completed" || state == "expired" || state == "rejected" || state == "skipped" || state == "preempted"
}

func samplePoint(route []domain.Point, part int) domain.Point {
	if len(route) == 0 {
		return domain.Point{}
	}
	idx := part * (len(route) - 1) / 6
	return route[idx]
}

func sampleCorridor(route []domain.Point, part int) []domain.Point {
	return []domain.Point{samplePoint(route, part), samplePoint(route, part+1)}
}
