import { describe, expect, it } from "vitest";
import fixtureJSON from "../../contracts/fixtures/mission-intent-v1.json";
import type { MissionIntent } from "./types";

describe("shared contract fixtures", () => {
  it("reads MissionIntentV1 using the TypeScript contract", () => {
    const fixture = fixtureJSON as MissionIntent;
    expect(fixture.schema_version).toBe(1);
    expect(fixture.requested_asset_count).toBe(6);
    expect(fixture.area.type).toBe("Polygon");
    expect(fixture.constraints.avoid_zones).toEqual(["exclusion-2"]);
  });
});
