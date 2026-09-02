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
  quiet_fleet?: QuietFleetSnapshot;
};

export type CoordinationDecision = { node_id:string; proposal_hash:string; decision:"armed"|"reject"|"abstain"; reason_code?:string; decided_tick:number; signature:string };
export type GroupProposal = { id:string; revision:number; authority_epoch:number; reason:string; source:string; created_tick:number; expires_tick:number; affected_nodes:string[]; assignments:Assignment[]; content_hash:string; signature:string };
export type GroupCommit = { id:string; proposal_hash:string; authority_epoch:number; commit_tick:number; activation_tick:number; armed_nodes:string[]; content_hash:string; signature:string };
export type QuietFleetSnapshot = {
  schema_version:number; state_version:number; scenario_id:string; phase:string; mission_tick:number;
  contract:{ id:string; mission_id:string; plan_hash:string; authority_epoch:number; members:string[]; coordinator_order:string[]; quorum:number; window_interval_seconds:number; maximum_bytes_per_window:number; maximum_rounds:number; minimum_activation_lead_seconds:number; tape_boundary_seconds:number; bulk_traffic_suppressed:boolean; content_hash:string; signature:string };
  coordinator_id:string; vessel4_speed_mps:number; active_assignments:Assignment[]; proposal?:GroupProposal; decisions:CoordinationDecision[]; commit?:GroupCommit;
  windows:Array<{round:number;opens_tick:number;closes_tick:number;bytes_used:number;byte_budget:number;message_count:number;state:string}>;
  metrics:{rounds:number;bytes_sent:number;byte_budget:number;messages_sent:number;bulk_messages_suppressed:number;quorum_count:number;quorum_required:number;affected_armed:number;affected_required:number};
  summary:string; next_action?:string; auto_run_available:boolean; inference_label:string;
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
export type StreamMessage = { schema_version: number; sequence: number; kind: string; snapshot?: FleetSnapshot; audit?: AuditEvent; resilience?: ResilienceSnapshot; quiet_fleet?: QuietFleetSnapshot; platform?: PlatformSnapshot; ai?: AgentSnapshot };
export type APIError = { code: string; message: string };

export type PlatformMetrics = { attempted:number; produced:number; unique_inserted:number; duplicates_suppressed:number; out_of_order:number; quarantined:number; replayed:number; throttled:number; dropped:number; events_per_second:number; bytes_per_second:number; latency_p50_ms:number; latency_p95_ms:number; latency_p99_ms:number; db_write_p95_ms:number; current_lag:number; peak_lag:number; rebalance_count:number; recovery_seconds:number };
export type PlatformWorker = { id:string; pid:number; state:string; assigned_partitions:number[]; cpu_percent:number; rss_bytes:number; batch_rate:number; rebalance_epoch:number; last_heartbeat:string };
export type LoadRun = { id:string; profile:string; seed:number; vessel_count:number; rate_hz:number; state:string; started_at:string; stopped_at?:string };
export type QuarantineRecord = { id:string; event_id:string; reason:string; original_topic:string; original_partition:number; original_offset:number; checksum:string; repair_state:string; created_at:string };
export type ReplayRun = { id:string; source_run_id:string; state:string; live_count:number; shadow_count:number; live_checksum:string; shadow_checksum:string; matches:boolean; started_at:string; completed_at?:string };
export type PlatformSnapshot = { schema_version:number; state_version:number; available:boolean; phase:string; sampled_at:string; services:Array<{id:string;kind:string;state:string;detail?:string}>; topics:Array<{name:string;partitions:number;events_per_second:number;bytes_per_second:number;current_lag:number;peak_lag:number}>; workers:PlatformWorker[]; assignments:Array<{topic:string;partition:number;worker_id:string;lag:number}>; metrics:PlatformMetrics; active_run?:LoadRun; quarantine:QuarantineRecord[]; replay?:ReplayRun; selected_trace:Array<{event_id:string;stage:string;at:string;service:string;detail?:string}>; retrieval:Array<{id:string;title:string;summary:string;similarity:number;provenance:string;fixture:boolean}>; summary:string };

export type ProviderAttempt = { provider:string; model:string; state:string; started_at:string; latency_ms:number; status_code?:number; error_code?:string };
export type ToolReceipt = { id:string; tool:string; state:string; arguments:string; result_hash:string; at:string; duration_ms:number };
export type Citation = { source_id:string; chunk_id:string; title:string; trust:string; excerpt:string };
export type IncidentManifest = { schema_version:number; id:string; title:string; summary:string; scenario_seed:number; fault_schedule:string[]; evidence:Array<{id:string;kind:string;source_id:string;summary:string;trust:string;tick?:number}>; state_checksum:string; build_commit:string; fixture:boolean; classification:string; captured_at:string };
export type ReplayResult = { incident_id:string; state:string; expected_checksum:string; actual_checksum:string; matches:boolean; transition_count:number; first_divergence?:string; live_state_changed:boolean };
export type InvestigationRun = { schema_version:number; id:string; incident_id:string; state:string; diagnosis:string; confidence:number; evidence_ids:string[]; citations:Citation[]; tool_receipts:ToolReceipt[]; provider_attempts:ProviderAttempt[]; proposed_assertions:string[]; replay?:ReplayResult; candidate_id?:string; trace_id:string; started_at:string; completed_at?:string; failure?:string };
export type EvalCandidate = { schema_version:number; id:string; incident_id:string; investigation_id:string; version:number; assertions:string[]; evidence_ids:string[]; candidate_hash:string; state:string; created_at:string; approved_at?:string; approved_by?:string };
export type EvalResult = { case_id:string; provider:string; model:string; state:string; passed:number; failed:number; skipped:number; failures:string[]; latency_ms:number };
export type EvalRun = { schema_version:number; id:string; candidate_id:string; state:string; suite_version:string; results:EvalResult[]; started_at:string; completed_at?:string };
export type TraceSnapshot = { trace_id:string; spans:Array<{trace_id:string;span_id:string;parent_span_id?:string;name:string;service:string;state:string;started_at:string;duration_ms:number;attributes?:Record<string,string>}> };
export type AgentSnapshot = { schema_version:number; state_version:number; available:boolean; phase:string; provider:{mode:string;selected:string;models:string[];attempts:ProviderAttempt[];circuit_open:string[];local_enabled:boolean;cloud_enabled:boolean}; incidents:IncidentManifest[]; investigation?:InvestigationRun; candidate?:EvalCandidate; evaluation?:EvalRun; trace?:TraceSnapshot; security_denials:number; summary:string };
