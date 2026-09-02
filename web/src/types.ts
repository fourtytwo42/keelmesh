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
  resilience?: ResilienceSnapshot;
};

export type TapeSegment = { sequence: number; activation_tick: number; expiry_tick: number; lifecycle: string; content_hash: string };
export type PntEstimate = { position: Point; uncertainty_m: number; integrity: string; contributing_sources: string[]; excluded_sources: string[]; reason_codes: string[]; behavior: string };
export type PntObservation = { source: string; position: Point; uncertainty_m: number; integrity_ok: boolean; reason_codes: string[] };
export type NodeSnapshot = { id: string; name: string; position: Point; behavior: string; tape: { depth_seconds: number; watermark: string; segments: TapeSegment[] }; active_route: string[]; buffered_bundles: number; buffered_events: number; execution_watermark: number; pnt: PntEstimate; pnt_observations: PntObservation[] };
export type LinkState = { id: string; source_id: string; destination_id: string; underlay: string; reachable: boolean; latency_ms: number; loss_percent: number; trusted: boolean };
export type RejoinBridge = { execution_watermark: number; discarded_sequences: number[]; target_sequence: number; target_tick: number; route: Point[]; policy_status: string; reason_codes: string[]; content_hash: string; requires_approval: boolean };
export type ResilienceSnapshot = {
  schema_version: number; state_version: number; scenario_id: string; phase: string; mission_tick: number;
  incident_node_id: string; relay_node_id: string; nodes: NodeSnapshot[]; links: LinkState[];
  active_path: string[]; hop_receipts: Array<{ bundle_id: string; relay_id: string; result: string }>;
  queued_bundles: number; duplicate_deliveries: number; discarded_sequences: number[];
  raw_gnss_position?: Point; bridge?: RejoinBridge; summary: string; next_action?: string; auto_run_available: boolean;
  pnt_transitions: PntEstimate[];
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
export type StreamMessage = { schema_version: number; sequence: number; kind: string; snapshot?: FleetSnapshot; audit?: AuditEvent; resilience?: ResilienceSnapshot };
export type APIError = { code: string; message: string };
