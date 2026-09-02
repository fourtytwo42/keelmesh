package platform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func RunWorkerSupervisor(ctx context.Context, cfg Config, logger *slog.Logger) error {
	consumer, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.ConsumerGroup("keelmesh-control-"+cfg.WorkerID), kgo.ConsumeTopics(ControlTopic), kgo.DisableAutoCommit())
	if err != nil {
		return err
	}
	defer consumer.Close()
	type childResult struct{ err error }
	result := make(chan childResult, 1)
	var mu sync.Mutex
	var child *exec.Cmd
	holdUntil := time.Time{}
	spawn := func() {
		mu.Lock()
		defer mu.Unlock()
		command := exec.Command(os.Args[0], "--role", "worker-child")
		command.Env = append(os.Environ(), "KEELMESH_WORKER_ID="+cfg.WorkerID)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Start(); err != nil {
			logger.Error("start worker child", "error", err)
			return
		}
		child = command
		logger.Info("worker child started", "worker", cfg.WorkerID, "pid", command.Process.Pid)
		go func(c *exec.Cmd) { result <- childResult{c.Wait()} }(command)
	}
	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if child != nil && child.Process != nil {
			_ = child.Process.Signal(syscall.SIGTERM)
			time.Sleep(300 * time.Millisecond)
			_ = child.Process.Kill()
			child = nil
		}
	}
	spawn()
	poll := time.NewTicker(250 * time.Millisecond)
	defer poll.Stop()
	defer stop()
	go func() {
		for ctx.Err() == nil {
			fetches := consumer.PollFetches(ctx)
			fetches.EachRecord(func(record *kgo.Record) {
				var cmd controlCommand
				if json.Unmarshal(record.Value, &cmd) != nil || !validCommand(cmd, cfg.ControlSecret) || cmd.TargetID != cfg.WorkerID {
					return
				}
				if cmd.Kind == "worker.terminate" {
					logger.Warn("authorized worker termination", "worker", cfg.WorkerID)
					stop()
					mu.Lock()
					holdUntil = time.Now().Add(15 * time.Second)
					mu.Unlock()
				}
			})
			_ = consumer.CommitUncommittedOffsets(ctx)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case res := <-result:
			logger.Warn("worker child exited", "worker", cfg.WorkerID, "error", res.err)
			mu.Lock()
			child = nil
			if holdUntil.Before(time.Now()) {
				holdUntil = time.Now().Add(time.Second)
			}
			mu.Unlock()
		case <-poll.C:
			mu.Lock()
			should := child == nil && time.Now().After(holdUntil)
			mu.Unlock()
			if should {
				spawn()
			}
		}
	}
}

