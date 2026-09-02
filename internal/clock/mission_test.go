package clock

import "testing"

func TestClockNeverMovesBackward(t *testing.T) {
	c := New(10)
	c.Advance(5)
	c.SetForTest(3)
	if got := c.Tick(); got != 15 {
		t.Fatalf("tick = %d, want 15", got)
	}
}
