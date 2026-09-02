package clock

import "sync"

// MissionClock is an injectable monotonic clock. It deliberately has no wall
// clock or GNSS input, so external time can never extend authority.
type MissionClock struct {
	mu   sync.RWMutex
	tick int64
}

func New(start int64) *MissionClock { return &MissionClock{tick: start} }

func (c *MissionClock) Tick() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tick
}

func (c *MissionClock) Advance(seconds int64) int64 {
	if seconds < 0 {
		return c.Tick()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tick += seconds
	return c.tick
}

func (c *MissionClock) SetForTest(tick int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if tick >= c.tick {
		c.tick = tick
	}
}
