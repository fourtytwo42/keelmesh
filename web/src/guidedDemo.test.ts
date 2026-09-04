import { describe, expect, it } from "vitest";
import { guidedDemoBeats, guidedDemoEstimatedSeconds } from "./guidedDemo";

describe("guided demo release contract", () => {
  it("keeps both personas on the same concise, truthful sequence", () => {
    expect(guidedDemoBeats).toHaveLength(13);
    expect(new Set(guidedDemoBeats.map((beat) => beat.id)).size).toBe(guidedDemoBeats.length);
    expect(guidedDemoEstimatedSeconds).toBeLessThanOrEqual(300);
    for (const beat of guidedDemoBeats) {
      expect(beat.audio.navy).toBe(`/assets/demo/navy/${beat.id}.mp3`);
      expect(beat.audio.pirate).toBe(`/assets/demo/pirate/${beat.id}.mp3`);
      expect(beat.transcript.navy.length).toBeGreaterThan(100);
      expect(beat.transcript.pirate.length).toBeGreaterThan(100);
    }
    const closing = guidedDemoBeats.at(-1)!;
    expect(closing.transcript.navy).toContain("simulated");
    expect(closing.transcript.navy).toContain("real Raft");
    expect(closing.transcript.pirate).toContain("Physical radios");
  });
});
