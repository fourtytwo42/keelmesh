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
INSERT INTO incidents(id,title,summary,provenance,fixture,embedding) VALUES
('fixture-worker-rebalance','Consumer worker loss and cooperative recovery','A consumer child exited; partitions were reassigned and lag recovered after supervised restart.','deterministic M3 fixture',true,array_prepend(1::real,array_fill(0::real,ARRAY[383]))::vector),
('fixture-pnt-spoof','GNSS spoof rejected at the edge','A large GNSS jump was excluded while fused uncertainty increased and the vessel entered safe hold.','deterministic M2 fixture',true,array_prepend(0::real,array_prepend(1::real,array_fill(0::real,ARRAY[382])))::vector)
ON CONFLICT(id) DO UPDATE SET title=excluded.title,summary=excluded.summary,provenance=excluded.provenance,fixture=excluded.fixture,embedding=excluded.embedding;
