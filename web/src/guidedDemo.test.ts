import { describe, expect, it } from "vitest";
import { guidedDemoBeats, guidedDemoEstimatedSeconds } from "./guidedDemo";

describe("guided demo release contract", () => {
  it("keeps both personas on the same concise, truthful sequence", () => {
    expect(guidedDemoBeats).toHaveLength(13);
    expect(new Set(guidedDemoBeats.map((beat) => beat.id)).size).toBe(guidedDemoBeats.length);
    expect(guidedDemoEstimatedSeconds).toBeGreaterThanOrEqual(360);
    expect(guidedDemoEstimatedSeconds).toBeLessThanOrEqual(540);
    for (const beat of guidedDemoBeats) {
      const navyRevision = beat.id === "06-execution" ? "?v=20260909b" : "";
      expect(beat.audio.navy).toBe(`/assets/demo/navy/${beat.id}.mp3${navyRevision}`);
      expect(beat.audio.pirate).toBe(`/assets/demo/pirate/${beat.id}.mp3`);
      expect(beat.transcript.navy.length).toBeGreaterThan(100);
      expect(beat.transcript.pirate.length).toBeGreaterThan(100);
    }
    const closing = guidedDemoBeats.at(-1)!;
    const execution = guidedDemoBeats.find((beat) => beat.id === "06-execution")!;
    const resilience = guidedDemoBeats.find((beat) => beat.id === "08-resilience")!;
    expect(execution.focus).toBe("execution");
    expect(execution.transcript.navy).toContain("complete finite signed program");
    expect(execution.transcript.navy).toContain("authorization-expiration tick");
    expect(resilience.transcript.navy).toContain("decision scope from group to local");
    expect(resilience.transcript.navy).toContain("program revision");
    expect(guidedDemoBeats.flatMap((beat) => [beat.transcript.navy, beat.transcript.pirate]).join(" ").toLowerCase()).not.toContain("cached approved work");
    expect(closing.transcript.navy).toContain("simulated");
    expect(closing.transcript.navy).toContain("real Raft");
    expect(closing.transcript.pirate).toContain("Physical radios");
  });
});
