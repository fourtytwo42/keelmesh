import { type MouseEvent as ReactMouseEvent, useEffect, useMemo, useRef, useState } from "react";
import * as maplibregl from "maplibre-gl";
import type {
  GeoJSONSource,
  Map as MLMap,
  MapMouseEvent,
  StyleSpecification,
} from "maplibre-gl";
import maplibreWorkerURL from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";
import "maplibre-gl/dist/maplibre-gl.css";
import type {
  FleetPlanV2,
  FleetSnapshotV2,
  MissionWaypointV2,
  MissionWorkspaceV2,
  Point,
  SurfaceContactV2,
  SceneMapAnnotationV1,
} from "./types";
import { projectOntoRoute, remainingRoute, routeLengthM } from "./routeProgress";

maplibregl.setWorkerUrl(maplibreWorkerURL);
type Tool = "select" | "box" | "waypoint" | "include" | "exclude" | "hold" | "orbit";
export type WaypointColor = MissionWaypointV2["color"];
type ContextMenu =
  | { kind: "vessel"; x: number; y: number; vessel: string; group: string }
  | { kind: "contact"; x: number; y: number; contact: string }
  | { kind: "location"; x: number; y: number; point: Point; surface: "land" | "water"; depth?: number };
type Props = {
  fleet: FleetSnapshotV2;
  mission: MissionWorkspaceV2 | null;
  selected: Set<string>;
  activePlan: FleetPlanV2 | null;
  activeMissionPlans: Array<{
    mission: MissionWorkspaceV2;
    plan: FleetPlanV2;
  }>;
  tool: Tool;
  editingEnabled: boolean;
  focusedGeometry: { kind: "waypoint" | "include" | "exclude" | "poi"; index: number } | null;
  pirate: boolean;
  onSelect: (ids: string[], mode: "replace" | "toggle") => void;
  onGroup: (id: string) => void;
  onVessel: (id: string) => void;
  onOpenFleet: () => void;
  onContact: (id: string) => void;
  onPlanContact: (contactID: string) => void;
  onGeometryFocus: (kind: "waypoint" | "include" | "exclude" | "poi", index: number) => void;
  onWaypoint: (p: Point, color: WaypointColor) => void;
  onPOI: (kind: "hold" | "orbit", p: Point) => void;
  onMoveWaypoint: (index: number, p: Point) => void;
  onMovePOI: (index: number, p: Point) => void;
  onMoveGroupAssembly: (groupID: string, p: Point) => void;
  onHoldGroupAtVessel: (groupID: string, vesselID: string) => void;
  onArea: (kind: "include" | "exclude", polygon: Point[]) => void;
  onToolDone: () => void;
  sceneAnnotations: SceneMapAnnotationV1[];
  sceneCamera?: { center: Point; zoom: number; bearing: number; pitch: number };
};
const empty: GeoJSON.FeatureCollection = {
  type: "FeatureCollection",
  features: [],
};
function sceneAnnotationData(values: SceneMapAnnotationV1[]): GeoJSON.FeatureCollection {
  return { type: "FeatureCollection", features: values.flatMap((value) => {
    if (value.points.length === 0) return [];
    return [{ type: "Feature" as const, properties: { id: value.id, kind: value.kind, label: value.label, color: value.color }, geometry: value.points.length === 1 ? { type: "Point" as const, coordinates: value.points[0] } : { type: "LineString" as const, coordinates: value.points } }];
  }) };
}
const style: StyleSpecification = {
  version: 8,
  sources: {
    coast: { type: "geojson", data: "/assets/maps/narragansett.geojson" },
  },
  layers: [
    { id: "sea", type: "background", paint: { "background-color": "#0b2731" } },
    {
      id: "bathymetry",
      type: "background",
      paint: { "background-color": "#12333d", "background-opacity": 0.78 },
    },
    {
      id: "land",
      type: "fill",
      source: "coast",
      filter: ["==", ["get", "kind"], "land"],
      paint: {
        "fill-color": "#30362f",
        "fill-opacity": 0.96,
        "fill-outline-color": "#718076",
      },
    },
    {
      id: "shipping",
      type: "line",
      source: "coast",
      filter: ["==", ["get", "kind"], "shipping"],
      paint: {
        "line-color": "#9f8b55",
        "line-width": 1,
        "line-dasharray": [3, 4],
        "line-opacity": 0.55,
      },
    },
    {
      id: "labels",
      type: "symbol",
      source: "coast",
      filter: ["==", ["get", "kind"], "label"],
      layout: {
        "text-field": ["get", "name"],
        "text-size": 11,
        "text-letter-spacing": 0.08,
      },
      paint: {
        "text-color": "#b9b5a8",
        "text-halo-color": "#151c1d",
        "text-halo-width": 1.5,
      },
    },
  ],
};

function vesselData(fleet: FleetSnapshotV2): GeoJSON.FeatureCollection {
  return {
    type: "FeatureCollection",
    features: fleet.vessels.map((v) => ({
      type: "Feature",
      id: v.id,
      properties: {
        id: v.id,
        name: v.callsign,
        designation: v.designation,
        group: v.group_id,
        groupCode: v.group_code,
        color: v.group_color,
        class: v.class.id,
        heading: v.telemetry.heading_deg,
        reserve: Math.round(v.telemetry.reserve * 100),
        mode: v.telemetry.mode,
      },
      geometry: { type: "Point", coordinates: v.telemetry.position },
    })),
  };
}
function surfaceContactData(fleet: FleetSnapshotV2): GeoJSON.FeatureCollection {
  return {
    type: "FeatureCollection",
    features: fleet.surface_contacts.map((contact) => ({
      type: "Feature",
      id: contact.id,
      properties: {
        id: contact.id,
        name: contact.name,
        boatID: contact.boat_id,
        class: contact.class,
        color: contact.color,
        colorName: contact.color_name,
        heading: contact.heading_deg,
        speed: contact.speed_knots,
        scale: ["container", "tanker"].includes(contact.class)
          ? 1.45
          : contact.class === "trawler" || contact.class === "yacht"
            ? 0.78
            : 1,
      },
      geometry: { type: "Point", coordinates: contact.position },
    })),
  };
}
function surfaceRouteData(
  contacts: SurfaceContactV2[],
): GeoJSON.FeatureCollection {
  return {
    type: "FeatureCollection",
    features: contacts.filter((contact) => contact.speed_mps > 0 && contact.route.length > 1).map((contact) => ({
      type: "Feature",
      properties: { id: contact.id, color: contact.color },
      geometry: { type: "LineString", coordinates: contact.route },
    })),
  };
}
function routeData(
  plan: FleetPlanV2 | null,
  mission: MissionWorkspaceV2 | null,
  fleet: FleetSnapshotV2,
): GeoJSON.FeatureCollection {
  if (!plan || !mission || ["completed", "ended"].includes(mission.status)) {
    return { type: "FeatureCollection", features: [] };
  }
  if (plan.continuous_tracking && plan.follow_contact_id) {
    const contact = fleet.surface_contacts.find((item) => item.id === plan.follow_contact_id);
    if (contact) {
      return {
        type: "FeatureCollection",
        features: plan.assignments.flatMap((assignment) => {
          const vessel = fleet.vessels.find((item) => item.id === assignment.vessel_id);
          return vessel ? [{
            type: "Feature" as const,
            properties: { vessel: assignment.vessel_id, tracking: true },
            geometry: { type: "LineString" as const, coordinates: [vessel.telemetry.position, contact.position] },
          }] : [];
        }),
      };
    }
  }
  return {
    type: "FeatureCollection",
    features: plan.assignments.flatMap((assignment) => {
      const vessel = fleet.vessels.find((item) => item.id === assignment.vessel_id);
      if (!vessel || !["executing", "paused"].includes(mission.status)) {
        return assignment.route.length > 1 ? [{
          type: "Feature" as const,
          properties: { vessel: assignment.vessel_id },
          geometry: { type: "LineString" as const, coordinates: assignment.route },
        }] : [];
      }
      const cursor = mission.trajectory?.execution[assignment.vessel_id];
      const totalM = routeLengthM(assignment.route);
      const durationSeconds = Math.max(10, Math.ceil(totalM / Math.max(0.1, assignment.speed_mps)));
      const segmentCount = Math.max(1, Math.ceil(durationSeconds / 10));
      const minimumProgressM = cursor
        ? totalM * Math.min(1, cursor.sequence / segmentCount)
        : 0;
      const remaining = remainingRoute(
        assignment.route,
        vessel.telemetry.position,
        minimumProgressM,
      ).coordinates;
      return remaining.length > 1 ? [{
        type: "Feature" as const,
        properties: { vessel: assignment.vessel_id, consumed: true },
        geometry: { type: "LineString" as const, coordinates: remaining },
      }] : [];
    }),
  };
}

const metresPerDegreeLatitude = 111_000;

function projectPoint(point: Point, headingDeg: number, distanceM: number): Point {
  const heading = headingDeg * Math.PI / 180;
  const latitudeScale = Math.max(0.2, Math.cos(point[1] * Math.PI / 180));
  return [
    point[0] + Math.sin(heading) * distanceM / (metresPerDegreeLatitude * latitudeScale),
    point[1] + Math.cos(heading) * distanceM / metresPerDegreeLatitude,
  ];
}

function localVector(origin: Point, target: Point): [number, number] {
  const latitudeScale = Math.max(0.2, Math.cos(origin[1] * Math.PI / 180));
  return [
    (target[0] - origin[0]) * metresPerDegreeLatitude * latitudeScale,
    (target[1] - origin[1]) * metresPerDegreeLatitude,
  ];
}

