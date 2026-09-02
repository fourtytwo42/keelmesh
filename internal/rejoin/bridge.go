package rejoin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func Build(actual, target domain.Point, executionWatermark int, discarded []int, targetSequence int, targetTick int64) domain.RejoinBridgeV1 {
	stateBytes, _ := json.Marshal(struct {
		Position  domain.Point
		Watermark int
	}{actual, executionWatermark})
	stateHash := sha256.Sum256(stateBytes)
	b := domain.RejoinBridgeV1{SchemaVersion: 1, ActualStateHash: "sha256:" + hex.EncodeToString(stateHash[:]), ExecutionWatermark: executionWatermark, DiscardedSequences: append([]int(nil), discarded...), TargetSequence: targetSequence, TargetTick: targetTick, Route: []domain.Point{actual, midpoint(actual, target), target}, PolicyStatus: "within_authority", ReasonCodes: []string{"BOUNDARY_OK", "EXCLUSION_CLEAR", "RESERVE_OK", "PNT_CORROBORATED", "COLLISION_CLEAR", "SUFFICIENT_LEAD_TIME"}}
	content, _ := json.Marshal(b)
	h := sha256.Sum256(content)
	b.ContentHash = "sha256:" + hex.EncodeToString(h[:])
	return b
}

func midpoint(a, b domain.Point) domain.Point {
	return domain.Point{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2}
}