func RunWorkerChild(ctx context.Context, cfg Config, logger *slog.Logger) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	var assignments []int32
	var epoch int64
	var counters struct {
		processed, duplicates, outOfOrder, quarantined int64
		dbWriteMS                                      float64
	}
	var stateMu sync.Mutex
	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.ConsumerGroup("keelmesh-ingestion-v1"), kgo.ConsumeTopics(RawTopic, RetryTopic), kgo.DisableAutoCommit(), kgo.SessionTimeout(6*time.Second), kgo.HeartbeatInterval(2*time.Second), kgo.Balancers(kgo.CooperativeStickyBalancer()), kgo.OnPartitionsAssigned(func(c context.Context, _ *kgo.Client, m map[string][]int32) {
		stateMu.Lock()
		assignments = mergePartitions(assignments, m[RawTopic])
		epoch++
		localEpoch := epoch
		local := append([]int32(nil), assignments...)
		stateMu.Unlock()
		_, _ = pool.Exec(c, `INSERT INTO rebalance_events(worker_id,epoch,kind,partitions) VALUES($1,$2,'assigned',$3)`, cfg.WorkerID, localEpoch, local)
	}), kgo.OnPartitionsRevoked(func(c context.Context, _ *kgo.Client, m map[string][]int32) {
		stateMu.Lock()
		assignments = removePartitions(assignments, m[RawTopic])
		epoch++
		localEpoch := epoch
		stateMu.Unlock()
		_, _ = pool.Exec(c, `INSERT INTO rebalance_events(worker_id,epoch,kind,partitions) VALUES($1,$2,'revoked',$3)`, cfg.WorkerID, localEpoch, m[RawTopic])
	}))
	if err != nil {
		return err
	}
	defer client.Close()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		last := int64(0)
		lastTicks := processCPUTicks()
		lastCPUAt := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				ticks := processCPUTicks()
				cpu := float64(ticks-lastTicks) / 100 / now.Sub(lastCPUAt).Seconds() * 100
				lastTicks = ticks
				lastCPUAt = now
				stateMu.Lock()
				processed, dups, ooo, q, dbMS := counters.processed, counters.duplicates, counters.outOfOrder, counters.quarantined, counters.dbWriteMS
				parts := append([]int32(nil), assignments...)
				ep := epoch
				stateMu.Unlock()
				rate := float64(processed - last)
				last = processed
				_, _ = pool.Exec(context.Background(), `INSERT INTO worker_heartbeats(worker_id,pid,state,partitions,rebalance_epoch,processed,duplicates,out_of_order,quarantined,batch_rate,rss_bytes,cpu_percent,last_heartbeat) VALUES($1,$2,'running',$3,$4,$5,$6,$7,$8,$9,$10,$11,now()) ON CONFLICT(worker_id) DO UPDATE SET pid=excluded.pid,state=excluded.state,partitions=excluded.partitions,rebalance_epoch=excluded.rebalance_epoch,processed=excluded.processed,duplicates=excluded.duplicates,out_of_order=excluded.out_of_order,quarantined=excluded.quarantined,batch_rate=excluded.batch_rate,rss_bytes=excluded.rss_bytes,cpu_percent=excluded.cpu_percent,last_heartbeat=now()`, cfg.WorkerID, os.Getpid(), parts, ep, processed, dups, ooo, q, rate, processRSS(), cpu)
				_, _ = pool.Exec(context.Background(), `INSERT INTO benchmark_samples(worker_id,db_write_ms,batch_rate)VALUES($1,$2,$3)`, cfg.WorkerID, dbMS, rate)
			}
		}
	}()
	for ctx.Err() == nil {
		fetches := client.PollRecords(ctx, 1000)
		if fetches.IsClientClosed() {
			return nil
		}
		records := make([]*kgo.Record, 0, 1000)
		fetches.EachRecord(func(r *kgo.Record) { records = append(records, r) })
		if len(records) == 0 {
			continue
		}
		for start := 0; start < len(records); start += 200 {
			end := start + 200
			if end > len(records) {
				end = len(records)
			}
			batch := records[start:end]
			batchStarted := time.Now()
			accepted, duplicates, outOfOrder, quarantined, err := ingestBatch(ctx, pool, batch, cfg.WorkerID)
			dbWriteMS := float64(time.Since(batchStarted).Microseconds()) / 1000
			if err != nil {
				logger.Warn("ingest batch failed; offsets retained", "error", err)
				time.Sleep(250 * time.Millisecond)
				break
			}
			stateMu.Lock()
			counters.processed += accepted
			counters.duplicates += duplicates
			counters.outOfOrder += outOfOrder
			counters.quarantined += quarantined
			counters.dbWriteMS = dbWriteMS
			stateMu.Unlock()
			if err := client.CommitRecords(ctx, batch...); err != nil {
				logger.Warn("commit offsets", "error", err)
				break
			}
		}
	}
	return nil
}

type validRecord struct {
	event  domain.EventEnvelopeV1
	record *kgo.Record
}

