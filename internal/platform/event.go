package platform

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type controlCommand struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	TargetID  string            `json:"target_id,omitempty"`
	Run       *domain.LoadRunV1 `json:"run,omitempty"`
	Signature string            `json:"signature"`
	CreatedAt time.Time         `json:"created_at"`
}

func checksumPayload(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func signCommand(command controlCommand, secret string) string {
	command.Signature = ""
	data, _ := json.Marshal(command)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
func validCommand(command controlCommand, secret string) bool {
	expected := signCommand(command, secret)
	return hmac.Equal([]byte(expected), []byte(command.Signature))
}

func makeEnvelope(run domain.LoadRunV1, vesselIndex int, sequence int64, producedAt time.Time) domain.EventEnvelopeV1 {
	vesselID := fmt.Sprintf("sim-%04d", vesselIndex+1)
	payload, _ := json.Marshal(map[string]any{"lat": 41.8100 + float64(vesselIndex%50)*0.0001, "lon": -70.5220 + float64(vesselIndex/50)*0.0001, "heading_deg": (sequence*7 + int64(vesselIndex)) % 360, "speed_mps": 3.2, "reserve": 0.72, "padding": "keelmesh-measured-payload-000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"})
	e := domain.EventEnvelopeV1{SchemaVersion: 1, EventID: fmt.Sprintf("%s-%s-%09d", run.ID, vesselID, sequence), LogicalKey: vesselID, FleetID: "background-fleet", VesselID: vesselID, Sequence: sequence, Type: "vessel.telemetry", PayloadSchema: 1, TraceID: fmt.Sprintf("trace-%s-%09d", run.ID, sequence), RunID: run.ID, ProducedAt: producedAt, Payload: payload}
	e.Checksum = checksumPayload(payload)
	return e
}
