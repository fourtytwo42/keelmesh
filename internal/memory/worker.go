package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

// RunWorker is the independent at-least-once memory candidate projection.
// Kafka offsets advance only after PostgreSQL acknowledges the immutable row.
func RunWorker(ctx context.Context, cfg Config, logger *slog.Logger) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.ConsumerGroup("keelmesh-memory-v1"), kgo.ConsumeTopics("memory.candidates.v1"), kgo.DisableAutoCommit(), kgo.FetchMaxBytes(4<<20))
	if err != nil {
		return err
	}
	defer client.Close()
	for {
		fetches := client.PollRecords(ctx, 200)
		if ctx.Err() != nil {
			return nil
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			logger.Warn("memory worker poll", "error", errs[0].Err)
			time.Sleep(250 * time.Millisecond)
			continue
		}
		for _, record := range fetches.Records() {
			var candidate domain.MemoryCandidateV1
			if json.Unmarshal(record.Value, &candidate) != nil || candidate.ID == "" || candidate.CandidateHash == "" || candidate.Content == "" {
				logger.Warn("memory worker rejected malformed candidate", "partition", record.Partition, "offset", record.Offset)
				_ = client.CommitRecords(ctx, record)
				continue
			}
			source, _ := json.Marshal(candidate.Source)
			_, err = pool.Exec(ctx, "INSERT INTO memory_candidates(id,scope_kind,scope_id,kind,content,candidate_hash,state,requires_human,source,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO NOTHING", candidate.ID, candidate.Scope.Kind, candidate.Scope.ID, candidate.Kind, candidate.Content, candidate.CandidateHash, candidate.State, candidate.RequiresHuman, source, candidate.CreatedAt)
			if err != nil {
				logger.Warn("memory worker projection paused", "error", err)
				time.Sleep(time.Second)
				break
			}
			if err = client.CommitRecords(ctx, record); err != nil {
				logger.Warn("memory worker commit", "error", err)
			}
		}
	}
}
