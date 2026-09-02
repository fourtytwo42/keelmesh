package platform

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/twmb/franz-go/pkg/kgo"
	bolt "go.etcd.io/bbolt"
)

const maxOutboxBytes = 64 << 20

type outboxRecord struct {
	Topic string `json:"topic"`
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

func RunLoadgen(ctx context.Context, cfg Config, logger *slog.Logger) error {
	producer, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.MaxBufferedRecords(20000), kgo.MaxBufferedBytes(32<<20))
	if err != nil {
		return err
	}
	defer producer.Close()
	consumer, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.ConsumerGroup("keelmesh-control-loadgen"), kgo.ConsumeTopics(ControlTopic), kgo.DisableAutoCommit())
	if err != nil {
		return err
	}
	defer consumer.Close()
	db, err := bolt.Open("/data/loadgen-outbox.db", 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	defer db.Close()
	_ = db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists([]byte("outbox")); return e })
	var mu sync.Mutex
	var counterMu sync.Mutex
	var active *domain.LoadRunV1
	sequences := map[int]int64{}
	delayed := map[int][]outboxRecord{}
	var ordinal, attempted, produced, bytesProduced, throttled, dropped, outboxSequence int64
	var outboxBytes int64
	_ = db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("outbox")).ForEach(func(_, v []byte) error { outboxBytes += int64(len(v)); return nil })
	})
	go func() {
		for ctx.Err() == nil {
			fetches := consumer.PollFetches(ctx)
			fetches.EachRecord(func(record *kgo.Record) {
				var cmd controlCommand
				if json.Unmarshal(record.Value, &cmd) != nil || !validCommand(cmd, cfg.ControlSecret) {
					return
				}
				mu.Lock()
				switch cmd.Kind {
				case "load.start":
					active = cmd.Run
					sequences = map[int]int64{}
					delayed = map[int][]outboxRecord{}
					counterMu.Lock()
					ordinal = 0
					attempted = 0
					produced = 0
					bytesProduced = 0
					throttled = 0
					dropped = 0
					counterMu.Unlock()
				case "load.stop":
					if active != nil && (cmd.TargetID == "" || cmd.TargetID == active.ID) {
						runID := active.ID
						active = nil
						producer.Flush(ctx)
						counterMu.Lock()
						a, p, b, t, d := attempted, produced, bytesProduced, throttled, dropped
						counterMu.Unlock()
						updateLoadgenCounters(context.Background(), cfg, runID, a, p, b, t, d)
					}
				}
				mu.Unlock()
			})
			_ = consumer.CommitUncommittedOffsets(ctx)
		}
	}()
	emit := func(records []outboxRecord) {
		for _, item := range records {
			item := item
			counterMu.Lock()
			attempted++
			counterMu.Unlock()
			producer.TryProduce(ctx, &kgo.Record{Topic: item.Topic, Key: item.Key, Value: item.Value}, func(_ *kgo.Record, produceErr error) {
				counterMu.Lock()
				defer counterMu.Unlock()
				if produceErr == nil {
					produced++
					bytesProduced += int64(len(item.Value))
					return
				}
				data, _ := json.Marshal(item)
				if outboxBytes+int64(len(data)) > maxOutboxBytes {
					dropped++
					return
				}
				outboxSequence++
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, uint64(time.Now().UnixNano()+outboxSequence))
				if db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("outbox")).Put(key, data) }) == nil {
					outboxBytes += int64(len(data))
					throttled++
				}
				logger.Warn("telemetry spooled to durable outbox", "error", produceErr)
			})
		}
	}
	drainOne := func() {
		var key, data []byte
		_ = db.View(func(tx *bolt.Tx) error {
			k, v := tx.Bucket([]byte("outbox")).Cursor().First()
			if k != nil {
				key = append([]byte(nil), k...)
				data = append([]byte(nil), v...)
			}
			return nil
		})
		if key == nil {
			return
		}
		var item outboxRecord
		if json.Unmarshal(data, &item) != nil {
			_ = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("outbox")).Delete(key) })
			return
		}
		sendCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		if producer.ProduceSync(sendCtx, &kgo.Record{Topic: item.Topic, Key: item.Key, Value: item.Value}).FirstErr() != nil {
			return
		}
		_ = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("outbox")).Delete(key) })
		counterMu.Lock()
		outboxBytes -= int64(len(data))
		produced++
		bytesProduced += int64(len(item.Value))
		counterMu.Unlock()
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastDB := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			drainOne()
			mu.Lock()
			run := active
			if run == nil {
				mu.Unlock()
				continue
			}
			count := int(float64(run.VesselCount) * run.RateHz / 10)
			if count < 1 {
				count = 1
			}
			for i := 0; i < count; i++ {
				ordinal++
				cycle := (ordinal - 1) / int64(run.VesselCount)
				vessel := int((ordinal - 1 + cycle) % int64(run.VesselCount))
				sequences[vessel]++
				envelope := makeEnvelope(*run, vessel, sequences[vessel], now.UTC())
				if ordinal%500 == 0 {
					envelope.Checksum = "sha256:invalid"
				}
				data, _ := json.Marshal(envelope)
				records := []outboxRecord{{Topic: RawTopic, Key: []byte(envelope.VesselID), Value: data}}
				if ordinal%100 == 0 {
					records = append(records, records[0])
				}
				pending := delayed[vessel]
				if len(pending) > 0 {
					emit(records)
					emit(pending)
					delete(delayed, vessel)
					continue
				}
				if ordinal%250 == 0 {
					delayed[vessel] = records
					continue
				}
				emit(records)
			}
			if time.Since(lastDB) >= time.Second {
				counterMu.Lock()
				a, p, b, t, d := attempted, produced, bytesProduced, throttled, dropped
				counterMu.Unlock()
				runID := run.ID
				lastDB = time.Now()
				go updateLoadgenCounters(context.Background(), cfg, runID, a, p, b, t, d)
			}
			mu.Unlock()
		}
	}
}
