CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS platform_state (singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton), state_version bigint NOT NULL DEFAULT 1, updated_at timestamptz NOT NULL DEFAULT now());
INSERT INTO platform_state(singleton) VALUES(true) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS load_runs (id text PRIMARY KEY, profile text NOT NULL, seed bigint NOT NULL, vessel_count int NOT NULL, rate_hz double precision NOT NULL, state text NOT NULL, started_at timestamptz NOT NULL, stopped_at timestamptz, attempted bigint NOT NULL DEFAULT 0, produced bigint NOT NULL DEFAULT 0, throttled bigint NOT NULL DEFAULT 0, dropped bigint NOT NULL DEFAULT 0, bytes_produced bigint NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS telemetry_events (run_id text NOT NULL, vessel_id text NOT NULL, event_id text NOT NULL, event_type text NOT NULL, sequence bigint NOT NULL, produced_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT now(), payload jsonb NOT NULL, checksum text NOT NULL, PRIMARY KEY(vessel_id,event_id), UNIQUE(vessel_id,event_type,sequence)) PARTITION BY HASH(vessel_id);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_partitioned_table WHERE partrelid='telemetry_events'::regclass) THEN
    ALTER TABLE telemetry_events RENAME TO telemetry_events_unpartitioned;
    CREATE TABLE telemetry_events (run_id text NOT NULL, vessel_id text NOT NULL, event_id text NOT NULL, event_type text NOT NULL, sequence bigint NOT NULL, produced_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT now(), payload jsonb NOT NULL, checksum text NOT NULL, PRIMARY KEY(vessel_id,event_id), UNIQUE(vessel_id,event_type,sequence)) PARTITION BY HASH(vessel_id);
    CREATE TABLE telemetry_events_p0 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 0);
    CREATE TABLE telemetry_events_p1 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 1);
    CREATE TABLE telemetry_events_p2 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 2);
    CREATE TABLE telemetry_events_p3 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 3);
    CREATE TABLE telemetry_events_p4 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 4);
    CREATE TABLE telemetry_events_p5 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 5);
    CREATE TABLE telemetry_events_p6 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 6);
    CREATE TABLE telemetry_events_p7 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 7);
    INSERT INTO telemetry_events SELECT * FROM telemetry_events_unpartitioned;
    DROP TABLE telemetry_events_unpartitioned;
  END IF;
