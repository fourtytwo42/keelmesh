export type Point = [number, number];
export type Polygon = { type: "Polygon"; coordinates: Point[][] };

export type Vessel = {
  id: string;
  name: string;
  position: Point;
  heading_deg: number;
  reserve: number;
  speed_mps: number;
  available: boolean;
  route_index: number;
  route_progress: number;
};

export type MissionState = {
  id?: string;
  phase: string;
  plan_id?: string;
  plan_hash?: string;
  lease_id?: string;
  started_at?: string;
  progress: number;
};

export type FleetSnapshot = {
  schema_version: number;
  state_version: number;
  scenario_id: string;
  scenario_name: string;
  simulation_rate: number;
  vessels: Vessel[];
  mission: MissionState;
};

export type Zone = { id: string; name: string; kind: string; geometry: Polygon };
export type AuditEvent = { schema_version: number; id: string; trace_id: string; kind: string; at: string; summary: string; details?: Record<string, unknown> };

export type Bootstrap = {
  schema_version: number;
  snapshot: FleetSnapshot;
  boundary: Zone;
  suggested_area: Zone;
  exclusion_zone: Zone;
  holding_area: Zone;
  capabilities: string[];
  audit: AuditEvent[];
};

export type MissionIntent = {
  schema_version: number;
  id: string;
  trace_id: string;
  source_state_version: number;
  objective: string;
  area: Polygon;
  requested_asset_count: number;
  constraints: { minimum_reserve: number; maximum_duration_minutes: number; avoid_zones: string[] };
  source_text: string;
  content_hash: string;
};

export type Assignment = { vessel_id: string; route: Point[]; speed_mps: number; distance_km: number };
export type PlanCandidate = {
  schema_version: number;
  id: string;
  intent_id: string;
  source_state_version: number;
  name: string;
  strategy: string;
  summary: string;
  assignments: Assignment[];
  metrics: { coverage_percent: number; duration_minutes: number; minimum_reserve: number; total_route_distance_km: number };
  policy: { status: string; reason_codes: string[]; summary: string };
  score: { coverage: number; reserve: number; duration: number; total: number };
  recommended: boolean;
  content_hash: string;
};

export type Preview = { plan_id: string; plan_hash: string; duration_seconds: number; samples: Array<{ second: number; positions: Record<string, Point> }> };
export type Lease = { id: string; mission_id: string; plan_id: string; plan_hash: string; operator_id: string; asset_ids: string[]; minimum_reserve: number; issued_at: string; expires_at: string; signature: string };
export type StreamMessage = { schema_version: number; sequence: number; kind: string; snapshot?: FleetSnapshot; audit?: AuditEvent };
export type APIError = { code: string; message: string };

