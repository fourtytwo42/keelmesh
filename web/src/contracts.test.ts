import { describe, expect, it } from "vitest";
import fixtureJSON from "../../contracts/fixtures/mission-intent-v1.json";
import resilienceJSON from "../../contracts/fixtures/resilience-snapshot-v1.json";
import platformJSON from "../../contracts/fixtures/platform-snapshot-v1.json";
import type { MissionIntent, PlatformSnapshot, ResilienceSnapshot } from "./types";

describe("shared contract fixtures", () => {
  it("reads MissionIntentV1 using the TypeScript contract", () => {
    const fixture = fixtureJSON as MissionIntent;
    expect(fixture.schema_version).toBe(1);
    expect(fixture.requested_asset_count).toBe(6);
    expect(fixture.area.type).toBe("Polygon");
    expect(fixture.constraints.avoid_zones).toEqual(["exclusion-2"]);
  });

  it("reads ResilienceSnapshotV1 using the TypeScript contract", () => {
    const fixture = resilienceJSON as ResilienceSnapshot;
    expect(fixture.scenario_id).toBe("resilient-edge-v1");
    expect(fixture.incident_node_id).toBe("vessel-04");
    expect(fixture.active_path).toEqual(["operator", "vessel-04"]);
  });

  it("reads PlatformSnapshotV1 using the TypeScript contract", () => {
    const fixture = platformJSON as PlatformSnapshot;
    expect(fixture.active_run?.vessel_count).toBe(1000);
    expect(fixture.topics[0].partitions).toBe(12);
    expect(fixture.metrics.attempted).toBe(fixture.metrics.unique_inserted + fixture.metrics.duplicates_suppressed + fixture.metrics.quarantined);
  });
});