function geoDistanceM(origin: Point, target: Point): number {
  const [east, north] = localVector(origin, target);
  return Math.hypot(east, north);
}

function interceptSeconds(origin: Point, target: Point, targetHeading: number, targetSpeed: number, pursuerSpeed: number): number | null {
  const [east, north] = localVector(origin, target);
  const heading = targetHeading * Math.PI / 180;
  const velocityEast = Math.sin(heading) * targetSpeed;
  const velocityNorth = Math.cos(heading) * targetSpeed;
  const a = velocityEast ** 2 + velocityNorth ** 2 - pursuerSpeed ** 2;
  const b = 2 * (east * velocityEast + north * velocityNorth);
  const c = east ** 2 + north ** 2;
  const candidates: number[] = [];
  if (Math.abs(a) < 1e-6) {
    if (Math.abs(b) > 1e-6) candidates.push(-c / b);
  } else {
    const discriminant = b ** 2 - 4 * a * c;
    if (discriminant >= 0) {
      const root = Math.sqrt(discriminant);
      candidates.push((-b - root) / (2 * a), (-b + root) / (2 * a));
    }
  }
  const positive = candidates.filter((value) => Number.isFinite(value) && value >= 0).sort((left, right) => left - right)[0];
  return positive !== undefined && positive <= 86_400 ? positive : null;
}