func ingestBatch(ctx context.Context, pool *pgxpool.Pool, records []*kgo.Record, workerID string) (int64, int64, int64, int64, error) {
	validatedAt := time.Now().UTC()
	valid := make([]validRecord, 0, len(records))
	invalid := make([]struct {
		record *kgo.Record
		event  domain.EventEnvelopeV1
		reason string
	}, 0)
	for _, record := range records {
		var event domain.EventEnvelopeV1
		reason := ""
		if len(record.Value) > 64<<10 || json.Unmarshal(record.Value, &event) != nil {
			reason = "INVALID_SCHEMA"
		} else if event.SchemaVersion != 1 || event.RunID == "" || event.VesselID == "" {
			reason = "INVALID_SCHEMA"
		} else if checksumPayload(event.Payload) != event.Checksum {
			reason = "INVALID_CHECKSUM"
		}
		if reason != "" {
			invalid = append(invalid, struct {
				record *kgo.Record
				event  domain.EventEnvelopeV1
				reason string
			}{record, event, reason})
		} else {
			valid = append(valid, validRecord{event, record})
		}
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `CREATE TEMP TABLE IF NOT EXISTS ingest_stage(run_id text,vessel_id text,event_id text,event_type text,sequence bigint,produced_at timestamptz,payload jsonb,checksum text) ON COMMIT DELETE ROWS`); err != nil {
		return 0, 0, 0, 0, err
	}
	rows := make([][]any, 0, len(valid))
	for _, item := range valid {
		e := item.event
		rows = append(rows, []any{e.RunID, e.VesselID, e.EventID, e.Type, e.Sequence, e.ProducedAt, e.Payload, e.Checksum})
	}
	if len(rows) > 0 {
		if _, err = tx.CopyFrom(ctx, pgx.Identifier{"ingest_stage"}, []string{"run_id", "vessel_id", "event_id", "event_type", "sequence", "produced_at", "payload", "checksum"}, pgx.CopyFromRows(rows)); err != nil {
			return 0, 0, 0, 0, err
		}
	}
	var outOfOrder int64
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM ingest_stage s WHERE EXISTS(SELECT 1 FROM vessel_latest v WHERE v.vessel_id=s.vessel_id AND v.sequence>=s.sequence) OR EXISTS(SELECT 1 FROM ingest_stage newer WHERE newer.vessel_id=s.vessel_id AND newer.sequence>s.sequence)`).Scan(&outOfOrder); err != nil {
		return 0, 0, 0, 0, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO telemetry_events(run_id,vessel_id,event_id,event_type,sequence,produced_at,payload,checksum) SELECT run_id,vessel_id,event_id,event_type,sequence,produced_at,payload,checksum FROM ingest_stage ON CONFLICT DO NOTHING`)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	accepted := tag.RowsAffected()
	duplicates := int64(len(valid)) - accepted
	_, err = tx.Exec(ctx, `INSERT INTO vessel_latest(vessel_id,run_id,sequence,produced_at,payload,event_id) SELECT DISTINCT ON(vessel_id) vessel_id,run_id,sequence,produced_at,payload,event_id FROM ingest_stage ORDER BY vessel_id,sequence DESC ON CONFLICT(vessel_id) DO UPDATE SET run_id=excluded.run_id,sequence=excluded.sequence,produced_at=excluded.produced_at,payload=excluded.payload,event_id=excluded.event_id,updated_at=now() WHERE vessel_latest.sequence<excluded.sequence`)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, item := range invalid {
		data, _ := json.Marshal(item.event)
		id := fmt.Sprintf("q-%s-%d-%d", item.record.Topic, item.record.Partition, item.record.Offset)
		if _, err = tx.Exec(ctx, `INSERT INTO quarantine_records(id,event_id,run_id,reason,original_topic,original_partition,original_offset,checksum,envelope) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`, id, item.event.EventID, item.event.RunID, item.reason, item.record.Topic, item.record.Partition, item.record.Offset, item.event.Checksum, data); err != nil {
			return 0, 0, 0, 0, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, 0, 0, err
	}
	runID := ""
	if len(valid) > 0 {
		runID = valid[0].event.RunID
		first := valid[0]
		brokered := first.record.Timestamp
		if brokered.IsZero() {
			brokered = validatedAt
		}
		projected := time.Now().UTC()
		detail := fmt.Sprintf("batch=%d partition=%d offset=%d", len(valid), first.record.Partition, first.record.Offset)
		_, _ = pool.Exec(ctx, `INSERT INTO trace_stages(event_id,stage,service,detail,happened_at) VALUES($1,'produced','loadgen','',$2),($1,'brokered','kafka',$3,$4),($1,'assigned',$5,$3,$6),($1,'validated',$5,$3,$7),($1,'committed','postgres',$3,$8),($1,'projected','postgres',$3,$8) ON CONFLICT DO NOTHING`, first.event.EventID, first.event.ProducedAt, detail, brokered, workerID, validatedAt, projected, projected)
	} else if len(invalid) > 0 {
		runID = invalid[0].event.RunID
	}
	if runID != "" {
		_, _ = pool.Exec(ctx, `INSERT INTO run_ingest_metrics(run_id,accepted,duplicates,out_of_order,quarantined)VALUES($1,$2,$3,$4,$5) ON CONFLICT(run_id)DO UPDATE SET accepted=run_ingest_metrics.accepted+excluded.accepted,duplicates=run_ingest_metrics.duplicates+excluded.duplicates,out_of_order=run_ingest_metrics.out_of_order+excluded.out_of_order,quarantined=run_ingest_metrics.quarantined+excluded.quarantined,updated_at=now()`, runID, accepted, duplicates, outOfOrder, len(invalid))
	}
	return accepted, duplicates, outOfOrder, int64(len(invalid)), nil
}

func processRSS() int64 {
	file, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	fields := strings.Fields(scanner.Text())
	if len(fields) < 2 {
		return 0
	}
	pages, _ := strconv.ParseInt(fields[1], 10, 64)
	return pages * int64(os.Getpagesize())
}
func processCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0
	}
	user, _ := strconv.ParseInt(fields[13], 10, 64)
	system, _ := strconv.ParseInt(fields[14], 10, 64)
	return user + system
}
func mergePartitions(current, added []int32) []int32 {
	seen := map[int32]bool{}
	for _, p := range current {
		seen[p] = true
	}
	for _, p := range added {
		seen[p] = true
	}
	out := make([]int32, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
func removePartitions(current, removed []int32) []int32 {
	drop := map[int32]bool{}
	for _, p := range removed {
		drop[p] = true
	}
	out := make([]int32, 0, len(current))
	for _, p := range current {
		if !drop[p] {
			out = append(out, p)
		}
	}
	return out
}
