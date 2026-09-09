package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string                  { return e.Message }
func platformError(code, message string) *Error { return &Error{code, message} }

type Manager struct {
	mu                  sync.RWMutex
	cfg                 Config
	logger              *slog.Logger
	pool                *pgxpool.Pool
	producer            *kgo.Client
	snapshot            domain.PlatformSnapshotV1
	subs                map[chan domain.PlatformSnapshotV1]struct{}
	idempotency         map[string]string
	lastAttempted       int64
	lastBytes           int64
	lastSample          time.Time
	faultAt             time.Time
	faultBaselineLag    int64
	faultSawDown        bool
	lastRecoverySeconds float64
}

func NewManager(cfg Config, logger *slog.Logger) *Manager {
	return &Manager{cfg: cfg, logger: logger, snapshot: domain.PlatformSnapshotV1{SchemaVersion: 1, StateVersion: 1, Phase: "degraded", SampledAt: time.Now().UTC(), Summary: "Platform services are starting; mission control remains independent."}, subs: map[chan domain.PlatformSnapshotV1]struct{}{}, idempotency: map[string]string{}}
}
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.close()
			return
		case <-ticker.C:
			m.refresh(ctx)
		}
	}
}
func (m *Manager) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.producer != nil {
		m.producer.Close()
	}
	if m.pool != nil {
		m.pool.Close()
	}
}
func (m *Manager) connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pool == nil {
		pool, err := pgxpool.New(ctx, m.cfg.DatabaseURL)
		if err != nil {
			return err
		}
		if err = pool.Ping(ctx); err != nil {
			pool.Close()
			return err
		}
		m.pool = pool
	}
	if m.producer == nil {
		client, err := kgo.NewClient(kgo.SeedBrokers(m.cfg.Brokers...))
		if err != nil {
			return err
		}
		m.producer = client
	}
	return nil
}
func (m *Manager) Snapshot() domain.PlatformSnapshotV1 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return clonePlatform(m.snapshot)
}
func (m *Manager) Subscribe() (<-chan domain.PlatformSnapshotV1, func()) {
	ch := make(chan domain.PlatformSnapshotV1, 4)
	m.mu.Lock()
	m.subs[ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() { m.mu.Lock(); delete(m.subs, ch); close(ch); m.mu.Unlock() }
}

func (m *Manager) refresh(ctx context.Context) {
	if err := m.connect(ctx); err != nil {
		m.setDegraded(err)
		return
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	sample := domain.PlatformSnapshotV1{SchemaVersion: 1, Available: true, Phase: "ready", SampledAt: time.Now().UTC(), Services: []domain.ServiceNodeV1{{ID: "loadgen", Kind: "producer", State: "ready"}, {ID: "kafka", Kind: "broker", State: "ready"}, {ID: "postgres", Kind: "database", State: "ready"}, {ID: "core", Kind: "control", State: "ready"}}, Topics: []domain.TopicSnapshotV1{{Name: RawTopic, Partitions: 12}, {Name: AuditTopic, Partitions: 3}, {Name: QuarantineTopic, Partitions: 3}, {Name: ControlTopic, Partitions: 1}}}
	_ = pool.QueryRow(ctx, `SELECT state_version FROM platform_state WHERE singleton=true`).Scan(&sample.StateVersion)
	sample.Workers = []domain.WorkerSnapshotV1{}
	sample.Assignments = []domain.PartitionAssignmentV1{}
	sample.Quarantine = []domain.QuarantineRecordV1{}
	sample.SelectedTrace = []domain.TraceStageV1{}
	rows, err := pool.Query(ctx, `SELECT worker_id,pid,state,partitions,rebalance_epoch,processed,duplicates,out_of_order,quarantined,batch_rate,rss_bytes,cpu_percent,last_heartbeat FROM worker_heartbeats ORDER BY worker_id`)
	if err == nil {
		for rows.Next() {
			var w domain.WorkerSnapshotV1
			var processed, dups, ooo, q int64
			if rows.Scan(&w.ID, &w.PID, &w.State, &w.AssignedPartitions, &w.RebalanceEpoch, &processed, &dups, &ooo, &q, &w.BatchRate, &w.RSSBytes, &w.CPUPercent, &w.LastHeartbeat) == nil {
				if time.Since(w.LastHeartbeat) > 10*time.Second {
					w.State = "offline"
				}
				sample.Workers = append(sample.Workers, w)
				sample.Metrics.UniqueInserted += processed
				sample.Metrics.DuplicatesSuppressed += dups
				sample.Metrics.OutOfOrder += ooo
				sample.Metrics.Quarantined += q
				for _, p := range w.AssignedPartitions {
					sample.Assignments = append(sample.Assignments, domain.PartitionAssignmentV1{Topic: RawTopic, Partition: p, WorkerID: w.ID})
				}
			}
		}
		rows.Close()
	}
	var run domain.LoadRunV1
	var stopped *time.Time
	err = pool.QueryRow(ctx, `SELECT id,profile,seed,vessel_count,rate_hz,state,started_at,stopped_at,attempted,produced,throttled,dropped,bytes_produced FROM load_runs ORDER BY started_at DESC LIMIT 1`).Scan(&run.ID, &run.Profile, &run.Seed, &run.VesselCount, &run.RateHz, &run.State, &run.StartedAt, &stopped, &sample.Metrics.Attempted, &sample.Metrics.Produced, &sample.Metrics.Throttled, &sample.Metrics.Dropped, &sample.Metrics.BytesPerSecond)
	if err == nil {
		run.StoppedAt = stopped
		sample.ActiveRun = &run
		if run.State == "running" {
			sample.Phase = "load_running"
		}
	}
	var persisted int64
	if run.ID != "" {
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM telemetry_events WHERE run_id=$1`, run.ID).Scan(&persisted)
		_ = pool.QueryRow(ctx, `SELECT duplicates,out_of_order,quarantined FROM run_ingest_metrics WHERE run_id=$1`, run.ID).Scan(&sample.Metrics.DuplicatesSuppressed, &sample.Metrics.OutOfOrder, &sample.Metrics.Quarantined)
	}
	sample.Metrics.UniqueInserted = persisted
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM rebalance_events`).Scan(&sample.Metrics.RebalanceCount)
	qrows, qerr := pool.Query(ctx, `SELECT id,event_id,reason,original_topic,original_partition,original_offset,checksum,repair_state,created_at FROM quarantine_records ORDER BY created_at DESC LIMIT 10`)
	if qerr == nil {
		for qrows.Next() {
			var q domain.QuarantineRecordV1
			if qrows.Scan(&q.ID, &q.EventID, &q.Reason, &q.OriginalTopic, &q.OriginalPartition, &q.OriginalOffset, &q.Checksum, &q.RepairState, &q.CreatedAt) == nil {
				sample.Quarantine = append(sample.Quarantine, q)
			}
		}
		qrows.Close()
	}
	var replay domain.ReplayRunV1
	var completed *time.Time
	if pool.QueryRow(ctx, `SELECT id,source_run_id,state,live_count,shadow_count,live_checksum,shadow_checksum,matches,started_at,completed_at FROM replay_runs ORDER BY started_at DESC LIMIT 1`).Scan(&replay.ID, &replay.SourceRunID, &replay.State, &replay.LiveCount, &replay.ShadowCount, &replay.LiveChecksum, &replay.ShadowChecksum, &replay.Matches, &replay.StartedAt, &completed) == nil {
		replay.CompletedAt = completed
		sample.Replay = &replay
	}
	traceRows, traceErr := pool.Query(ctx, `SELECT event_id,stage,happened_at,service,detail FROM trace_stages WHERE event_id=(SELECT event_id FROM trace_stages ORDER BY happened_at DESC LIMIT 1) ORDER BY happened_at`)
	if traceErr == nil {
		for traceRows.Next() {
			var stage domain.TraceStageV1
			if traceRows.Scan(&stage.EventID, &stage.Stage, &stage.At, &stage.Service, &stage.Detail) == nil {
				sample.SelectedTrace = append(sample.SelectedTrace, stage)
			}
		}
		traceRows.Close()
	}
	var latestTraceID string
	if pool.QueryRow(ctx, `SELECT trace_id FROM otel_spans ORDER BY started_at DESC LIMIT 1`).Scan(&latestTraceID) == nil {
		if trace, traceErr := readTrace(ctx, pool, latestTraceID); traceErr == nil {
			sample.RealTrace = &trace
		}
	}
	_ = pool.QueryRow(ctx, `SELECT COALESCE(percentile_cont(.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM(received_at-produced_at))*1000),0),COALESCE(percentile_cont(.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM(received_at-produced_at))*1000),0),COALESCE(percentile_cont(.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM(received_at-produced_at))*1000),0) FROM telemetry_events WHERE received_at>now()-interval '60 seconds'`).Scan(&sample.Metrics.LatencyP50MS, &sample.Metrics.LatencyP95MS, &sample.Metrics.LatencyP99MS)
	_ = pool.QueryRow(ctx, `SELECT COALESCE(percentile_cont(.95) WITHIN GROUP (ORDER BY db_write_ms),0) FROM benchmark_samples WHERE sampled_at>now()-interval '60 seconds'`).Scan(&sample.Metrics.DBWriteP95MS)
	m.mu.RLock()
	producer := m.producer
	m.mu.RUnlock()
	admin := kadm.NewClient(producer)
	endOffsets, endErr := admin.ListEndOffsets(ctx, RawTopic)
	committed, commitErr := admin.FetchOffsetsForTopics(ctx, "keelmesh-ingestion-v1", RawTopic)
	if endErr == nil && commitErr == nil {
		var lag int64
		for partition, end := range endOffsets[RawTopic] {
			at := int64(0)
			if current, ok := committed.Lookup(RawTopic, partition); ok && current.At > 0 {
				at = current.At
			}
			partitionLag := max64(0, end.Offset-at)
			lag += partitionLag
			for i := range sample.Assignments {
				if sample.Assignments[i].Partition == partition {
					sample.Assignments[i].Lag = partitionLag
				}
			}
		}
		sample.Metrics.CurrentLag = lag
		if len(sample.Topics) > 0 {
			sample.Topics[0].CurrentLag = lag
		}
	}
	now := time.Now()
	m.mu.Lock()
	elapsed := now.Sub(m.lastSample).Seconds()
	totalBytes := int64(sample.Metrics.BytesPerSecond)
	if elapsed > 0 && m.lastSample.IsZero() == false {
		sample.Metrics.EventsPerSecond = float64(sample.Metrics.Attempted-m.lastAttempted) / elapsed
		sample.Metrics.BytesPerSecond = float64(totalBytes-m.lastBytes) / elapsed
	}
	m.lastAttempted = sample.Metrics.Attempted
	m.lastBytes = totalBytes
	m.lastSample = now
	if sample.Metrics.CurrentLag > m.snapshot.Metrics.PeakLag {
		sample.Metrics.PeakLag = sample.Metrics.CurrentLag
	} else {
		sample.Metrics.PeakLag = m.snapshot.Metrics.PeakLag
	}
	if len(sample.Topics) > 0 {
		sample.Topics[0].EventsPerSecond = sample.Metrics.EventsPerSecond
		sample.Topics[0].BytesPerSecond = sample.Metrics.BytesPerSecond
		sample.Topics[0].PeakLag = sample.Metrics.PeakLag
	}
	liveWorkers := 0
	for _, w := range sample.Workers {
		if w.State == "running" {
			liveWorkers++
		}
	}
	if !m.faultAt.IsZero() {
		if liveWorkers < 3 {
			m.faultSawDown = true
		}
		sample.Metrics.RecoverySeconds = now.Sub(m.faultAt).Seconds()
		if m.faultSawDown && liveWorkers == 3 && sample.Metrics.CurrentLag <= m.faultBaselineLag {
			m.lastRecoverySeconds = sample.Metrics.RecoverySeconds
			m.faultAt = time.Time{}
			m.faultSawDown = false
		}
	} else {
		sample.Metrics.RecoverySeconds = m.lastRecoverySeconds
	}
	if liveWorkers < 3 {
		sample.Phase = "degraded"
	}
	sample.Summary = fmt.Sprintf("%d vessels · %.0f events/s · %d workers · lag %d", func() int {
		if sample.ActiveRun != nil {
			return sample.ActiveRun.VesselCount
		}
		return 0
	}(), sample.Metrics.EventsPerSecond, liveWorkers, sample.Metrics.CurrentLag)
	m.snapshot = sample
	subs := make([]chan domain.PlatformSnapshotV1, 0, len(m.subs))
	for ch := range m.subs {
		subs = append(subs, ch)
	}
	m.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- clonePlatform(sample):
		default:
		}
	}
}

