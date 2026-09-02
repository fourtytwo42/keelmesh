package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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
	m.mu.RLock()
	producer := m.producer
	pool := m.pool
	m.mu.RUnlock()
	cmd := controlCommand{ID: req.RequestID, Kind: "worker.terminate", TargetID: req.TargetID, CreatedAt: time.Now().UTC()}
	cmd.Signature = signCommand(cmd, m.cfg.ControlSecret)
	data, _ := json.Marshal(cmd)
	if err := producer.ProduceSync(ctx, &kgo.Record{Topic: ControlTopic, Key: []byte(req.TargetID), Value: data}).FirstErr(); err != nil {
		return domain.PlatformSnapshotV1{}, err
	}
	_, _ = pool.Exec(ctx, `UPDATE platform_state SET state_version=state_version+1,updated_at=now()`)
	m.mu.Lock()
	m.faultAt = time.Now()
	m.faultBaselineLag = m.snapshot.Metrics.CurrentLag
	m.faultSawDown = false
	m.lastRecoverySeconds = 0
	m.mu.Unlock()
	return m.Snapshot(), nil
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
