import { describe, expect, it } from "vitest";
import type { Point } from "./types";
import { projectOntoRoute, remainingRoute, routeLengthM } from "./routeProgress";

describe("remaining mission route", () => {
  const route: Point[] = [[-71.5, 41], [-71.49, 41], [-71.48, 41]];

  it("removes traveled coordinates and begins at the projected vessel pose", () => {
    const result = remainingRoute(route, [-71.485, 41]);
    expect(result.coordinates).toHaveLength(2);
    expect(result.coordinates[0][0]).toBeCloseTo(-71.485, 5);
    expect(result.coordinates.at(-1)).toEqual(route.at(-1));
  });

  it("uses execution progress as a monotonic floor at crossing routes", () => {
    const crossing: Point[] = [[0, 0], [0.01, 0.01], [0, 0.01], [0.01, 0]];
    const floor = routeLengthM(crossing) * 0.72;
    const projected = projectOntoRoute(crossing, [0.005, 0.005], floor);
    expect(projected?.progressM).toBeGreaterThanOrEqual(floor);
  });

  it("removes the line after its final metre is consumed", () => {
    expect(remainingRoute(route, route.at(-1)!).coordinates).toEqual([]);
  });
});