func (m *Manager) Trace(ctx context.Context, traceID string) (domain.TraceSnapshotV1, error) {
	if strings.TrimSpace(traceID) == "" {
		return domain.TraceSnapshotV1{}, platformError("TRACE_NOT_FOUND", "Trace ID is required.")
	}
	if err := m.connect(ctx); err != nil {
		return domain.TraceSnapshotV1{}, platformError("PLATFORM_UNAVAILABLE", "Trace storage is unavailable.")
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	trace, err := readTrace(ctx, pool, traceID)
	if err != nil {
		return trace, platformError("TRACE_NOT_FOUND", "Trace was not found.")
	}
	return trace, nil
}

func (m *Manager) RecentTraces(ctx context.Context, limit int) ([]domain.TraceSummaryV1, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if err := m.connect(ctx); err != nil {
		return nil, platformError("PLATFORM_UNAVAILABLE", "Trace storage is unavailable.")
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	rows, err := pool.Query(ctx, `SELECT trace_id,min(started_at),max(started_at + duration_ms * interval '1 millisecond'),count(*),array_agg(DISTINCT service ORDER BY service),bool_or(state NOT IN ('ok','accepted','completed')) FROM otel_spans GROUP BY trace_id ORDER BY min(started_at) DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.TraceSummaryV1{}
	for rows.Next() {
		var item domain.TraceSummaryV1
		var endedAt time.Time
		var failed bool
		if err := rows.Scan(&item.TraceID, &item.StartedAt, &endedAt, &item.SpanCount, &item.Services, &failed); err != nil {
			return nil, err
		}
		item.DurationMS = endedAt.Sub(item.StartedAt).Seconds() * 1000
		item.State = "ok"
		if failed {
			item.State = "error"
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (m *Manager) TraceCount24H(ctx context.Context) int64 {
	if m.connect(ctx) != nil {
		return 0
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	var count int64
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM otel_spans WHERE started_at > now()-interval '24 hours'`).Scan(&count)
	return count
}

// LatestCompletedCapacity returns a durable measurement from the newest
// completed load run. It lets the evidence UI remain useful while the
// generator is idle without presenting a projection as a live measurement.
func (m *Manager) LatestCompletedCapacity(ctx context.Context) (rate, ingestP95 float64, source string, sampledAt time.Time, ok bool) {
	if err := m.connect(ctx); err != nil {
		return 0, 0, "", time.Time{}, false
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	var runID string
	err := pool.QueryRow(ctx, `
		SELECT lr.id,
		       lr.produced / GREATEST(EXTRACT(EPOCH FROM (lr.stopped_at-lr.started_at)), 0.001),
		       COALESCE((SELECT percentile_cont(.95) WITHIN GROUP
		         (ORDER BY EXTRACT(EPOCH FROM (te.received_at-te.produced_at))*1000)
		         FROM telemetry_events te WHERE te.run_id=lr.id), 0),
		       lr.stopped_at
		FROM load_runs lr
		WHERE lr.state='stopped' AND lr.stopped_at IS NOT NULL AND lr.produced>0
		ORDER BY lr.stopped_at DESC LIMIT 1`).Scan(&runID, &rate, &ingestP95, &sampledAt)
	if err != nil {
		return 0, 0, "", time.Time{}, false
	}
	return rate, ingestP95, "completed load run " + runID, sampledAt.UTC(), true
}

func readTrace(ctx context.Context, pool *pgxpool.Pool, traceID string) (domain.TraceSnapshotV1, error) {
	trace := domain.TraceSnapshotV1{TraceID: traceID, Spans: []domain.SpanSnapshotV1{}}
	rows, err := pool.Query(ctx, `SELECT trace_id,span_id,parent_span_id,name,service,state,started_at,duration_ms,attributes FROM otel_spans WHERE trace_id=$1 ORDER BY started_at,span_id`, traceID)
	if err != nil {
		return trace, err
	}
	defer rows.Close()
	for rows.Next() {
		var span domain.SpanSnapshotV1
		var attributes []byte
		if err := rows.Scan(&span.TraceID, &span.SpanID, &span.ParentSpanID, &span.Name, &span.Service, &span.State, &span.StartedAt, &span.DurationMS, &attributes); err != nil {
			return trace, err
		}
		_ = json.Unmarshal(attributes, &span.Attributes)
		trace.Spans = append(trace.Spans, span)
	}
	if len(trace.Spans) == 0 {
		return trace, errors.New("trace not found")
	}
	return trace, rows.Err()
}
func (m *Manager) setDegraded(err error) {
	m.mu.Lock()
	m.snapshot.Available = false
	m.snapshot.Phase = "degraded"
	m.snapshot.SampledAt = time.Now().UTC()
	m.snapshot.Summary = "Scale services unavailable; M1/M2 remain operational."
	m.snapshot.Services = []domain.ServiceNodeV1{{ID: "platform", Kind: "optional", State: "unavailable", Detail: err.Error()}}
	m.mu.Unlock()
}

func (m *Manager) StartRun(ctx context.Context, req domain.LoadRunRequestV1) (domain.LoadRunV1, error) {
	if err := m.validateMutation(req.PlatformMutationV1, "start|"+req.Profile); err != nil {
		return domain.LoadRunV1{}, err
	}
	profiles := map[string][2]float64{"interview": {1000, 2}, "stress": {2500, 4}, "calibration": {1000, 1}}
	profile, ok := profiles[strings.ToLower(req.Profile)]
	if !ok {
		return domain.LoadRunV1{}, platformError("INVALID_LOAD_PROFILE", "Use interview, stress, or calibration.")
	}
	if err := m.connect(ctx); err != nil {
		return domain.LoadRunV1{}, platformError("PLATFORM_UNAVAILABLE", err.Error())
	}
	m.mu.RLock()
	pool, producer := m.pool, m.producer
	m.mu.RUnlock()
	var count int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM load_runs WHERE state='running'`).Scan(&count)
	if count > 0 {
		return domain.LoadRunV1{}, platformError("LOAD_ALREADY_RUNNING", "A load run is already active.")
	}
	run := domain.LoadRunV1{ID: fmt.Sprintf("run-%d", time.Now().UnixMilli()), Profile: strings.ToLower(req.Profile), Seed: req.Seed, VesselCount: int(profile[0]), RateHz: profile[1], State: "running", StartedAt: time.Now().UTC()}
	if run.Seed == 0 {
		run.Seed = 424242
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return run, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO load_runs(id,profile,seed,vessel_count,rate_hz,state,started_at)VALUES($1,$2,$3,$4,$5,$6,$7)`, run.ID, run.Profile, run.Seed, run.VesselCount, run.RateHz, run.State, run.StartedAt); err != nil {
		return run, err
	}
	if _, err = tx.Exec(ctx, `UPDATE platform_state SET state_version=state_version+1,updated_at=now()`); err != nil {
		return run, err
	}
	if err = tx.Commit(ctx); err != nil {
		return run, err
	}
	cmd := controlCommand{ID: req.RequestID, Kind: "load.start", Run: &run, CreatedAt: time.Now().UTC()}
	cmd.Signature = signCommand(cmd, m.cfg.ControlSecret)
	data, _ := json.Marshal(cmd)
	if err = producer.ProduceSync(ctx, &kgo.Record{Topic: ControlTopic, Key: []byte("loadgen"), Value: data}).FirstErr(); err != nil {
		return run, platformError("PLATFORM_UNAVAILABLE", err.Error())
	}
	return run, nil
}
func (m *Manager) StopRun(ctx context.Context, id string, req domain.PlatformMutationV1) (domain.LoadRunV1, error) {
	if err := m.validateMutation(req, "stop|"+id); err != nil {
		return domain.LoadRunV1{}, err
	}
	if err := m.connect(ctx); err != nil {
		return domain.LoadRunV1{}, err
	}
	m.mu.RLock()
	pool, producer := m.pool, m.producer
	m.mu.RUnlock()
	now := time.Now().UTC()
	var run domain.LoadRunV1
	err := pool.QueryRow(ctx, `UPDATE load_runs SET state='stopped',stopped_at=$2 WHERE id=$1 RETURNING id,profile,seed,vessel_count,rate_hz,state,started_at,stopped_at`, id, now).Scan(&run.ID, &run.Profile, &run.Seed, &run.VesselCount, &run.RateHz, &run.State, &run.StartedAt, &run.StoppedAt)
	if err != nil {
		return run, platformError("PLATFORM_UNAVAILABLE", err.Error())
	}
	cmd := controlCommand{ID: req.RequestID, Kind: "load.stop", TargetID: id, CreatedAt: now}
	cmd.Signature = signCommand(cmd, m.cfg.ControlSecret)
	data, _ := json.Marshal(cmd)
	_ = producer.ProduceSync(ctx, &kgo.Record{Topic: ControlTopic, Key: []byte("loadgen"), Value: data}).FirstErr()
	_, _ = pool.Exec(ctx, `UPDATE platform_state SET state_version=state_version+1,updated_at=now()`)
	return run, nil
}
func (m *Manager) Fault(ctx context.Context, req domain.PlatformFaultCommandV1) (domain.PlatformSnapshotV1, error) {
	if err := m.validateMutation(domain.PlatformMutationV1{RequestID: req.RequestID, IdempotencyKey: req.IdempotencyKey, ExpectedPlatformStateVersion: req.ExpectedPlatformStateVersion}, req.Kind+"|"+req.TargetID); err != nil {
		return domain.PlatformSnapshotV1{}, err
	}
	if req.Kind != "terminate_worker" || req.TargetID == "" {
		return domain.PlatformSnapshotV1{}, platformError("WORKER_NOT_FOUND", "Only terminate_worker with a worker target is supported.")
	}
	if err := m.connect(ctx); err != nil {
		return domain.PlatformSnapshotV1{}, err
	}
	if err := m.dispatchWorkerTermination(ctx, req.RequestID, req.TargetID); err != nil {
		return domain.PlatformSnapshotV1{}, err
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	_, _ = pool.Exec(ctx, `UPDATE platform_state SET state_version=state_version+1,updated_at=now()`)
	m.mu.Lock()
	m.faultAt = time.Now()
	m.faultBaselineLag = m.snapshot.Metrics.CurrentLag
	m.faultSawDown = false
	m.lastRecoverySeconds = 0
	m.mu.Unlock()
	return m.Snapshot(), nil
}

func (m *Manager) dispatchWorkerTermination(ctx context.Context, requestID, targetID string) error {
	m.mu.RLock()
	producer := m.producer
	m.mu.RUnlock()
	cmd := controlCommand{ID: requestID, Kind: "worker.terminate", TargetID: targetID, CreatedAt: time.Now().UTC()}
	cmd.Signature = signCommand(cmd, m.cfg.ControlSecret)
	data, _ := json.Marshal(cmd)
	return producer.ProduceSync(ctx, &kgo.Record{Topic: ControlTopic, Key: []byte(targetID), Value: data}).FirstErr()
}

func (m *Manager) StartDrill(ctx context.Context, req domain.PlatformDrillRequestV1) (domain.PlatformDrillReceiptV1, error) {
	if !req.Confirmed || strings.TrimSpace(req.ActorIdentity) == "" {
		return domain.PlatformDrillReceiptV1{}, platformError("DRILL_CONFIRMATION_REQUIRED", "A named operator must confirm the exact drill before it starts.")
	}
	if req.Type != "data_pipeline_worker_recovery" || strings.TrimSpace(req.TargetID) == "" {
		return domain.PlatformDrillReceiptV1{}, platformError("DRILL_UNSUPPORTED", "The supported bounded drill is data_pipeline_worker_recovery with an explicit worker target.")
	}
	if err := m.validateMutation(req.PlatformMutationV1, "drill|"+req.Type+"|"+req.TargetID); err != nil {
		return domain.PlatformDrillReceiptV1{}, err
	}
	if err := m.connect(ctx); err != nil {
		return domain.PlatformDrillReceiptV1{}, platformError("PLATFORM_UNAVAILABLE", err.Error())
	}
	snapshot := m.Snapshot()
	var worker *domain.WorkerSnapshotV1
	for i := range snapshot.Workers {
		if snapshot.Workers[i].ID == req.TargetID && snapshot.Workers[i].State == "running" {
			copy := snapshot.Workers[i]
			worker = &copy
			break
		}
	}
	if worker == nil {
		return domain.PlatformDrillReceiptV1{}, platformError("WORKER_NOT_FOUND", "The target must be a currently running ingestion worker.")
	}
	sum := sha256.Sum256([]byte(req.RequestID + "|" + req.TargetID))
	receipt := domain.PlatformDrillReceiptV1{SchemaVersion: 1, ID: "drill-" + hex.EncodeToString(sum[:6]), Type: req.Type, TargetID: req.TargetID, ActorIdentity: req.ActorIdentity, State: "running", Outcome: "pending", ExpectedInvariants: []string{"committed edge missions continue", "worker supervisor restarts the child", "consumer lag returns to baseline", "logical projection remains deduplicated"}, InitialState: map[string]string{"worker_pid": strconv.Itoa(worker.PID), "consumer_lag": strconv.FormatInt(snapshot.Metrics.CurrentLag, 10), "platform_state_version": strconv.FormatInt(snapshot.StateVersion, 10)}, StartedAt: time.Now().UTC()}
	receipt.Observations = append(receipt.Observations, domain.PlatformDrillObservationV1{At: receipt.StartedAt, Kind: "preflight_passed", Summary: "Target worker is healthy; the supervisor rollback path is active.", SourceID: worker.ID})
	if err := m.storeDrill(ctx, receipt); err != nil {
		return receipt, platformError("PLATFORM_UNAVAILABLE", err.Error())
	}
	if err := m.dispatchWorkerTermination(ctx, req.RequestID+"-terminate", req.TargetID); err != nil {
		receipt.State, receipt.Outcome, receipt.Reason = "failed", "failed", err.Error()
		now := time.Now().UTC()
		receipt.CompletedAt = &now
		_ = m.finalizeDrill(context.Background(), &receipt)
		return receipt, err
	}
	receipt.Observations = append(receipt.Observations, domain.PlatformDrillObservationV1{At: time.Now().UTC(), Kind: "fault_dispatched", Summary: "A signed worker-child termination command was published; the supervisor remains active.", SourceID: req.TargetID})
	_ = m.storeDrill(ctx, receipt)
	go m.monitorWorkerDrill(receipt, worker.PID, snapshot.Metrics.CurrentLag)
	return receipt, nil
}

func (m *Manager) monitorWorkerDrill(receipt domain.PlatformDrillReceiptV1, initialPID int, initialLag int64) {
	deadline := time.NewTimer(75 * time.Second)
	ticker := time.NewTicker(time.Second)
	defer deadline.Stop()
	defer ticker.Stop()
	sawDown := false
	for {
		select {
		case <-deadline.C:
			receipt.State, receipt.Outcome, receipt.Reason = "failed", "failed", "Worker did not restart and drain lag within 75 seconds."
			now := time.Now().UTC()
			receipt.CompletedAt = &now
			_ = m.finalizeDrill(context.Background(), &receipt)
			return
		case <-ticker.C:
			if m.drillState(receipt.ID) == "aborted" {
				return
			}
			snapshot := m.Snapshot()
			var current *domain.WorkerSnapshotV1
			for i := range snapshot.Workers {
				if snapshot.Workers[i].ID == receipt.TargetID {
					copy := snapshot.Workers[i]
					current = &copy
					break
				}
			}
			if !sawDown && (current == nil || current.State != "running") {
				sawDown = true
				receipt.Observations = append(receipt.Observations, domain.PlatformDrillObservationV1{At: time.Now().UTC(), Kind: "worker_interrupted", Summary: "The original worker child stopped and Kafka began reassignment.", SourceID: receipt.TargetID})
				_ = m.storeDrill(context.Background(), receipt)
			}
			if sawDown && current != nil && current.State == "running" && current.PID != initialPID && snapshot.Metrics.CurrentLag <= max64(100, initialLag) {
				now := time.Now().UTC()
				receipt.State, receipt.Outcome, receipt.CompletedAt = "completed", "passed", &now
				receipt.RecoveryMS = now.Sub(receipt.StartedAt).Milliseconds()
				receipt.FinalState = map[string]string{"worker_pid": strconv.Itoa(current.PID), "consumer_lag": strconv.FormatInt(snapshot.Metrics.CurrentLag, 10), "duplicates_suppressed": strconv.FormatInt(snapshot.Metrics.DuplicatesSuppressed, 10)}
				receipt.Observations = append(receipt.Observations, domain.PlatformDrillObservationV1{At: now, Kind: "recovery_verified", Summary: "A new worker child is running and consumer lag returned to the bounded baseline.", SourceID: receipt.TargetID})
				_ = m.finalizeDrill(context.Background(), &receipt)
				return
			}
		}
	}
}

func (m *Manager) storeDrill(ctx context.Context, receipt domain.PlatformDrillReceiptV1) error {
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	payload, _ := json.Marshal(receipt)
	_, err := pool.Exec(ctx, `INSERT INTO platform_drills(id,drill_type,target_id,state,payload,evidence_hash,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET state=excluded.state,payload=excluded.payload,evidence_hash=excluded.evidence_hash,completed_at=excluded.completed_at`, receipt.ID, receipt.Type, receipt.TargetID, receipt.State, payload, receipt.EvidenceHash, receipt.StartedAt, receipt.CompletedAt)
	return err
}

func (m *Manager) finalizeDrill(ctx context.Context, receipt *domain.PlatformDrillReceiptV1) error {
	receipt.EvidenceHash = ""
	payload, _ := json.Marshal(receipt)
	sum := sha256.Sum256(payload)
	receipt.EvidenceHash = "sha256:" + hex.EncodeToString(sum[:])
	return m.storeDrill(ctx, *receipt)
}

func (m *Manager) drillState(id string) string {
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	var state string
	_ = pool.QueryRow(context.Background(), `SELECT state FROM platform_drills WHERE id=$1`, id).Scan(&state)
	return state
}

func (m *Manager) Drills(ctx context.Context) []domain.PlatformDrillReceiptV1 {
	result := m.externalDrills()
	if m.connect(ctx) != nil {
		return result
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	rows, err := pool.Query(ctx, `SELECT payload FROM platform_drills ORDER BY started_at DESC LIMIT 25`)
	if err != nil {
		return result
	}
	defer rows.Close()
	seen := make(map[string]bool, len(result))
	for _, receipt := range result {
		seen[receipt.ID] = true
	}
	for rows.Next() {
		var payload []byte
		var receipt domain.PlatformDrillReceiptV1
		if rows.Scan(&payload) == nil && json.Unmarshal(payload, &receipt) == nil && !seen[receipt.ID] {
			result = append(result, receipt)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.After(result[j].StartedAt) })
	if len(result) > 25 {
		result = result[:25]
	}
	return result
}

func (m *Manager) Drill(ctx context.Context, id string) (domain.PlatformDrillReceiptV1, error) {
	if err := m.connect(ctx); err != nil {
		for _, receipt := range m.externalDrills() {
			if receipt.ID == id {
				return receipt, nil
			}
		}
		return domain.PlatformDrillReceiptV1{}, err
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	var payload []byte
	if err := pool.QueryRow(ctx, `SELECT payload FROM platform_drills WHERE id=$1`, id).Scan(&payload); err != nil {
		for _, receipt := range m.externalDrills() {
			if receipt.ID == id {
				return receipt, nil
			}
		}
		return domain.PlatformDrillReceiptV1{}, platformError("DRILL_NOT_FOUND", "Drill receipt was not found.")
	}
	var receipt domain.PlatformDrillReceiptV1
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

// externalDrills imports only completed, self-hashed receipts from the
// read-only operator evidence directory. The privileged runner owns that
// directory; the public HTTP service cannot create or modify these records.
func (m *Manager) externalDrills() []domain.PlatformDrillReceiptV1 {
	dir := strings.TrimSpace(m.cfg.EvidenceDir)
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	result := make([]domain.PlatformDrillReceiptV1, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil || info.Size() > 1<<20 {
			continue
		}
		encoded, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			continue
		}
		var receipt domain.PlatformDrillReceiptV1
		var canonical map[string]any
		if json.Unmarshal(encoded, &receipt) != nil || json.Unmarshal(encoded, &canonical) != nil || receipt.ID == "" || receipt.CompletedAt == nil || receipt.EvidenceHash == "" {
			continue
		}
		expected := receipt.EvidenceHash
		canonical["evidence_hash"] = ""
		unsigned, marshalErr := json.Marshal(canonical)
		if marshalErr != nil {
			continue
		}
		digest := sha256.Sum256(unsigned)
		if expected != "sha256:"+hex.EncodeToString(digest[:]) {
			continue
		}
		result = append(result, receipt)
	}
	return result
}

func (m *Manager) AbortDrill(ctx context.Context, id string, req domain.PlatformMutationV1) (domain.PlatformDrillReceiptV1, error) {
	if err := m.validateMutation(req, "abort-drill|"+id); err != nil {
		return domain.PlatformDrillReceiptV1{}, err
	}
	receipt, err := m.Drill(ctx, id)
	if err != nil {
		return receipt, err
	}
	if receipt.State != "running" {
		return receipt, platformError("DRILL_NOT_RUNNING", "Only a running drill can be aborted.")
	}
	now := time.Now().UTC()
	receipt.State, receipt.Outcome, receipt.CompletedAt, receipt.Reason = "aborted", "aborted", &now, "Operator requested abort; the worker supervisor rollback remains active."
	receipt.Observations = append(receipt.Observations, domain.PlatformDrillObservationV1{At: now, Kind: "operator_abort", Summary: receipt.Reason})
	if err := m.finalizeDrill(ctx, &receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}
func (m *Manager) validateMutation(req domain.PlatformMutationV1, fingerprint string) error {
	if req.RequestID == "" || req.IdempotencyKey == "" {
		return platformError("INVALID_REQUEST", "request_id and idempotency_key are required")
	}
	snapshot := m.Snapshot()
	if req.ExpectedPlatformStateVersion != snapshot.StateVersion {
		return platformError("PLATFORM_STALE_STATE", "Platform state changed; refresh and retry.")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if prior, ok := m.idempotency[req.IdempotencyKey]; ok && prior != fingerprint {
		return platformError("FAULT_CONFLICT", "Idempotency key was used for another mutation.")
	}
	m.idempotency[req.IdempotencyKey] = fingerprint
	return nil
}

func (m *Manager) Quarantine(ctx context.Context) []domain.QuarantineRecordV1 {
	out := []domain.QuarantineRecordV1{}
	if m.connect(ctx) != nil {
		return out
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	rows, err := pool.Query(ctx, `SELECT id,event_id,reason,original_topic,original_partition,original_offset,checksum,repair_state,created_at FROM quarantine_records ORDER BY created_at DESC LIMIT 10`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var q domain.QuarantineRecordV1
		if rows.Scan(&q.ID, &q.EventID, &q.Reason, &q.OriginalTopic, &q.OriginalPartition, &q.OriginalOffset, &q.Checksum, &q.RepairState, &q.CreatedAt) == nil {
			out = append(out, q)
		}
	}
	return out
}
func (m *Manager) Redrive(ctx context.Context, id string, req domain.PlatformMutationV1) (domain.QuarantineRecordV1, error) {
	if err := m.validateMutation(req, "redrive|"+id); err != nil {
		return domain.QuarantineRecordV1{}, err
	}
	if err := m.connect(ctx); err != nil {
		return domain.QuarantineRecordV1{}, err
	}
	m.mu.RLock()
	pool, producer := m.pool, m.producer
	m.mu.RUnlock()
	var raw []byte
	var q domain.QuarantineRecordV1
	err := pool.QueryRow(ctx, `SELECT id,event_id,reason,original_topic,original_partition,original_offset,checksum,repair_state,created_at,envelope FROM quarantine_records WHERE id=$1`, id).Scan(&q.ID, &q.EventID, &q.Reason, &q.OriginalTopic, &q.OriginalPartition, &q.OriginalOffset, &q.Checksum, &q.RepairState, &q.CreatedAt, &raw)
	if err != nil {
		return q, platformError("QUARANTINE_NOT_FOUND", "Quarantine record not found.")
	}
	var event domain.EventEnvelopeV1
	if json.Unmarshal(raw, &event) != nil {
		return q, platformError("REPAIR_HASH_MISMATCH", "Stored envelope is invalid.")
	}
	event.Checksum = checksumPayload(event.Payload)
	data, _ := json.Marshal(event)
	if err = producer.ProduceSync(ctx, &kgo.Record{Topic: RetryTopic, Key: []byte(event.VesselID), Value: data}).FirstErr(); err != nil {
		return q, err
	}
	_, _ = pool.Exec(ctx, `UPDATE quarantine_records SET repair_state='redriven',repaired_at=now() WHERE id=$1`, id)
	q.RepairState = "redriven"
	return q, nil
}
func (m *Manager) Replay(ctx context.Context, sourceRunID string, req domain.PlatformMutationV1) (domain.ReplayRunV1, error) {
	if err := m.validateMutation(req, "replay|"+sourceRunID); err != nil {
		return domain.ReplayRunV1{}, err
	}
	if err := m.connect(ctx); err != nil {
		return domain.ReplayRunV1{}, err
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	// A request or process may disappear while a synchronous replay is running.
	// Retire abandoned rows so one dead caller cannot lock replay forever.
	_, _ = pool.Exec(ctx, `UPDATE replay_runs SET state='failed',completed_at=now() WHERE state='running' AND started_at < now()-interval '2 minutes'`)
	var active int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM replay_runs WHERE state='running'`).Scan(&active)
	if active > 0 {
		return domain.ReplayRunV1{}, platformError("REPLAY_IN_PROGRESS", "A replay is already running.")
	}
	replay := domain.ReplayRunV1{ID: fmt.Sprintf("replay-%d", time.Now().UnixMilli()), SourceRunID: sourceRunID, State: "running", StartedAt: time.Now().UTC()}
	_, err := pool.Exec(ctx, `INSERT INTO replay_runs(id,source_run_id,state,started_at)VALUES($1,$2,'running',$3)`, replay.ID, replay.SourceRunID, replay.StartedAt)
	if err != nil {
		return replay, err
	}
	// Rebuild from immutable Kafka history at each partition's earliest retained
	// offset. Direct assignment avoids a consumer-group rebalance and four readers
	// use Kafka's partition parallelism without changing deterministic projection.
	metadataClient, err := kgo.NewClient(kgo.SeedBrokers(m.cfg.Brokers...))
	if err != nil {
		return replay, err
	}
	defer metadataClient.Close()
	admin := kadm.NewClient(metadataClient)
	ends, err := admin.ListEndOffsets(ctx, RawTopic, RetryTopic)
	if err != nil {
		return replay, err
	}
	starts, err := admin.ListStartOffsets(ctx, RawTopic, RetryTopic)
	if err != nil {
		return replay, err
	}
	const replayReaders = 4
	assignments := make([]map[string]map[int32]kgo.Offset, replayReaders)
	readerEnds := make([]map[string]map[int32]int64, replayReaders)
	for i := 0; i < replayReaders; i++ {
		assignments[i] = map[string]map[int32]kgo.Offset{}
		readerEnds[i] = map[string]map[int32]int64{}
	}
	shard := 0
	for topic, partitions := range ends {
		for partition, end := range partitions {
			start := starts[topic][partition].Offset
			if end.Offset <= start {
				continue
			}
			reader := shard % replayReaders
			if assignments[reader][topic] == nil {
				assignments[reader][topic] = map[int32]kgo.Offset{}
				readerEnds[reader][topic] = map[int32]int64{}
			}
			assignments[reader][topic][partition] = kgo.NewOffset().At(start)
			readerEnds[reader][topic][partition] = end.Offset
			shard++
		}
	}
	latest := map[string]domain.EventEnvelopeV1{}
	var latestMu sync.Mutex
	replayCtx, cancel := context.WithTimeout(ctx, 65*time.Second)
	defer cancel()
	var readers sync.WaitGroup
	replayErrors := make(chan error, replayReaders)
	for reader := 0; reader < replayReaders; reader++ {
		if len(assignments[reader]) == 0 {
			continue
		}
		readers.Add(1)
		go func(owned map[string]map[int32]kgo.Offset, ownedEnds map[string]map[int32]int64) {
			defer readers.Done()
			client, clientErr := kgo.NewClient(kgo.SeedBrokers(m.cfg.Brokers...), kgo.ConsumePartitions(owned))
			if clientErr != nil {
				replayErrors <- clientErr
				return
			}
			defer client.Close()
			done := map[string]map[int32]bool{}
			for topic, partitions := range ownedEnds {
				done[topic] = map[int32]bool{}
				for partition := range partitions {
					done[topic][partition] = false
				}
			}
			for !allReplayPartitionsDone(done) && replayCtx.Err() == nil {
				fetches := client.PollRecords(replayCtx, 20000)
				fetches.EachRecord(func(record *kgo.Record) {
					if record.Offset >= ownedEnds[record.Topic][record.Partition]-1 {
						done[record.Topic][record.Partition] = true
					}
					var event domain.EventEnvelopeV1
					if json.Unmarshal(record.Value, &event) != nil || event.RunID != sourceRunID || checksumPayload(event.Payload) != event.Checksum {
						return
					}
					latestMu.Lock()
					if prior, ok := latest[event.VesselID]; !ok || event.Sequence > prior.Sequence {
						latest[event.VesselID] = event
					}
					latestMu.Unlock()
				})
			}
		}(assignments[reader], readerEnds[reader])
	}
	readers.Wait()
	close(replayErrors)
	if replayCtx.Err() != nil {
		_, _ = pool.Exec(context.Background(), `UPDATE replay_runs SET state='failed',completed_at=now() WHERE id=$1`, replay.ID)
		replay.State = "failed"
		return replay, platformError("PLATFORM_UNAVAILABLE", "Kafka replay timed out before reaching captured end offsets.")
	}
	for replayErr := range replayErrors {
		if replayErr != nil {
			return replay, replayErr
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return replay, err
	}
	defer tx.Rollback(ctx)
	for _, event := range latest {
		if _, err = tx.Exec(ctx, `INSERT INTO vessel_latest_shadow(replay_id,vessel_id,sequence,event_id,payload)VALUES($1,$2,$3,$4,$5) ON CONFLICT(replay_id,vessel_id)DO UPDATE SET sequence=excluded.sequence,event_id=excluded.event_id,payload=excluded.payload WHERE vessel_latest_shadow.sequence<excluded.sequence`, replay.ID, event.VesselID, event.Sequence, event.EventID, event.Payload); err != nil {
			return replay, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return replay, err
	}
	liveIDs, shadowIDs := []string{}, []string{}
	rows, _ := pool.Query(ctx, `SELECT event_id FROM vessel_latest WHERE run_id=$1 ORDER BY vessel_id`, sourceRunID)
	if rows != nil {
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			liveIDs = append(liveIDs, s)
		}
		rows.Close()
	}
	rows, _ = pool.Query(ctx, `SELECT event_id FROM vessel_latest_shadow WHERE replay_id=$1 ORDER BY vessel_id`, replay.ID)
	if rows != nil {
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			shadowIDs = append(shadowIDs, s)
		}
		rows.Close()
	}
	replay.LiveCount = int64(len(liveIDs))
	replay.ShadowCount = int64(len(shadowIDs))
	replay.LiveChecksum = hashStrings(liveIDs)
	replay.ShadowChecksum = hashStrings(shadowIDs)
	replay.Matches = replay.LiveCount == replay.ShadowCount && replay.LiveChecksum == replay.ShadowChecksum
	replay.State = "completed"
	now := time.Now().UTC()
	replay.CompletedAt = &now
	_, _ = pool.Exec(ctx, `UPDATE replay_runs SET state='completed',live_count=$2,shadow_count=$3,live_checksum=$4,shadow_checksum=$5,matches=$6,completed_at=$7 WHERE id=$1`, replay.ID, replay.LiveCount, replay.ShadowCount, replay.LiveChecksum, replay.ShadowChecksum, replay.Matches, now)
	return replay, nil
}
func allReplayPartitionsDone(done map[string]map[int32]bool) bool {
	for _, partitions := range done {
		for _, complete := range partitions {
			if !complete {
				return false
			}
		}
	}
	return true
}
func (m *Manager) Retrieval(ctx context.Context, query string) []domain.RetrievalHitV1 {
	if m.connect(ctx) != nil {
		return nil
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	parts := make([]string, 384)
	for i := range parts {
		parts[i] = "0"
	}
	if strings.Contains(strings.ToLower(query), "gnss") || strings.Contains(strings.ToLower(query), "spoof") {
		parts[1] = "1"
	} else {
		parts[0] = "1"
	}
	vector := "[" + strings.Join(parts, ",") + "]"
	rows, err := pool.Query(ctx, `SELECT id,title,summary,1-(embedding <=> $1::vector) similarity,provenance,fixture FROM incidents ORDER BY embedding <=> $1::vector LIMIT 3`, vector)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var hits []domain.RetrievalHitV1
	for rows.Next() {
		var h domain.RetrievalHitV1
		if rows.Scan(&h.ID, &h.Title, &h.Summary, &h.Similarity, &h.Provenance, &h.Fixture) == nil {
			hits = append(hits, h)
		}
	}
	return hits
}
func (m *Manager) ReplayByID(ctx context.Context, id string) (domain.ReplayRunV1, error) {
	if err := m.connect(ctx); err != nil {
		return domain.ReplayRunV1{}, err
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	var replay domain.ReplayRunV1
	var completed *time.Time
	err := pool.QueryRow(ctx, `SELECT id,source_run_id,state,live_count,shadow_count,live_checksum,shadow_checksum,matches,started_at,completed_at FROM replay_runs WHERE id=$1`, id).Scan(&replay.ID, &replay.SourceRunID, &replay.State, &replay.LiveCount, &replay.ShadowCount, &replay.LiveChecksum, &replay.ShadowChecksum, &replay.Matches, &replay.StartedAt, &completed)
	replay.CompletedAt = completed
	return replay, err
}
func (m *Manager) Evidence(ctx context.Context, runID string) domain.EvidenceReportV1 {
	snapshot := m.Snapshot()
	report := domain.EvidenceReportV1{RunID: runID, Commit: env("KEELMESH_COMMIT", "working-tree"), ImageDigest: env("KEELMESH_IMAGE_DIGEST", "local-compose"), Hardware: "VM 214 · 8 vCPU · 16 GiB RAM", GeneratedAt: time.Now().UTC(), Metrics: snapshot.Metrics, Workers: snapshot.Workers, Replay: snapshot.Replay}
	return report
}
func (m *Manager) Reset(ctx context.Context, req domain.PlatformMutationV1) error {
	if err := m.validateMutation(req, "reset"); err != nil {
		return err
	}
	if err := m.connect(ctx); err != nil {
		return err
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()
	_, err := pool.Exec(ctx, `TRUNCATE telemetry_events,vessel_latest,worker_heartbeats,rebalance_events,quarantine_records,trace_stages,replay_runs,vessel_latest_shadow,load_runs,benchmark_samples,run_ingest_metrics;UPDATE platform_state SET state_version=state_version+1,updated_at=now()`)
	if err == nil {
		m.mu.Lock()
		m.snapshot.Metrics = domain.PipelineMetricsV1{}
		m.lastAttempted = 0
		m.lastBytes = 0
		m.lastSample = time.Time{}
		m.mu.Unlock()
	}
	return err
}
func hashStrings(values []string) string {
	sort.Strings(values)
	sum := sha256.Sum256([]byte(strings.Join(values, "\n")))
	return hex.EncodeToString(sum[:])
}
func clonePlatform(v domain.PlatformSnapshotV1) domain.PlatformSnapshotV1 {
	data, _ := json.Marshal(v)
	var out domain.PlatformSnapshotV1
	_ = json.Unmarshal(data, &out)
	return out
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

var _ = errors.Is
