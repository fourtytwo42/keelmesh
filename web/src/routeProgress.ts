import type { Point } from "./types";

const METRES_PER_DEGREE_LATITUDE = 111_000;

export type RouteProjection = {
  point: Point;
  progressM: number;
  totalM: number;
  distanceM: number;
  segmentIndex: number;
};

function segmentVector(origin: Point, target: Point): [number, number] {
  const latitudeScale = Math.max(
    0.2,
    Math.cos(((origin[1] + target[1]) / 2) * Math.PI / 180),
  );
  return [
    (target[0] - origin[0]) * METRES_PER_DEGREE_LATITUDE * latitudeScale,
    (target[1] - origin[1]) * METRES_PER_DEGREE_LATITUDE,
  ];
}

export function routeLengthM(route: Point[]): number {
  let total = 0;
  for (let index = 1; index < route.length; index += 1) {
    const [east, north] = segmentVector(route[index - 1], route[index]);
    total += Math.hypot(east, north);
  }
  return total;
}

export function projectOntoRoute(
  route: Point[],
  position: Point,
  minimumProgressM = 0,
): RouteProjection | null {
  if (route.length < 2) return null;
  const totalM = routeLengthM(route);
  let traversedM = 0;
  let best: RouteProjection | null = null;

  for (let index = 1; index < route.length; index += 1) {
    const start = route[index - 1];
    const end = route[index];
    const [segmentEast, segmentNorth] = segmentVector(start, end);
    const segmentLengthM = Math.hypot(segmentEast, segmentNorth);
    if (segmentLengthM < 0.001) continue;
    if (traversedM + segmentLengthM < minimumProgressM) {
      traversedM += segmentLengthM;
      continue;
    }
    const [pointEast, pointNorth] = segmentVector(start, position);
    const unconstrained =
      (pointEast * segmentEast + pointNorth * segmentNorth) /
      (segmentLengthM * segmentLengthM);
    const minimumFraction = Math.max(
      0,
      Math.min(1, (minimumProgressM - traversedM) / segmentLengthM),
    );
    const fraction = Math.max(minimumFraction, Math.min(1, unconstrained));
    const projected: Point = [
      start[0] + (end[0] - start[0]) * fraction,
      start[1] + (end[1] - start[1]) * fraction,
    ];
    const [errorEast, errorNorth] = segmentVector(projected, position);
    const candidate: RouteProjection = {
      point: projected,
      progressM: Math.min(totalM, traversedM + fraction * segmentLengthM),
      totalM,
      distanceM: Math.hypot(errorEast, errorNorth),
      segmentIndex: index - 1,
    };
    if (!best || candidate.distanceM < best.distanceM) best = candidate;
    traversedM += segmentLengthM;
  }
  return best;
}

export function remainingRoute(
  route: Point[],
  position: Point,
  minimumProgressM = 0,
): { coordinates: Point[]; projection: RouteProjection | null } {
  const projection = projectOntoRoute(route, position, minimumProgressM);
  if (!projection || projection.totalM - projection.progressM <= 1) {
    return { coordinates: [], projection };
  }
  return {
    coordinates: [projection.point, ...route.slice(projection.segmentIndex + 1)],
    projection,
  };
}