function formatETA(seconds: number | null): string {
  if (seconds === null) return "NO INTERCEPT";
  const rounded = Math.max(0, Math.ceil(seconds));
  const hours = Math.floor(rounded / 3600);
  const minutes = Math.floor((rounded % 3600) / 60);
  const remainder = rounded % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`
    : `${minutes}:${String(remainder).padStart(2, "0")}`;
}

function contactMissionOverlayData(
  missionPlans: Array<{
    mission: MissionWorkspaceV2;
    plan: FleetPlanV2;
  }>,
  fleet: FleetSnapshotV2,
): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = [];
  for (const { mission, plan } of missionPlans) {
    if (! ["authorized", "executing", "paused"].includes(mission.status) || !plan.follow_contact_id) continue;
    const contact = fleet.surface_contacts.find((item) => item.id === plan.follow_contact_id);
    const vessels = mission.target_ids.flatMap((id) => {
      const vessel = fleet.vessels.find((item) => item.id === id);
      return vessel ? [vessel] : [];
    });
    if (!contact || vessels.length === 0) continue;

    const origin: Point = [
      vessels.reduce((sum, vessel) => sum + vessel.telemetry.position[0], 0) / vessels.length,
      vessels.reduce((sum, vessel) => sum + vessel.telemetry.position[1], 0) / vessels.length,
    ];
    const group = fleet.groups.find((item) => item.member_ids.some((id) => mission.target_ids.includes(id)));
    const color = group?.color ?? "#e9a93f";
    const standoff = Math.max(0, plan.contact_standoff_m ?? mission.contact_standoff_m ?? 0);
    const currentRendezvous = projectPoint(contact.position, contact.heading_deg + 180, standoff);
    let eta: number | null = 0;
    for (const vessel of vessels) {
      const assignment = plan.assignments.find((item) => item.vessel_id === vessel.id);
      const speed = Math.max(0.1, assignment?.speed_mps ?? vessel.telemetry.speed_mps);
      const vesselETA = interceptSeconds(vessel.telemetry.position, currentRendezvous, contact.heading_deg, contact.speed_mps, speed);
      if (vesselETA === null) {
        eta = null;
        break;
      }
      eta = Math.max(eta ?? 0, vesselETA);
    }
    const predictedContact = eta === null
      ? contact.position
      : projectPoint(contact.position, contact.heading_deg, contact.speed_mps * eta);
    const destination = projectPoint(predictedContact, contact.heading_deg + 180, standoff);
    const midpoint: Point = [(origin[0] + destination[0]) / 2, (origin[1] + destination[1]) / 2];
    const distanceNM = geoDistanceM(origin, currentRendezvous) / 1852;
    const groupLabel = group ? `${group.color_name.toUpperCase()} ${group.code}` : mission.name.toUpperCase();
    const etaText = `ETA ${formatETA(eta)} · ${distanceNM.toFixed(1)} NM`;
    features.push({
      type: "Feature",
      properties: { kind: "rendezvous-route", color, label: `${groupLabel} → ${contact.name.toUpperCase()}`, missionID: mission.id },
      geometry: { type: "LineString", coordinates: [origin, destination] },
    }, {
      type: "Feature",
      properties: { kind: "rendezvous-eta", color, label: etaText, missionID: mission.id },
      geometry: { type: "Point", coordinates: midpoint },
    });
    if (contact.speed_mps > 0.05 && geoDistanceM(contact.position, predictedContact) > 10) {
      features.push({
        type: "Feature",
        properties: { kind: "contact-prediction", color: contact.color, missionID: mission.id },
        geometry: { type: "LineString", coordinates: [contact.position, predictedContact] },
      });
    }
  }
  return { type: "FeatureCollection", features };
}
function geometryData(
  mission: MissionWorkspaceV2 | null,
  plan: FleetPlanV2 | null,
  fleet: FleetSnapshotV2,
): GeoJSON.FeatureCollection {
  if (!mission || ["completed", "ended"].includes(mission.status)) {
    return { type: "FeatureCollection", features: [] };
  }
  const waypoints = mission?.geometry.waypoints ?? [],
    details = mission?.geometry.waypoint_details ?? [];
  const consumedWaypointIndexes = new Set<number>();
  if (plan && ["executing", "paused"].includes(mission.status)) {
    waypoints.forEach((waypoint, waypointIndex) => {
      const ownerGroupID = details[waypointIndex]?.owner_group_id;
      const relevantAssignments = plan.assignments.filter((assignment) => {
        if (!ownerGroupID) return mission.target_ids.includes(assignment.vessel_id);
        return fleet.groups.find((group) => group.id === ownerGroupID)?.member_ids.includes(assignment.vessel_id);
      });
      if (relevantAssignments.length === 0) return;
      const passedByAll = relevantAssignments.every((assignment) => {
        const vessel = fleet.vessels.find((item) => item.id === assignment.vessel_id);
        if (!vessel || assignment.route.length < 2) return false;
        const totalM = routeLengthM(assignment.route);
        const durationSeconds = Math.max(10, Math.ceil(totalM / Math.max(0.1, assignment.speed_mps)));
        const segmentCount = Math.max(1, Math.ceil(durationSeconds / 10));
        const cursor = mission.trajectory?.execution[assignment.vessel_id];
        const floor = cursor ? totalM * Math.min(1, cursor.sequence / segmentCount) : 0;
        const vesselProgress = projectOntoRoute(assignment.route, vessel.telemetry.position, floor);
        const waypointProgress = projectOntoRoute(assignment.route, waypoint);
        return !!vesselProgress && !!waypointProgress &&
          waypointProgress.distanceM <= Math.max(250, mission.constraints.formation_spacing_m * 3) &&
          vesselProgress.progressM >= waypointProgress.progressM - 3;
      });
      if (passedByAll) consumedWaypointIndexes.add(waypointIndex);
    });
  }
  const routes = new Map<string, MissionWaypointV2[]>();
  details.forEach((waypoint, index) => {
    if (consumedWaypointIndexes.has(index)) return;
    const key = waypoint.owner_group_id || `color:${waypoint.color}`;
    const existing = routes.get(key) ?? [];
    existing.push({ ...waypoint, position: waypoints[index] ?? waypoint.position });
    routes.set(key, existing);
  });
  return {
    type: "FeatureCollection",
    features: [
      ...(mission?.geometry.included_areas ?? []).map((coordinates, index) => ({
        type: "Feature" as const,
        properties: { kind: "include", index },
        geometry: { type: "Polygon" as const, coordinates: [coordinates] },
      })),
      ...(mission?.geometry.exclusion_areas ?? []).map(
        (coordinates, index) => ({
          type: "Feature" as const,
          properties: { kind: "exclude", index },
          geometry: { type: "Polygon" as const, coordinates: [coordinates] },
        }),
      ),
      ...[...routes.entries()].flatMap(([key, entries]) =>
        entries.length < 2
          ? []
          : [{
              type: "Feature" as const,
              id: `route-${key}`,
              properties: { kind: "waypoint-route", color: entries[0].color },
              geometry: {
                type: "LineString" as const,
                coordinates: entries
                  .sort((a, b) => a.sequence - b.sequence)
                  .map((entry) => entry.position),
              },
            }],
      ),
      ...waypoints.flatMap((coordinates, index) => consumedWaypointIndexes.has(index) ? [] : [{
        type: "Feature" as const,
        id: details[index]?.id ?? `legacy-waypoint-${index + 1}`,
        properties: {
          kind: "waypoint",
          index,
          sequence: index + 1,
          color: details[index]?.color ?? "amber",
          ownerGroup: details[index]?.owner_group_id ?? "",
        },
        geometry: { type: "Point" as const, coordinates },
      }]),
      ...(mission?.geometry.pois ?? []).map((poi, index) => ({
        type: "Feature" as const,
        id: poi.id,
        properties: { kind: "poi", poiKind: poi.kind, name: poi.name, index },
        geometry: { type: "Point" as const, coordinates: poi.position },
      })),
    ],
  };
}
function groupAssemblyData(fleet: FleetSnapshotV2): GeoJSON.FeatureCollection {
  const missionTargets = new Set(
    fleet.missions
      .filter((mission) => ["authorized", "executing", "paused"].includes(mission.status))
      .flatMap((mission) => mission.target_ids),
  );
  return {
    type: "FeatureCollection",
    features: fleet.groups
      .filter((group) => group.assembly_point && !group.member_ids.some((id) => missionTargets.has(id)))
      .map((group) => ({
        type: "Feature",
        id: `assembly-${group.id}`,
        properties: {
          group: group.id,
          code: group.code,
          color: group.color,
          formation: group.formation,
          spacing: group.formation_spacing_m,
        },
        geometry: { type: "Point", coordinates: group.assembly_point! },
      })),
  };
}
function flowData(
  fleet: FleetSnapshotV2,
  anchors: Point[],
  phase: number,
): GeoJSON.FeatureCollection {
  const env = fleet.environment;
  return {
    type: "FeatureCollection",
    features: anchors.flatMap((anchor, index) => {
      const spatial = Math.sin(anchor[0] * 9.7 + anchor[1] * 7.1);
      const temporal = Math.sin(phase * Math.PI * 2 + index * 0.31);
      const currentBearing =
        (env.current_direction_deg + spatial * 14 + temporal * 3 + 360) % 360;
      const windBearing =
        (env.wind_direction_deg + 180 + spatial * 9 + temporal * 2 + 360) % 360;
      const make = (
        kind: "current" | "wind",
        bearing: number,
        speed: number,
        offset: number,
      ) => {
        const radians = (bearing * Math.PI) / 180;
        const travel =
          (((phase + index * 0.071 + offset) % 1) - 0.5) *
          (kind === "wind" ? 0.018 : 0.01);
        return {
          type: "Feature" as const,
          properties: {
            kind,
            bearing,
            speed,
            label: `${kind} ${speed.toFixed(kind === "wind" ? 1 : 2)} m/s`,
          },
          geometry: {
            type: "Point" as const,
            coordinates: [
              anchor[0] + Math.sin(radians) * travel,
              anchor[1] + Math.cos(radians) * travel,
            ],
          },
        };
      };
      return [
        make(
          "current",
          currentBearing,
          Math.max(0.02, env.current_speed_mps * (1 + spatial * 0.16)),
          0,
        ),
        make(
          "wind",
          windBearing,
          Math.max(0.1, env.wind_speed_mps * (1 + spatial * 0.12)),
          0.43,
        ),
      ];
    }),
  };
}

export function OperationsMap({
  fleet,
  mission,
  selected,
  activePlan,
  activeMissionPlans,
  tool,
  editingEnabled,
  focusedGeometry,
  pirate,
  onSelect,
  onGroup,
  onVessel,
  onOpenFleet,
  onContact,
  onPlanContact,
  onGeometryFocus,
  onWaypoint,
  onPOI,
  onMoveWaypoint,
  onMovePOI,
  onMoveGroupAssembly,
  onHoldGroupAtVessel,
  onArea,
  onToolDone,
  sceneAnnotations,
  sceneCamera,
}: Props) {
  const host = useRef<HTMLDivElement>(null),
    mapRef = useRef<MLMap | null>(null),
    boxStart = useRef<{ x: number; y: number } | null>(null),
    selectionMode = useRef(false),
    fleetRef = useRef(fleet),
    flowAnchors = useRef<Point[]>([]),
    multiClick = useRef({ count: 0, at: 0 }),
    dragTarget = useRef<
      | { kind: "waypoint"; index: number }
      | { kind: "poi"; index: number }
      | { kind: "assembly"; group: string }
      | null
    >(null),
    suppressClick = useRef(false);
  const [ready, setReady] = useState(false),
    [box, setBox] = useState<{
      x: number;
      y: number;
      w: number;
      h: number;
    } | null>(null),
    [overlays, setOverlays] = useState({
      current: true,
      wind: true,
      depth: true,
    }),
    [contextMenu, setContextMenu] = useState<ContextMenu | null>(null),
    [mapHover, setMapHover] = useState<{
      x: number;
      y: number;
      title: string;
      detail: string;
      accent?: string;
    } | null>(null),
    [dragMarker, setDragMarker] = useState<{ x: number; y: number; color: string } | null>(null);
  const longPress = useRef<{
    timer: number | null;
    pointer: number;
    x: number;
    y: number;
  } | null>(null);
  fleetRef.current = fleet;
  const selectedGroup = useMemo(() => {
    if (selected.size === 0) return undefined;
    return fleet.groups.find((group) =>
      group.member_ids.length === selected.size &&
      [...selected].every((id) => group.member_ids.includes(id)),
    );
  }, [fleet.groups, selected]);
  const effectiveWaypointColor = selectedGroup
    ? (selectedGroup.color_name as WaypointColor)
    : "amber";
  const contactMissionOverlay = useMemo(
    () => contactMissionOverlayData(activeMissionPlans, fleet),
    [activeMissionPlans, fleet],
  );
  const remainingMissionRoutes = useMemo(
    () => routeData(activePlan, mission, fleet),
    [activePlan, mission, fleet],
  );
  const visibleMissionGeometry = useMemo(
    () => geometryData(mission, activePlan, fleet),
    [mission, activePlan, fleet],
  );
  const rendezvousStatus = String(
    contactMissionOverlay.features.find((feature) => feature.properties?.kind === "rendezvous-eta")?.properties?.label ?? "",
  );
  const visibleHoldGroups = useMemo(() => groupAssemblyData(fleet), [fleet]);
  const visibleVesselIDs = () => {
    const map = mapRef.current;
    if (!map) return [];
    const bounds = map.getBounds();
    return fleetRef.current.vessels
      .filter((v) => bounds.contains(v.telemetry.position))
      .map((v) => v.id);
  };
  useEffect(
    () => () => {
      const active = longPress.current;
      if (active && active.timer !== null) window.clearTimeout(active.timer);
      longPress.current = null;
    },
    [],
  );
  useEffect(() => {
    if (!host.current || mapRef.current) return;
    let disposed = false;
    const map = new maplibregl.Map({
      container: host.current,
      style,
      center: [-71.34, 41.35],
      zoom: 9.1,
      pitch: 24,
      bearing: -8,
      maxPitch: 60,
      minZoom: 7,
      maxZoom: 16,
      attributionControl: false,
      maxBounds: [
        [-72.1, 40.75],
        [-70.55, 42.05],
      ],
    });
    map.addControl(
      new maplibregl.NavigationControl({
        showCompass: true,
        showZoom: true,
        visualizePitch: true,
      }),
      "bottom-right",
    );
    map.addControl(
      new maplibregl.ScaleControl({ unit: "nautical" }),
      "bottom-left",
    );
    mapRef.current = map;
    map.on("load", async () => {
      for (const name of ["kestrel", "mariner", "atlas"]) {
        const image = await map.loadImage(`/assets/vessels/${name}-2p5d.png`);
        if (disposed || mapRef.current !== map) return;
        map.addImage(`vessel-${name}`, image.data, { pixelRatio: 8 });
        const pirateImage = await map.loadImage(
          `/assets/vessels/pirate-${name}.png`,
        );
        if (disposed || mapRef.current !== map) return;
        map.addImage(`pirate-vessel-${name}`, pirateImage.data, {
          pixelRatio: 8,
        });
      }
      for (const name of [
        "container",
        "tanker",
        "ferry",
        "trawler",
        "patrol",
        "yacht",
      ]) {
        const image = await map.loadImage(`/assets/traffic/${name}.png`);
        if (disposed || mapRef.current !== map) return;
        map.addImage(`traffic-${name}`, image.data, { pixelRatio: 2 });
      }
      if (disposed || mapRef.current !== map || !map.getSource("coast")) return;
      map.addLayer({
        id: "depth-contours",
        type: "line",
        source: "coast",
        filter: ["==", ["get", "kind"], "depth"],
        paint: {
          "line-color": [
            "interpolate",
            ["linear"],
            ["get", "depth_m"],
            5,
            "#62a7b4",
            20,
            "#397b8c",
            80,
            "#23546a",
          ],
          "line-width": [
            "interpolate",
            ["linear"],
            ["get", "depth_m"],
            5,
            1.15,
            80,
            0.65,
          ],
          "line-opacity": 0.72,
        },
      });
      map.addLayer({
        id: "depth-labels",
        type: "symbol",
        source: "coast",
        filter: ["==", ["get", "kind"], "depth_label"],
        layout: {
          "text-field": ["concat", ["to-string", ["get", "depth_m"]], " m"],
          "text-size": 8,
          "text-allow-overlap": false,
        },
        paint: {
          "text-color": "#75b9c6",
          "text-halo-color": "#102d35",
          "text-halo-width": 1.5,
        },
      });
      map.addSource("environment-flow", { type: "geojson", data: empty });
      map.addLayer({
        id: "current-vectors",
        type: "symbol",
        source: "environment-flow",
        filter: ["==", ["get", "kind"], "current"],
        layout: {
          "text-field": "➤",
          "text-size": 11,
          "text-rotate": ["get", "bearing"],
          "text-rotation-alignment": "map",
          "text-allow-overlap": true,
        },
        paint: {
          "text-color": "#53c1d1",
          "text-opacity": 0.82,
          "text-halo-color": "#0c2730",
          "text-halo-width": 1,
        },
      });
      map.addLayer({
        id: "wind-vectors",
        type: "symbol",
        source: "environment-flow",
        filter: ["==", ["get", "kind"], "wind"],
        layout: {
          "text-field": "➤",
          "text-size": 13,
          "text-rotate": ["get", "bearing"],
          "text-rotation-alignment": "map",
          "text-allow-overlap": true,
        },
        paint: {
          "text-color": "#e7d79a",
          "text-opacity": 0.7,
          "text-halo-color": "#1d2926",
          "text-halo-width": 1,
        },
      });
      map.addSource("mission-geometry", { type: "geojson", data: empty });
      map.addSource("command-scene", { type: "geojson", data: empty });
      map.addLayer({ id: "command-scene-lines", type: "line", source: "command-scene", filter: ["==", ["geometry-type"], "LineString"], paint: { "line-color": ["get", "color"], "line-width": 3, "line-dasharray": [2, 2], "line-opacity": 0.9 } });
      map.addLayer({ id: "command-scene-points", type: "circle", source: "command-scene", filter: ["==", ["geometry-type"], "Point"], paint: { "circle-radius": 11, "circle-color": "#101718", "circle-stroke-color": ["get", "color"], "circle-stroke-width": 3 } });
      map.addLayer({ id: "command-scene-labels", type: "symbol", source: "command-scene", filter: ["==", ["geometry-type"], "Point"], layout: { "text-field": ["get", "label"], "text-size": 11, "text-offset": [0, 1.8], "text-anchor": "top", "text-allow-overlap": true }, paint: { "text-color": "#f4ead6", "text-halo-color": "#101718", "text-halo-width": 2 } });
      map.addLayer({
        id: "mission-areas",
        type: "fill",
        source: "mission-geometry",
        filter: ["in", ["get", "kind"], ["literal", ["include", "exclude"]]],
        paint: {
          "fill-color": [
            "match",
            ["get", "kind"],
            "exclude",
            "#d06054",
            "#d6a242",
          ],
          "fill-opacity": 0.13,
        },
      });
      map.addLayer({
        id: "mission-area-lines",
        type: "line",
        source: "mission-geometry",
        filter: ["in", ["get", "kind"], ["literal", ["include", "exclude"]]],
        paint: {
          "line-color": [
            "match",
            ["get", "kind"],
            "exclude",
            "#e26e62",
            "#e6a63b",
          ],
          "line-width": 2,
          "line-dasharray": [3, 2],
        },
      });
      map.addLayer({
        id: "mission-waypoint-routes",
        type: "line",
        source: "mission-geometry",
        filter: ["==", ["get", "kind"], "waypoint-route"],
        paint: {
          "line-color": [
            "match", ["get", "color"],
            "amber", "#e9a93f", "teal", "#62c5a8", "coral", "#d86f5f",
            "violet", "#b895d8", "blue", "#7eb4df", "yellow", "#d2c05d",
            "pink", "#df8fb0", "lime", "#8fca72", "#e9a93f",
          ],
          "line-width": 2,
          "line-dasharray": [2, 2],
          "line-opacity": 0.8,
        },
      });
      map.addLayer({
        id: "mission-waypoints",
        type: "circle",
        source: "mission-geometry",
        filter: ["==", ["get", "kind"], "waypoint"],
        paint: {
          "circle-color": [
            "match",
            ["get", "color"],
            "amber",
            "#e9a93f",
            "teal",
            "#62c5a8",
            "coral",
            "#d86f5f",
            "violet",
            "#b895d8",
            "blue",
            "#7eb4df",
            "yellow",
            "#d2c05d",
            "pink",
            "#df8fb0",
            "lime",
            "#8fca72",
            "red",
            "#d86f5f",
            "green",
            "#8fca72",
            "cyan",
            "#62c5a8",
            "white",
            "#ece8dc",
            "#e9a93f",
          ],
          "circle-radius": 9,
          "circle-stroke-color": "#111718",
          "circle-stroke-width": 2.5,
        },
      });
      map.addLayer({
        id: "mission-waypoint-numbers",
        type: "symbol",
        source: "mission-geometry",
        filter: ["==", ["get", "kind"], "waypoint"],
        layout: {
          "text-field": ["to-string", ["get", "sequence"]],
          "text-size": 9,
          "text-font": ["Open Sans Bold", "Arial Unicode MS Bold"],
          "text-allow-overlap": true,
        },
        paint: {
          "text-color": "#111718",
          "text-halo-color": "#f4eee0",
          "text-halo-width": 0.35,
        },
      });
      map.addLayer({
        id: "mission-pois",
        type: "circle",
        source: "mission-geometry",
        filter: ["==", ["get", "kind"], "poi"],
        paint: {
          "circle-radius": 11,
          "circle-color": ["match", ["get", "poiKind"], "orbit", "#62c5a8", "#e9a93f"],
          "circle-opacity": 0.25,
          "circle-stroke-color": ["match", ["get", "poiKind"], "orbit", "#8ce0c5", "#ffd071"],
          "circle-stroke-width": 2,
        },
      });
      map.addLayer({
        id: "mission-poi-labels",
        type: "symbol",
        source: "mission-geometry",
        filter: ["==", ["get", "kind"], "poi"],
        layout: { "text-field": ["upcase", ["get", "poiKind"]], "text-size": 8, "text-offset": [0, 2.2] },
        paint: { "text-color": "#ece8dc", "text-halo-color": "#111718", "text-halo-width": 2 },
      });
      map.addSource("group-assembly", {
        type: "geojson",
        data: groupAssemblyData(fleet),
      });
      map.addLayer({
        id: "group-assembly-rings",
        type: "circle",
        source: "group-assembly",
        paint: {
          "circle-radius": 14,
          "circle-color": ["get", "color"],
          "circle-opacity": 0.13,
          "circle-stroke-color": ["get", "color"],
          "circle-stroke-width": 2,
          "circle-stroke-opacity": 0.9,
        },
      });
      map.addLayer({
        id: "group-assembly-labels",
        type: "symbol",
        source: "group-assembly",
        layout: {
          "text-field": ["concat", ["get", "code"], " · HOLD"],
          "text-size": 8,
          "text-offset": [0, 2.2],
          "text-allow-overlap": false,
        },
        paint: {
          "text-color": "#ece8dc",
          "text-halo-color": "#111718",
          "text-halo-width": 2,
        },
      });
      map.addSource("routes", { type: "geojson", data: empty });
      map.addLayer({
        id: "mission-routes",
        type: "line",
        source: "routes",
        paint: {
          "line-color": "#e6a63b",
          "line-width": 2.5,
          "line-dasharray": [2, 1.6],
          "line-opacity": 0.9,
        },
      });
      map.addSource("contact-mission-overlay", { type: "geojson", data: empty });
      map.addLayer({
        id: "contact-mission-prediction",
        type: "line",
        source: "contact-mission-overlay",
        filter: ["==", ["get", "kind"], "contact-prediction"],
        paint: {
          "line-color": ["get", "color"],
          "line-width": 1.5,
          "line-dasharray": [2, 3],
          "line-opacity": 0.7,
        },
      });
      map.addLayer({
        id: "contact-mission-route-glow",
        type: "line",
        source: "contact-mission-overlay",
        filter: ["==", ["get", "kind"], "rendezvous-route"],
        paint: {
          "line-color": ["get", "color"],
          "line-width": 8,
          "line-opacity": 0.18,
          "line-blur": 2,
        },
      });
      map.addLayer({
        id: "contact-mission-route",
        type: "line",
        source: "contact-mission-overlay",
        filter: ["==", ["get", "kind"], "rendezvous-route"],
        paint: {
          "line-color": ["get", "color"],
          "line-width": 3,
          "line-opacity": 0.96,
        },
      });
      map.addLayer({
        id: "contact-mission-eta",
        type: "symbol",
        source: "contact-mission-overlay",
        filter: ["==", ["get", "kind"], "rendezvous-eta"],
        layout: {
          "text-field": ["get", "label"],
          "text-size": 10,
          "text-offset": [0, -1.1],
          "text-allow-overlap": true,
        },
        paint: {
          "text-color": "#fff3d5",
          "text-halo-color": "#111718",
          "text-halo-width": 2.5,
        },
      });
      map.addSource("surface-contact-routes", {
        type: "geojson",
        data: surfaceRouteData(fleet.surface_contacts),
      });
      map.addLayer({
        id: "surface-contact-routes",
        type: "line",
        source: "surface-contact-routes",
        paint: {
          "line-color": ["get", "color"],
          "line-width": 1,
          "line-dasharray": [2, 5],
          "line-opacity": 0.22,
        },
      });
      map.addSource("surface-contacts", {
        type: "geojson",
        data: surfaceContactData(fleet),
      });
      map.addLayer({
        id: "surface-contact-halos",
        type: "circle",
        source: "surface-contacts",
        paint: {
          "circle-radius": 13,
          "circle-color": ["get", "color"],
          "circle-opacity": 0.12,
          "circle-stroke-color": ["get", "color"],
          "circle-stroke-width": 1.5,
        },
      });
      map.addLayer({
        id: "surface-contact-symbols",
        type: "symbol",
        source: "surface-contacts",
        layout: {
          "icon-image": ["concat", "traffic-", ["get", "class"]],
          "icon-size": [
            "interpolate",
            ["linear"],
            ["zoom"],
            7,
            ["*", 0.07, ["get", "scale"]],
            11,
            ["*", 0.14, ["get", "scale"]],
            15,
            ["*", 0.25, ["get", "scale"]],
          ],
          "icon-rotate": ["get", "heading"],
          "icon-rotation-alignment": "map",
          "icon-pitch-alignment": "map",
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
          "text-field": ["step", ["zoom"], "", 10, ["get", "name"]],
          "text-size": 9,
          "text-offset": [0, 2.6],
          "text-optional": true,
        },
        paint: {
          "text-color": ["get", "color"],
          "text-halo-color": "#111718",
          "text-halo-width": 2,
        },
      });
      map.addSource("vessels", {
        type: "geojson",
        data: vesselData(fleet),
        cluster: true,
        clusterRadius: 46,
        clusterMaxZoom: 8,
      });
      map.addLayer({
        id: "clusters",
        type: "circle",
        source: "vessels",
        filter: ["has", "point_count"],
        paint: {
          "circle-color": "#d29a3e",
          "circle-radius": ["step", ["get", "point_count"], 17, 20, 23, 40, 30],
          "circle-stroke-color": "#221d14",
          "circle-stroke-width": 3,
        },
      });
      map.addLayer({
        id: "cluster-count",
        type: "symbol",
        source: "vessels",
        filter: ["has", "point_count"],
        layout: {
          "text-field": ["get", "point_count_abbreviated"],
          "text-size": 11,
        },
        paint: { "text-color": "#17130d" },
      });
      map.addLayer({
        id: "group-halos",
        type: "circle",
        source: "vessels",
        filter: ["!", ["has", "point_count"]],
        paint: {
          "circle-radius": 6,
          "circle-color": ["get", "color"],
          "circle-opacity": 0.2,
          "circle-stroke-color": ["get", "color"],
          "circle-stroke-width": 1.25,
        },
      });
      map.addLayer({
        id: "selection-rings",
        type: "circle",
        source: "vessels",
        filter: ["!", ["has", "point_count"]],
        paint: {
          "circle-radius": [
            "case",
            ["boolean", ["feature-state", "selected"], false],
            21,
            0,
          ],
          "circle-color": "#efaa3d",
          "circle-opacity": [
            "case",
            ["boolean", ["feature-state", "selected"], false],
            0.2,
            0,
          ],
          "circle-stroke-color": "#ffd071",
          "circle-stroke-width": [
            "case",
            ["boolean", ["feature-state", "selected"], false],
            3.5,
            0,
          ],
          "circle-blur": 0.08,
        },
      });
      map.addLayer({
        id: "vessel-symbols",
        type: "symbol",
        source: "vessels",
        filter: ["!", ["has", "point_count"]],
        layout: {
          "icon-image": ["concat", "vessel-", ["get", "class"]],
          "icon-size": [
            "interpolate",
            ["linear"],
            ["zoom"],
            8,
            0.1,
            11,
            0.22,
            15,
            0.38,
          ],
          "icon-rotate": ["get", "heading"],
          "icon-rotation-alignment": "map",
          "icon-pitch-alignment": "map",
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
          "text-field": [
            "step",
            ["zoom"],
            "",
            10,
            ["concat", ["get", "name"], " · ", ["get", "groupCode"]],
          ],
          "text-size": 10,
          "text-offset": [0, 2.2],
          "text-allow-overlap": false,
          "text-optional": true,
        },
        paint: {
          "text-color": "#ece8dc",
          "text-halo-color": "#111718",
          "text-halo-width": 2,
        },
      });
      map.addLayer({
        id: "reserve-badges",
        type: "symbol",
        source: "vessels",
        filter: ["!", ["has", "point_count"]],
        minzoom: 12,
        layout: {
          "text-field": ["concat", ["to-string", ["get", "reserve"]], "%"],
          "text-size": 8,
          "text-offset": [0, -2.2],
        },
        paint: {
          "text-color": "#efe7d4",
          "text-halo-color": "#161d1e",
          "text-halo-width": 2,
        },
      });
      setReady(true);
    });
    return () => {
      disposed = true;
      map.remove();
      mapRef.current = null;
    };
  }, []);
  useEffect(() => {
    if (!ready || !mapRef.current) return;
    (mapRef.current.getSource("vessels") as GeoJSONSource)?.setData(
      vesselData(fleet),
    );
    (mapRef.current.getSource("routes") as GeoJSONSource)?.setData(
      remainingMissionRoutes,
    );
    (mapRef.current.getSource("contact-mission-overlay") as GeoJSONSource)?.setData(
      contactMissionOverlay,
    );
    (mapRef.current.getSource("surface-contacts") as GeoJSONSource)?.setData(
      surfaceContactData(fleet),
    );
    (
      mapRef.current.getSource("surface-contact-routes") as GeoJSONSource
    )?.setData(surfaceRouteData(fleet.surface_contacts));
    (mapRef.current.getSource("mission-geometry") as GeoJSONSource)?.setData(
      visibleMissionGeometry,
    );
    (mapRef.current.getSource("group-assembly") as GeoJSONSource)?.setData(
      visibleHoldGroups,
    );
    (mapRef.current.getSource("command-scene") as GeoJSONSource)?.setData(sceneAnnotationData(sceneAnnotations));
  }, [ready, fleet, contactMissionOverlay, remainingMissionRoutes, visibleMissionGeometry, visibleHoldGroups, sceneAnnotations]);
  useEffect(() => {
    if (!ready || !mapRef.current || !sceneCamera) return;
    mapRef.current.easeTo({ center: sceneCamera.center, zoom: sceneCamera.zoom, bearing: sceneCamera.bearing, pitch: sceneCamera.pitch, duration: 550 });
  }, [ready, sceneCamera?.center[0], sceneCamera?.center[1], sceneCamera?.zoom, sceneCamera?.bearing, sceneCamera?.pitch]);
  useEffect(() => {
    if (!ready || !mapRef.current || !mission || !focusedGeometry) return;
    let point: Point | undefined;
    if (focusedGeometry.kind === "waypoint") point = mission.geometry.waypoints[focusedGeometry.index];
    if (focusedGeometry.kind === "poi") point = mission.geometry.pois[focusedGeometry.index]?.position;
    if (focusedGeometry.kind === "include" || focusedGeometry.kind === "exclude") {
      const polygon = (focusedGeometry.kind === "include" ? mission.geometry.included_areas : mission.geometry.exclusion_areas)[focusedGeometry.index] ?? [];
      if (polygon.length) point = [polygon.reduce((sum, p) => sum + p[0], 0) / polygon.length, polygon.reduce((sum, p) => sum + p[1], 0) / polygon.length];
    }
    if (point) mapRef.current.easeTo({ center: point, duration: 350 });
  }, [ready, mission?.id, focusedGeometry?.kind, focusedGeometry?.index]);
  useEffect(() => {
    if (!ready || !mapRef.current) return;
    let cancelled = false,
      timer = 0;
    void fetch("/assets/maps/narragansett.geojson")
      .then((r) => r.json())
      .then((data: GeoJSON.FeatureCollection) => {
        if (cancelled) return;
        flowAnchors.current = data.features
          .filter(
            (f) =>
              f.properties?.kind === "flow_anchor" &&
              f.geometry.type === "Point",
          )
          .map((f) => (f.geometry as GeoJSON.Point).coordinates as Point);
        const animate = () => {
          const source = mapRef.current?.getSource("environment-flow") as
            GeoJSONSource | undefined;
          source?.setData(
            flowData(
              fleetRef.current,
              flowAnchors.current,
              (performance.now() % 7000) / 7000,
            ),
          );
        };
        animate();
        timer = window.setInterval(animate, 500);
      });
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [ready]);
  useEffect(() => {
    const map = mapRef.current;
    if (!ready || !map) return;
    map.setLayoutProperty(
      "current-vectors",
      "visibility",
      overlays.current ? "visible" : "none",
    );
    map.setLayoutProperty(
      "wind-vectors",
      "visibility",
      overlays.wind ? "visible" : "none",
    );
    map.setLayoutProperty(
      "depth-contours",
      "visibility",
      overlays.depth ? "visible" : "none",
    );
    map.setLayoutProperty(
      "depth-labels",
      "visibility",
      overlays.depth ? "visible" : "none",
    );
  }, [ready, overlays]);
  useEffect(() => {
    const map = mapRef.current;
    if (!ready || !map) return;
    map.setPaintProperty(
      "sea",
      "background-color",
      pirate ? "#071b1b" : "#0b2731",
    );
    map.setPaintProperty(
      "bathymetry",
      "background-color",
      pirate ? "#102a27" : "#12333d",
    );
    map.setPaintProperty("land", "fill-color", pirate ? "#332b1e" : "#30362f");
    map.setPaintProperty(
      "land",
      "fill-outline-color",
      pirate ? "#8d7449" : "#718076",
    );
    map.setPaintProperty(
      "shipping",
      "line-color",
      pirate ? "#d1aa58" : "#9f8b55",
    );
    map.setLayoutProperty("vessel-symbols", "icon-image", [
      "concat",
      pirate ? "pirate-vessel-" : "vessel-",
      ["get", "class"],
    ]);
  }, [ready, pirate]);
  useEffect(() => {
    if (!ready || !mapRef.current) return;
    for (const v of fleet.vessels)
      mapRef.current.setFeatureState(
        { source: "vessels", id: v.id },
        { selected: selected.has(v.id) },
      );
  }, [ready, selected, fleet.vessels]);
  useEffect(() => {
    const map = mapRef.current;
    if (!ready || !map) return;
    const canvas = map.getCanvas();
    canvas.style.cursor = editingEnabled && tool !== "select" ? "crosshair" : "default";
    const click = (e: MapMouseEvent) => {
      setContextMenu(null);
      if (suppressClick.current) {
        suppressClick.current = false;
        return;
      }
      if (editingEnabled && tool === "waypoint") {
        if (
          map.queryRenderedFeatures(e.point, { layers: ["land"] }).length === 0
        )
          onWaypoint([e.lngLat.lng, e.lngLat.lat], effectiveWaypointColor);
        onToolDone();
        return;
      }
      if (editingEnabled && (tool === "hold" || tool === "orbit")) {
        if (map.queryRenderedFeatures(e.point, { layers: ["land"] }).length === 0)
          onPOI(tool, [e.lngLat.lng, e.lngLat.lat]);
        onToolDone();
        return;
      }
      const now = performance.now();
      multiClick.current = {
        count:
          now - multiClick.current.at < 500 ? multiClick.current.count + 1 : 1,
        at: now,
      };
      const detail = Math.max(
        multiClick.current.count,
        (e.originalEvent as MouseEvent).detail || 1,
      );
      if (detail >= 4) {
        e.preventDefault();
        onSelect(
          fleetRef.current.vessels.map((v) => v.id),
          "replace",
        );
        return;
      }
      if (detail === 3) {
        e.preventDefault();
        onSelect(visibleVesselIDs(), "replace");
        return;
      }
      if (detail === 2) return;
      const contactBox: [[number, number], [number, number]] = [
        [e.point.x - 18, e.point.y - 18],
        [e.point.x + 18, e.point.y + 18],
      ];
      const contact = map.queryRenderedFeatures(contactBox, {
        layers: ["surface-contact-symbols", "surface-contact-halos"],
      })[0];
      if (contact?.properties?.id) {
        onContact(String(contact.properties.id));
        return;
      }
      const hit = map.queryRenderedFeatures(e.point, {
        layers: ["vessel-symbols", "group-halos"],
      })[0];
      if (hit?.properties?.id)
        onSelect(
          [hit.properties.id],
          e.originalEvent.shiftKey ? "toggle" : "replace",
        );
    };
    const dbl = (e: MapMouseEvent) => {
      e.preventDefault();
      if (
        Math.max(
          multiClick.current.count,
          (e.originalEvent as MouseEvent).detail || 2,
        ) >= 4
      )
        return;
      const hit = map.queryRenderedFeatures(e.point, {
        layers: ["vessel-symbols", "group-halos"],
      })[0];
      if (hit?.properties?.group) onGroup(hit.properties.group);
    };
    const openContext = (
      point: { x: number; y: number },
      lngLat: { lng: number; lat: number },
    ) => {
      multiClick.current = { count: 0, at: 0 };
      const projectedWaypoint = (mission?.geometry.waypoints ?? []).findIndex(
        (position) => {
          const p = map.project(position);
          return Math.hypot(p.x - point.x, p.y - point.y) <= 14;
        },
      );
      const waypointBox: [[number, number], [number, number]] = [
        [point.x - 12, point.y - 12],
        [point.x + 12, point.y + 12],
      ];
      const waypoint = map.queryRenderedFeatures(waypointBox, {
        layers: ["mission-waypoint-numbers", "mission-waypoints"],
      })[0];
      const waypointIndex =
        projectedWaypoint >= 0
          ? projectedWaypoint
          : waypoint?.properties?.index !== undefined
            ? Number(waypoint.properties.index)
            : -1;
      if (waypointIndex >= 0) {
        onGeometryFocus("waypoint", waypointIndex);
        setContextMenu(null);
        return;
      }
      const geometryFeature = map.queryRenderedFeatures(waypointBox, {
        layers: ["mission-poi-labels", "mission-pois", "mission-area-lines", "mission-areas"],
      })[0];
      if (geometryFeature?.properties?.kind && geometryFeature.properties.index !== undefined) {
        onGeometryFocus(
          geometryFeature.properties.kind === "poi" ? "poi" : geometryFeature.properties.kind,
          Number(geometryFeature.properties.index),
        );
        setContextMenu(null);
        return;
      }
      const vesselBox: [[number, number], [number, number]] = [
        [point.x - 20, point.y - 20],
        [point.x + 20, point.y + 20],
      ];
      const vessel = map.queryRenderedFeatures(vesselBox, {
        layers: ["vessel-symbols", "group-halos"],
      })[0];
      const maxX = Math.max(4, (host.current?.clientWidth ?? point.x) - 224),
        maxY = Math.max(4, (host.current?.clientHeight ?? point.y) - 360),
        x = Math.min(point.x, maxX),
        y = Math.min(point.y, maxY);
      if (vessel?.properties?.id) {
        setContextMenu({
          kind: "vessel",
          x,
          y,
          vessel: String(vessel.properties.id),
          group: String(vessel.properties.group),
        });
        return;
      }
      const contact = map.queryRenderedFeatures(vesselBox, {
        layers: ["surface-contact-symbols", "surface-contact-halos"],
      })[0];
      if (contact?.properties?.id) {
        setContextMenu({
          kind: "contact",
          x,
          y,
          contact: String(contact.properties.id),
        });
        return;
      }
      const surface = map.queryRenderedFeatures([point.x, point.y], { layers: ["land"] }).length > 0 ? "land" : "water";
      const nearbyDepth = map.queryRenderedFeatures(
        [[point.x - 30, point.y - 30], [point.x + 30, point.y + 30]],
        { layers: ["depth-contours", "depth-labels"] },
      ).map(feature => Number(feature.properties?.depth_m)).filter(Number.isFinite).sort((a,b)=>a-b)[0];
      setContextMenu({
        kind: "location",
        x,
        y,
        point: [lngLat.lng, lngLat.lat],
        surface,
        depth: nearbyDepth,
      });
    };
    const context = (e: MapMouseEvent) => {
      e.preventDefault();
      openContext(e.point, e.lngLat);
    };
    const cancelLongPress = () => {
      const active = longPress.current;
      if (active && active.timer !== null) window.clearTimeout(active.timer);
      longPress.current = null;
    };
    const pointerDown = (event: PointerEvent) => {
      if (event.pointerType !== "touch" && event.pointerType !== "pen") return;
      cancelLongPress();
      const rect = canvas.getBoundingClientRect();
      const point = { x: event.clientX - rect.left, y: event.clientY - rect.top };
      if (editingEnabled && tool === "select") {
        const hitBox: [[number, number], [number, number]] = [
          [point.x - 18, point.y - 18],
          [point.x + 18, point.y + 18],
        ];
        const waypoint = map.queryRenderedFeatures(hitBox, { layers: ["mission-waypoint-numbers", "mission-waypoints"] })[0];
        const poi = map.queryRenderedFeatures(hitBox, { layers: ["mission-poi-labels", "mission-pois"] })[0];
        const assembly = map.queryRenderedFeatures(hitBox, { layers: ["group-assembly-labels", "group-assembly-rings"] })[0];
        if (waypoint?.properties?.index !== undefined)
          dragTarget.current = { kind: "waypoint", index: Number(waypoint.properties.index) };
        else if (poi?.properties?.index !== undefined)
          dragTarget.current = { kind: "poi", index: Number(poi.properties.index) };
        else if (assembly?.properties?.group)
          dragTarget.current = { kind: "assembly", group: String(assembly.properties.group) };
        if (dragTarget.current) {
          event.preventDefault();
          setDragMarker({ x: point.x, y: point.y, color: String(waypoint?.properties?.color ?? assembly?.properties?.color ?? "#e9a93f") });
          map.dragPan.disable();
          map.touchZoomRotate.disable();
          canvas.style.cursor = "grabbing";
          return;
        }
      }
      if (editingEnabled && ["box", "include", "exclude"].includes(tool)) {
        event.preventDefault();
        selectionMode.current = true;
        boxStart.current = point;
        map.dragPan.disable();
        map.touchZoomRotate.disable();
        setBox({ x: point.x, y: point.y, w: 0, h: 0 });
        return;
      }
      const state = {
        timer: 0,
        pointer: event.pointerId,
        x: event.clientX,
        y: event.clientY,
      };
      state.timer = window.setTimeout(() => {
        const point = { x: state.x - rect.left, y: state.y - rect.top };
        suppressClick.current = true;
        map.stop();
        setMapHover(null);
        openContext(point, map.unproject([point.x, point.y]));
        navigator.vibrate?.(12);
        longPress.current = null;
      }, 560);
      longPress.current = state;
    };
    const pointerMove = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect();
      const point = { x: event.clientX - rect.left, y: event.clientY - rect.top };
      if (dragTarget.current) {
        event.preventDefault();
        setDragMarker((current) => ({ x: point.x, y: point.y, color: current?.color ?? "#e9a93f" }));
        return;
      }
      if (selectionMode.current && boxStart.current) {
        event.preventDefault();
        const start = boxStart.current;
        setBox({ x: Math.min(start.x, point.x), y: Math.min(start.y, point.y), w: Math.abs(point.x - start.x), h: Math.abs(point.y - start.y) });
        return;
      }
      const state = longPress.current;
      if (!state || event.pointerId !== state.pointer) return;
      if (Math.hypot(event.clientX - state.x, event.clientY - state.y) > 12)
        cancelLongPress();
    };
    const pointerUp = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect();
      const point = { x: event.clientX - rect.left, y: event.clientY - rect.top };
      if (dragTarget.current) {
        const target = dragTarget.current;
        const lngLat = map.unproject([point.x, point.y]);
        const position: Point = [lngLat.lng, lngLat.lat];
        if (map.queryRenderedFeatures([point.x, point.y], { layers: ["land"] }).length === 0) {
          if (target.kind === "waypoint") onMoveWaypoint(target.index, position);
          else if (target.kind === "poi") onMovePOI(target.index, position);
          else onMoveGroupAssembly(target.group, position);
        }
        dragTarget.current = null;
        suppressClick.current = true;
        setDragMarker(null);
        map.dragPan.enable();
        map.touchZoomRotate.enable();
        canvas.style.cursor = "default";
        cancelLongPress();
        return;
      }
      if (selectionMode.current && boxStart.current) {
        const start = boxStart.current;
        const min = { x: Math.min(start.x, point.x), y: Math.min(start.y, point.y) };
        const max = { x: Math.max(start.x, point.x), y: Math.max(start.y, point.y) };
        if (tool === "include" || tool === "exclude") {
          const nw = map.unproject([min.x, min.y]), se = map.unproject([max.x, max.y]);
          onArea(tool, [[nw.lng, nw.lat], [se.lng, nw.lat], [se.lng, se.lat], [nw.lng, se.lat], [nw.lng, nw.lat]]);
        } else {
          const hits = map.queryRenderedFeatures([[min.x, min.y], [max.x, max.y]], { layers: ["vessel-symbols", "group-halos"] });
          onSelect([...new Set(hits.map((hit) => String(hit.properties?.id)).filter(Boolean))], "replace");
        }
        selectionMode.current = false;
        boxStart.current = null;
        suppressClick.current = true;
        setBox(null);
        map.dragPan.enable();
        map.touchZoomRotate.enable();
        onToolDone();
      }
    };
    canvas.addEventListener("pointerdown", pointerDown, { passive: false });
    canvas.addEventListener("pointermove", pointerMove, { passive: false });
    canvas.addEventListener("pointerup", pointerUp);
    canvas.addEventListener("pointercancel", cancelLongPress);
    const down = (e: MapMouseEvent) => {
      if (editingEnabled && tool === "select" && e.originalEvent.button === 0) {
        const hitBox: [[number, number], [number, number]] = [
          [e.point.x - 14, e.point.y - 14],
          [e.point.x + 14, e.point.y + 14],
        ];
        const waypoint = map.queryRenderedFeatures(hitBox, {
          layers: ["mission-waypoint-numbers", "mission-waypoints"],
        })[0];
        const poi = map.queryRenderedFeatures(hitBox, {
          layers: ["mission-poi-labels", "mission-pois"],
        })[0];
        const assembly = map.queryRenderedFeatures(hitBox, {
          layers: ["group-assembly-labels", "group-assembly-rings"],
        })[0];
        if (waypoint?.properties?.index !== undefined) {
          dragTarget.current = {
            kind: "waypoint",
            index: Number(waypoint.properties.index),
          };
          setDragMarker({
            x: e.point.x,
            y: e.point.y,
            color: String(waypoint.properties.color ?? "#e9a93f"),
          });
          map.dragPan.disable();
          canvas.style.cursor = "grabbing";
          return;
        }
        if (poi?.properties?.index !== undefined) {
          dragTarget.current = {
            kind: "poi",
            index: Number(poi.properties.index),
          };
          setDragMarker({
            x: e.point.x,
            y: e.point.y,
            color: "#e9a93f",
          });
          map.dragPan.disable();
          canvas.style.cursor = "grabbing";
          return;
        }
        if (assembly?.properties?.group) {
          dragTarget.current = {
            kind: "assembly",
            group: String(assembly.properties.group),
          };
          setDragMarker({
            x: e.point.x,
            y: e.point.y,
            color: String(assembly.properties.color ?? "#e9a93f"),
          });
          map.dragPan.disable();
          canvas.style.cursor = "grabbing";
          return;
        }
      }
      if (!editingEnabled || !["box", "include", "exclude"].includes(tool))
        return;
      selectionMode.current = true;
      boxStart.current = { x: e.point.x, y: e.point.y };
      map.dragPan.disable();
      setBox({ x: e.point.x, y: e.point.y, w: 0, h: 0 });
    };
    const move = (e: MapMouseEvent) => {
      if (dragTarget.current) {
        setMapHover(null);
        setDragMarker((current) => ({
          x: e.point.x,
          y: e.point.y,
          color: current?.color ?? "#e9a93f",
        }));
        return;
      }
      if (!selectionMode.current || !boxStart.current) {
        const feature = map.queryRenderedFeatures(e.point, {
          layers: [
            "mission-waypoint-numbers",
            "mission-poi-labels",
            "group-assembly-labels",
            "vessel-symbols",
            "surface-contact-symbols",
            "current-vectors",
            "wind-vectors",
          ],
        })[0];
        const properties = feature?.properties;
        let next: typeof mapHover = null;
        if (properties?.id && feature.layer.id === "vessel-symbols") {
          const vessel = fleetRef.current.vessels.find((item) => item.id === String(properties.id));
          if (vessel) next = {
            x: e.point.x,
            y: e.point.y,
            title: `${vessel.callsign} · ${vessel.designation}`,
            detail: `${vessel.group_code || "UNASSIGNED"} · ${vessel.class.name} · ${Math.round(vessel.telemetry.reserve * 100)}% reserve · ${vessel.telemetry.speed_mps.toFixed(1)} m/s · ${vessel.telemetry.mode.replaceAll("_", " ")}`,
            accent: vessel.group_color,
          };
        } else if (properties?.id && feature.layer.id === "surface-contact-symbols") {
          const contact = fleetRef.current.surface_contacts.find((item) => item.id === String(properties.id));
          if (contact) next = {
            x: e.point.x,
            y: e.point.y,
            title: `${contact.name} · ${contact.boat_id}`,
            detail: `${contact.class} · ${contact.speed_mps.toFixed(1)} m/s (${contact.speed_knots.toFixed(1)} kn) · ${contact.navigation_state} · heading ${Math.round(contact.heading_deg)}°`,
            accent: contact.color,
          };
        } else if (feature?.layer.id === "mission-waypoint-numbers") {
          next = { x: e.point.x, y: e.point.y, title: `Waypoint ${properties?.sequence ?? ""}`, detail: `${properties?.ownerGroup ? "Group-owned route point" : "Mission route point"} · drag while editing to reposition`, accent: String(properties?.color ?? "#e9a93f") };
        } else if (feature?.layer.id === "mission-poi-labels") {
          next = { x: e.point.x, y: e.point.y, title: String(properties?.name ?? "Mission point"), detail: `${String(properties?.poiKind ?? "point").replaceAll("_", " ")} · mission geometry`, accent: "#e9a93f" };
        } else if (feature?.layer.id === "group-assembly-labels") {
          next = { x: e.point.x, y: e.point.y, title: `${properties?.code ?? "Group"} hold point`, detail: `${String(properties?.formation ?? "formation").replaceAll("_", " ")} · ${properties?.spacing ?? "—"} m spacing · drag to relocate`, accent: String(properties?.color ?? "#e9a93f") };
        } else if (feature?.layer.id === "current-vectors" || feature?.layer.id === "wind-vectors") {
          next = { x: e.point.x, y: e.point.y, title: feature.layer.id === "wind-vectors" ? "Wind field" : "Surface current", detail: `${Number(properties?.speed ?? 0).toFixed(feature.layer.id === "wind-vectors" ? 1 : 2)} m/s · bearing ${Math.round(Number(properties?.bearing ?? 0))}° · NOAA-derived simulation fixture`, accent: feature.layer.id === "wind-vectors" ? "#d2c05d" : "#62c5a8" };
        }
        setMapHover(next);
        return;
      }
      setMapHover(null);
      const s = boxStart.current;
      setBox({
        x: Math.min(s.x, e.point.x),
        y: Math.min(s.y, e.point.y),
        w: Math.abs(e.point.x - s.x),
        h: Math.abs(e.point.y - s.y),
      });
    };
    const up = (e: MapMouseEvent) => {
      if (dragTarget.current) {
        const target = dragTarget.current;
        const point: Point = [e.lngLat.lng, e.lngLat.lat];
        if (map.queryRenderedFeatures(e.point, { layers: ["land"] }).length === 0) {
          if (target.kind === "waypoint") onMoveWaypoint(target.index, point);
          else if (target.kind === "poi") onMovePOI(target.index, point);
          else onMoveGroupAssembly(target.group, point);
        }
        dragTarget.current = null;
        suppressClick.current = true;
        setDragMarker(null);
        map.dragPan.enable();
        canvas.style.cursor = "default";
        return;
      }
      if (!selectionMode.current || !boxStart.current) return;
      const s = boxStart.current;
      const min = { x: Math.min(s.x, e.point.x), y: Math.min(s.y, e.point.y) },
        max = { x: Math.max(s.x, e.point.x), y: Math.max(s.y, e.point.y) };
      if (tool === "include" || tool === "exclude") {
        const nw = map.unproject([min.x, min.y]),
          se = map.unproject([max.x, max.y]);
        onArea(tool, [
          [nw.lng, nw.lat],
          [se.lng, nw.lat],
          [se.lng, se.lat],
          [nw.lng, se.lat],
          [nw.lng, nw.lat],
        ]);
      } else {
        const hits = map.queryRenderedFeatures(
          [
            [min.x, min.y],
            [max.x, max.y],
          ],
          { layers: ["vessel-symbols", "group-halos"] },
        );
        onSelect(
          [
            ...new Set(
              hits.map((h) => String(h.properties?.id)).filter(Boolean),
            ),
          ],
          e.originalEvent.shiftKey ? "toggle" : "replace",
        );
      }
      selectionMode.current = false;
      boxStart.current = null;
      map.dragPan.enable();
      setBox(null);
      onToolDone();
    };
    map.on("click", click);
    map.on("dblclick", dbl);
    map.on("contextmenu", context);
    map.on("mousedown", down);
    map.on("mousemove", move);
    map.on("mouseup", up);
    return () => {
      map.off("click", click);
      map.off("dblclick", dbl);
      map.off("contextmenu", context);
      map.off("mousedown", down);
      map.off("mousemove", move);
      map.off("mouseup", up);
      canvas.removeEventListener("pointerdown", pointerDown);
      canvas.removeEventListener("pointermove", pointerMove);
      canvas.removeEventListener("pointerup", pointerUp);
      canvas.removeEventListener("pointercancel", cancelLongPress);
    };
  }, [
    ready,
    tool,
    effectiveWaypointColor,
    editingEnabled,
    mission,
    onSelect,
    onGroup,
    onVessel,
    onOpenFleet,
    onContact,
    onPlanContact,
    onGeometryFocus,
    onWaypoint,
    onPOI,
    onMoveWaypoint,
    onMovePOI,
    onMoveGroupAssembly,
    onArea,
    onToolDone,
  ]);
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (document.querySelector(".context-menu-scrim")) return;
        if (editingEnabled && tool !== "select") onToolDone();
        else onSelect([], "replace");
        setContextMenu(null);
      }
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [editingEnabled, tool, onSelect, onToolDone]);
  const contextContact =
    contextMenu?.kind === "contact"
      ? fleet.surface_contacts.find((contact) => contact.id === contextMenu.contact)
      : undefined;
  const handleMultiClickCapture = (event: ReactMouseEvent<HTMLDivElement>) => {
    if (event.detail < 3 || !(event.target instanceof HTMLCanvasElement)) return;
    event.preventDefault();
    event.stopPropagation();
    onSelect(
      event.detail >= 4
        ? fleetRef.current.vessels.map((v) => v.id)
        : visibleVesselIDs(),
      "replace",
    );
  };
  return (
    <div
      className="operations-map"
      ref={host}
      role="application"
      aria-label="Fleet operating map. Tap to select. Long press for contextual actions."
      data-rendezvous-status={rendezvousStatus || undefined}
      data-visible-hold-groups={visibleHoldGroups.features.length}
      data-remaining-route-points={remainingMissionRoutes.features.reduce(
        (count, feature) => count + (feature.geometry.type === "LineString" ? feature.geometry.coordinates.length : 0),
        0,
      )}
      data-remaining-route-metres={Math.round(remainingMissionRoutes.features.reduce(
        (total, feature) => total + (feature.geometry.type === "LineString" ? routeLengthM(feature.geometry.coordinates as Point[]) : 0),
        0,
      ))}
      data-visible-mission-waypoints={visibleMissionGeometry.features.filter(
        (feature) => feature.properties?.kind === "waypoint",
      ).length}
      onClickCapture={handleMultiClickCapture}
      onPointerLeave={() => setMapHover(null)}
    >
      {box && (
        <div
          className="selection-box"
          style={{ left: box.x, top: box.y, width: box.w, height: box.h }}
        />
      )}
      {dragMarker && (
        <div
          className="map-drag-marker"
          style={{ left: dragMarker.x, top: dragMarker.y, borderColor: dragMarker.color }}
        />
      )}
      {mapHover && !contextMenu && (
        <div
          className="map-hover-help"
          style={{
            left: Math.min(mapHover.x + 14, (host.current?.clientWidth ?? 320) - 250),
            top: Math.min(mapHover.y + 14, (host.current?.clientHeight ?? 180) - 78),
            borderColor: mapHover.accent,
          }}
        >
          <strong>{mapHover.title}</strong>
          <span>{mapHover.detail}</span>
        </div>
      )}
      {contextMenu?.kind === "vessel" && (
        <div
          className="map-context-menu vessel-menu"
          role="menu"
          aria-label="Controlled vessel menu"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <strong>{pirate ? "SHIP ORDERS" : "VESSEL SELECTION"}</strong>
          <button
            role="menuitem"
            onClick={() => {
              onSelect([contextMenu.vessel], "replace");
              setContextMenu(null);
            }}
          >
            {pirate ? "Muster this ship" : "Select vessel"}
          </button>
          <button
            role="menuitem"
            onClick={() => {
              onSelect([contextMenu.vessel], "toggle");
              setContextMenu(null);
            }}
          >
            {selected.has(contextMenu.vessel)
              ? "Remove vessel from selection"
              : "Add vessel to selection"}
          </button>
          <button
            role="menuitem"
            onClick={() => {
              onGroup(contextMenu.group);
              setContextMenu(null);
            }}
          >
            {pirate ? "Muster this crew" : "Select operational group"}
          </button>
          <button
            role="menuitem"
            onClick={() => {
              onVessel(contextMenu.vessel);
              setContextMenu(null);
            }}
          >
            {pirate ? "Study this ship" : "Inspect vessel status"}
          </button>
          <button
            role="menuitem"
            onClick={() => {
              onOpenFleet();
              setContextMenu(null);
            }}
          >
            {pirate ? "Open the muster roll" : "Open Fleet / Groups"}
          </button>
          {contextMenu.group && (
            <>
              <strong>{pirate ? "CREW NAVIGATION" : "GROUP NAVIGATION"}</strong>
              <button
                role="menuitem"
                onClick={() => {
                  onHoldGroupAtVessel(contextMenu.group, contextMenu.vessel);
                  setContextMenu(null);
                }}
              >
                {pirate ? "Muster and hold on this ship" : "Hold group at this vessel"}
              </button>
            </>
          )}
        </div>
      )}
      {contextMenu?.kind === "contact" && contextContact && (
        <div
          className="map-context-menu contact-menu"
          role="menu"
          aria-label="Surface contact menu"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <strong>
            {pirate ? "VESSEL ON THE HORIZON" : "SURFACE CONTACT"}
            <small>{contextContact.boat_id}</small>
          </strong>
          <div className="contact-menu-summary">
            <i style={{ background: contextContact.color }} />
            <span>
              <b>{contextContact.name}</b>
              <small>
                {contextContact.class} · {contextContact.speed_mps.toFixed(1)} m/s · {contextContact.navigation_state}
              </small>
            </span>
          </div>
          <button
            role="menuitem"
            onClick={() => {
              onContact(contextContact.id);
              setContextMenu(null);
            }}
          >
            {pirate ? "Study this vessel" : "Inspect contact details"}
          </button>
          <button
            role="menuitem"
            title="Open Mission Canvas with this contact as an uncommitted objective"
            onClick={() => {
              onPlanContact(contextContact.id);
              setContextMenu(null);
            }}
          >
            {pirate ? "Plot a voyage involving this vessel" : "Plan mission involving this contact"}
          </button>
          <p>
            {pirate
              ? "Opens the plotter only; no voyage or authority is created."
              : "Opens Mission Canvas only. No mission or movement is created."}
          </p>
        </div>
      )}
      {contextMenu?.kind === "location" && (
        <div
          className="map-context-menu location-menu"
          role="menu"
          aria-label="Location inspection menu"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <strong>
            {pirate ? "CHART INSPECTION" : "LOCATION INSPECTION"}
            <small>
              {Math.abs(contextMenu.point[1]).toFixed(4)}°N ·{" "}
              {Math.abs(contextMenu.point[0]).toFixed(4)}°W
            </small>
          </strong>
          <div className="location-facts">
            <span><small>SURFACE</small><b>{contextMenu.surface}</b></span>
            <span><small>DEPTH</small><b>{contextMenu.surface === "land" ? "LAND" : contextMenu.depth ? `≈ ${contextMenu.depth} m` : "OPEN WATER"}</b></span>
            <span><small>CURRENT</small><b>{fleet.environment.current_speed_mps.toFixed(2)} m/s · {Math.round(fleet.environment.current_direction_deg)}°</b></span>
            <span><small>WIND</small><b>{fleet.environment.wind_speed_mps.toFixed(1)} m/s · {Math.round(fleet.environment.wind_direction_deg)}°</b></span>
          </div>
          <button
            role="menuitem"
            onClick={() => {
              mapRef.current?.easeTo({
                center: contextMenu.point,
                duration: 350,
              });
              setContextMenu(null);
            }}
          >
            {pirate ? "Center the chart here" : "Center map here"}
          </button>
        </div>
      )}
      <div className="map-fixture-label">
        <strong>
          {pirate ? "CHARTED WATERS · SIMULATION" : "SIMULATION ONLY"}
        </strong>
        <span>
          {pirate
            ? `${fleet.surface_contacts.length} neutral contacts · not for navigation`
            : `${fleet.surface_contacts.filter((contact) => contact.speed_mps > 0).length} underway · ${fleet.surface_contacts.filter((contact) => contact.speed_mps === 0).length} anchored contacts · local/offline`}
        </span>
      </div>
      <div
        className="environment-overlays"
        aria-label="Environmental map layers"
      >
        <div>
          <button
            className={overlays.current ? "on" : ""}
            title="Toggle the time-varying simulated surface-current field"
            onClick={() => setOverlays((v) => ({ ...v, current: !v.current }))}
          >
            <i className="current" />
            CURRENT <b>{fleet.environment.current_speed_mps.toFixed(2)} m/s</b>
          </button>
          <button
            className={overlays.wind ? "on" : ""}
            title="Toggle the time-varying simulated wind field"
            onClick={() => setOverlays((v) => ({ ...v, wind: !v.wind }))}
          >
            <i className="wind" />
            WIND <b>{fleet.environment.wind_speed_mps.toFixed(1)} m/s</b>
          </button>
          <button
            className={overlays.depth ? "on" : ""}
            title="Toggle local bathymetry contours and depth labels"
            onClick={() => setOverlays((v) => ({ ...v, depth: !v.depth }))}
          >
            <i className="depth" />
            DEPTH <b>5–80 m</b>
          </button>
        </div>
      </div>
    </div>
  );
}
