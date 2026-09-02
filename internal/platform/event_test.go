package platform

import (
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestEnvelopeIsDeterministicAndChecksummed(t *testing.T) {
	run := domain.LoadRunV1{ID: "run-fixture", Seed: 424242, VesselCount: 1000, RateHz: 2}
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	first := makeEnvelope(run, 42, 7, at)
	second := makeEnvelope(run, 42, 7, at)
	if first.EventID != second.EventID || first.Checksum != second.Checksum || string(first.Payload) != string(second.Payload) {
		t.Fatal("same seed coordinates and sequence must produce the same envelope")
	}
	if checksumPayload(first.Payload) != first.Checksum {
		t.Fatal("payload checksum does not validate")
	}
}
