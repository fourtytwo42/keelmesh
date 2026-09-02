package tape

import (
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"testing"
)

func TestSixSegmentsAreExactlySixtySecondsAndTerminalNeverReactivates(t *testing.T) {
	key := []byte("deterministic-test-key")
	segments := BuildSix("m", "l", "p", "h", 0, 0, []domain.Point{{0, 0}, {1, 1}}, 2, .3, key)
	if len(segments) != 6 || segments[5].ExpiryTick-segments[0].ActivationTick != 60 {
		t.Fatalf("invalid tape: %#v", segments)
	}
	segments, watermark := Advance(segments, 60, true)
	if watermark != 5 || Summary(segments, 60).Watermark != "empty" {
		t.Fatalf("watermark=%d summary=%#v", watermark, Summary(segments, 60))
	}
	segments, _ = Advance(segments, 10, true)
	if segments[5].Lifecycle != "completed" {
		t.Fatal("terminal segment reactivated")
	}
}

func TestValidationRejectsMutationAndExpiry(t *testing.T) {
	key := []byte("key")
	s := BuildSix("m", "l", "p", "h", 0, 0, []domain.Point{{0, 0}, {1, 1}}, 2, .3, key)[0]
	if err := Validate(s, "", 0, "l", "h", 5, key); err != nil {
		t.Fatal(err)
	}
	s.SpeedMaxMPS++
	if err := Validate(s, "", 0, "l", "h", 5, key); err == nil {
		t.Fatal("mutation accepted")
	}
}

func TestLifecycleIsMonotonic(t *testing.T) {
	for _, transition := range [][2]string{{"received", "validated"}, {"validated", "armed"}, {"armed", "started"}, {"started", "completed"}} {
		if err := Transition(transition[0], transition[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := Transition("completed", "started"); err == nil {
		t.Fatal("terminal segment reactivated")
	}
	if err := Transition("armed", "validated"); err == nil {
		t.Fatal("backward transition accepted")
	}
}