END $$;
CREATE TABLE IF NOT EXISTS telemetry_events_p0 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 0);
CREATE TABLE IF NOT EXISTS telemetry_events_p1 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 1);
CREATE TABLE IF NOT EXISTS telemetry_events_p2 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 2);
CREATE TABLE IF NOT EXISTS telemetry_events_p3 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 3);
CREATE TABLE IF NOT EXISTS telemetry_events_p4 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 4);
CREATE TABLE IF NOT EXISTS telemetry_events_p5 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 5);
CREATE TABLE IF NOT EXISTS telemetry_events_p6 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 6);
CREATE TABLE IF NOT EXISTS telemetry_events_p7 PARTITION OF telemetry_events FOR VALUES WITH (MODULUS 8, REMAINDER 7);
CREATE INDEX IF NOT EXISTS telemetry_events_run_idx ON telemetry_events(run_id);
CREATE TABLE IF NOT EXISTS vessel_latest (vessel_id text PRIMARY KEY, run_id text NOT NULL, sequence bigint NOT NULL, produced_at timestamptz NOT NULL, payload jsonb NOT NULL, event_id text NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS worker_heartbeats (worker_id text PRIMARY KEY, pid int NOT NULL, state text NOT NULL, partitions int[] NOT NULL DEFAULT '{}', rebalance_epoch bigint NOT NULL DEFAULT 0, processed bigint NOT NULL DEFAULT 0, duplicates bigint NOT NULL DEFAULT 0, out_of_order bigint NOT NULL DEFAULT 0, quarantined bigint NOT NULL DEFAULT 0, batch_rate double precision NOT NULL DEFAULT 0, rss_bytes bigint NOT NULL DEFAULT 0, cpu_percent double precision NOT NULL DEFAULT 0, last_heartbeat timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS benchmark_samples (id bigserial PRIMARY KEY, worker_id text NOT NULL, sampled_at timestamptz NOT NULL DEFAULT now(), db_write_ms double precision NOT NULL, batch_rate double precision NOT NULL);
CREATE TABLE IF NOT EXISTS run_ingest_metrics (run_id text PRIMARY KEY, accepted bigint NOT NULL DEFAULT 0, duplicates bigint NOT NULL DEFAULT 0, out_of_order bigint NOT NULL DEFAULT 0, quarantined bigint NOT NULL DEFAULT 0, updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS rebalance_events (id bigserial PRIMARY KEY, worker_id text NOT NULL, epoch bigint NOT NULL, kind text NOT NULL, partitions int[] NOT NULL DEFAULT '{}', happened_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS quarantine_records (id text PRIMARY KEY, event_id text NOT NULL, run_id text NOT NULL, reason text NOT NULL, original_topic text NOT NULL, original_partition int NOT NULL, original_offset bigint NOT NULL, checksum text NOT NULL, envelope jsonb NOT NULL, repair_state text NOT NULL DEFAULT 'pending', created_at timestamptz NOT NULL DEFAULT now(), repaired_at timestamptz);
CREATE TABLE IF NOT EXISTS trace_stages (event_id text NOT NULL, stage text NOT NULL, service text NOT NULL, detail text NOT NULL DEFAULT '', happened_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(event_id,stage,service));
CREATE TABLE IF NOT EXISTS replay_runs (id text PRIMARY KEY, source_run_id text NOT NULL, state text NOT NULL, live_count bigint NOT NULL DEFAULT 0, shadow_count bigint NOT NULL DEFAULT 0, live_checksum text NOT NULL DEFAULT '', shadow_checksum text NOT NULL DEFAULT '', matches boolean NOT NULL DEFAULT false, started_at timestamptz NOT NULL DEFAULT now(), completed_at timestamptz);
CREATE TABLE IF NOT EXISTS vessel_latest_shadow (replay_id text NOT NULL, vessel_id text NOT NULL, sequence bigint NOT NULL, event_id text NOT NULL, payload jsonb NOT NULL, PRIMARY KEY(replay_id,vessel_id));
CREATE TABLE IF NOT EXISTS incidents (id text PRIMARY KEY, title text NOT NULL, summary text NOT NULL, provenance text NOT NULL, fixture boolean NOT NULL DEFAULT true, embedding vector(384) NOT NULL);

-- M4 bounded AI infrastructure. Mission authority has no dependency on these tables.
CREATE TABLE IF NOT EXISTS ai_state (singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton), state_version bigint NOT NULL DEFAULT 1, updated_at timestamptz NOT NULL DEFAULT now());
INSERT INTO ai_state(singleton) VALUES(true) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS incident_manifests (id text PRIMARY KEY, manifest jsonb NOT NULL, state_checksum text NOT NULL, build_commit text NOT NULL, fixture boolean NOT NULL, captured_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS retrieval_documents (id text PRIMARY KEY, revision int NOT NULL, title text NOT NULL, body text NOT NULL, trust_label text NOT NULL, checksum text NOT NULL, provenance text NOT NULL, approved boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS retrieval_chunks (id text PRIMARY KEY, document_id text NOT NULL REFERENCES retrieval_documents(id), section text NOT NULL, content text NOT NULL, token_count int NOT NULL, embedding vector(384) NOT NULL, search tsvector GENERATED ALWAYS AS (to_tsvector('english', content)) STORED);
CREATE INDEX IF NOT EXISTS retrieval_chunks_embedding_idx ON retrieval_chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS retrieval_chunks_search_idx ON retrieval_chunks USING gin(search);
CREATE TABLE IF NOT EXISTS investigations (id text PRIMARY KEY, incident_id text NOT NULL, state text NOT NULL, trace_id text NOT NULL, payload jsonb NOT NULL, started_at timestamptz NOT NULL, completed_at timestamptz);
CREATE TABLE IF NOT EXISTS tool_receipts (id text PRIMARY KEY, investigation_id text NOT NULL, tool_name text NOT NULL, result_hash text NOT NULL, receipt jsonb NOT NULL, created_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS provider_attempts (id bigserial PRIMARY KEY, investigation_id text NOT NULL, provider text NOT NULL, model text NOT NULL, state text NOT NULL, latency_ms bigint NOT NULL, payload jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS replay_results (investigation_id text PRIMARY KEY, result jsonb NOT NULL, expected_checksum text NOT NULL, actual_checksum text NOT NULL, matches boolean NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS eval_candidates (id text PRIMARY KEY, investigation_id text NOT NULL, version int NOT NULL, candidate_hash text NOT NULL UNIQUE, state text NOT NULL, payload jsonb NOT NULL, approved_by text, approved_at timestamptz);
CREATE TABLE IF NOT EXISTS eval_runs (id text PRIMARY KEY, candidate_id text NOT NULL, suite_version text NOT NULL, state text NOT NULL, payload jsonb NOT NULL, started_at timestamptz NOT NULL, completed_at timestamptz);
CREATE TABLE IF NOT EXISTS otel_spans (trace_id text NOT NULL, span_id text NOT NULL, parent_span_id text NOT NULL DEFAULT '', service text NOT NULL, name text NOT NULL, state text NOT NULL, started_at timestamptz NOT NULL, duration_ms double precision NOT NULL, attributes jsonb NOT NULL DEFAULT '{}', PRIMARY KEY(trace_id,span_id));
CREATE INDEX IF NOT EXISTS otel_spans_started_idx ON otel_spans(started_at);
CREATE TABLE IF NOT EXISTS ai_security_events (id bigserial PRIMARY KEY, kind text NOT NULL, reason text NOT NULL, trace_id text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now());

-- M6 fleet operations workspace. These records persist operator organization and
-- mission workspaces while the deterministic execution engine remains isolated.
CREATE TABLE IF NOT EXISTS fleet_vessels (
  id text PRIMARY KEY,
  designation text NOT NULL UNIQUE,
  callsign text NOT NULL UNIQUE,
  class_id text NOT NULL,
  profile jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS operational_groups (
  id text PRIMARY KEY,
  revision bigint NOT NULL,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS saved_collections (
  id text PRIMARY KEY,
  revision bigint NOT NULL,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS mission_workspaces (
  id text PRIMARY KEY,
  version bigint NOT NULL,
  status text NOT NULL,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS mission_workspaces_status_idx ON mission_workspaces(status, updated_at DESC);
CREATE TABLE IF NOT EXISTS trajectory_programs (
  mission_id text PRIMARY KEY REFERENCES mission_workspaces(id) ON DELETE CASCADE,
  active_revision integer NOT NULL,
  content_hash text NOT NULL,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS mission_command_drafts (
  id text PRIMARY KEY,
  mission_id text NOT NULL,
  content_hash text NOT NULL UNIQUE,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS fleet_plans (
  id text PRIMARY KEY,
  mission_id text NOT NULL,
  content_hash text NOT NULL UNIQUE,
  state text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS operator_window_layouts (
  operator_id text PRIMARY KEY,
  revision bigint NOT NULL,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- M11 distributed agent memory. These projections are optional to mission
-- authority and can be deterministically rebuilt from the memory Kafka log.
CREATE TABLE IF NOT EXISTS memory_state (
  singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
  state_version bigint NOT NULL DEFAULT 1,
  central_watermark bigint NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO memory_state(singleton) VALUES(true) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS memory_conversation_turns (
  id text PRIMARY KEY,
  actor_id text NOT NULL,
  session_id text NOT NULL,
  mission_id text NOT NULL DEFAULT '',
  role text NOT NULL CHECK(role IN ('user','assistant','system')),
  content text NOT NULL CHECK(octet_length(content)<=16000),
  source_id text NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS memory_turn_scope_idx ON memory_conversation_turns(actor_id,session_id,mission_id,created_at DESC);
CREATE TABLE IF NOT EXISTS memory_items (
  id text PRIMARY KEY,
  scope_kind text NOT NULL CHECK(scope_kind IN ('operator','mission','vessel','group','faction','approved_global')),
  scope_id text NOT NULL,
  kind text NOT NULL,
  content text NOT NULL CHECK(octet_length(content)<=24000),
  revision integer NOT NULL,
  source jsonb NOT NULL,
  embedding vector(384) NOT NULL,
  embedding_version text NOT NULL,
  outcome_quality double precision NOT NULL CHECK(outcome_quality BETWEEN 0 AND 1),
  inferred boolean NOT NULL DEFAULT false,
  supersedes_id text,
  tombstoned boolean NOT NULL DEFAULT false,
  search tsvector GENERATED ALWAYS AS (to_tsvector('english',content)) STORED,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS memory_items_scope_idx ON memory_items(scope_kind,scope_id,updated_at DESC) WHERE NOT tombstoned;
CREATE INDEX IF NOT EXISTS memory_items_search_idx ON memory_items USING gin(search);
CREATE INDEX IF NOT EXISTS memory_items_embedding_idx ON memory_items USING hnsw(embedding vector_cosine_ops);
CREATE TABLE IF NOT EXISTS memory_revisions (
  item_id text NOT NULL REFERENCES memory_items(id),
  revision integer NOT NULL,
  content text NOT NULL,
  content_hash text NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY(item_id,revision)
);
CREATE TABLE IF NOT EXISTS memory_candidates (
  id text PRIMARY KEY,
  scope_kind text NOT NULL,
  scope_id text NOT NULL,
  kind text NOT NULL,
  content text NOT NULL,
  candidate_hash text NOT NULL UNIQUE,
  state text NOT NULL,
  requires_human boolean NOT NULL,
  source jsonb NOT NULL,
  decided_by text,
  created_at timestamptz NOT NULL,
  decided_at timestamptz
);
CREATE TABLE IF NOT EXISTS memory_entities (
  id text PRIMARY KEY,
  entity_type text NOT NULL,
  name text NOT NULL,
  scope_kind text NOT NULL,
  scope_id text NOT NULL,
  version bigint NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  updated_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS memory_edges (
  id text PRIMARY KEY,
  from_id text NOT NULL,
  to_id text NOT NULL,
  kind text NOT NULL,
  source_id text NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS memory_retrieval_receipts (
  id text PRIMARY KEY,
  turn_id text NOT NULL DEFAULT '',
  actor_id text NOT NULL,
  query_hash text NOT NULL,
  mode text NOT NULL,
  payload jsonb NOT NULL,
  duration_ms bigint NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS memory_context_assemblies (
  id text PRIMARY KEY,
  turn_id text NOT NULL,
  actor_id text NOT NULL,
  session_id text NOT NULL,
  mission_id text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  estimated_tokens integer NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS memory_tombstones (
  item_id text PRIMARY KEY,
  actor_id text NOT NULL,
  reason text NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS memory_contradictions (
  id bigserial PRIMARY KEY,
  existing_item_id text NOT NULL,
  candidate_id text NOT NULL,
  resolution text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS memory_node_sync (
  node_id text PRIMARY KEY,
  central_watermark bigint NOT NULL DEFAULT 0,
  local_watermark bigint NOT NULL DEFAULT 0,
  pending_bundles integer NOT NULL DEFAULT 0,
  state text NOT NULL DEFAULT 'ready',
  last_receipt_at timestamptz
);
CREATE TABLE IF NOT EXISTS memory_bundle_receipts (
  bundle_id text PRIMARY KEY,
  node_id text NOT NULL,
  content_hash text NOT NULL,
  state text NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS memory_replays (
  id text PRIMARY KEY,
  state text NOT NULL,
  payload jsonb NOT NULL,
  live_checksum text NOT NULL,
  replay_checksum text NOT NULL,
  matches boolean NOT NULL,
  started_at timestamptz NOT NULL,
  completed_at timestamptz
);
INSERT INTO incidents(id,title,summary,provenance,fixture,embedding) VALUES
('fixture-worker-rebalance','Consumer worker loss and cooperative recovery','A consumer child exited; partitions were reassigned and lag recovered after supervised restart.','deterministic M3 fixture',true,array_prepend(1::real,array_fill(0::real,ARRAY[383]))::vector),
('fixture-pnt-spoof','GNSS spoof rejected at the edge','A large GNSS jump was excluded while fused uncertainty increased and the vessel entered safe hold.','deterministic M2 fixture',true,array_prepend(0::real,array_prepend(1::real,array_fill(0::real,ARRAY[382])))::vector)
ON CONFLICT(id) DO UPDATE SET title=excluded.title,summary=excluded.summary,provenance=excluded.provenance,fixture=excluded.fixture,embedding=excluded.embedding;
