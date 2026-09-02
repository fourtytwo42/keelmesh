package platform

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

func updateLoadgenCounters(ctx context.Context, cfg Config, runID string, attempted, produced, bytesProduced, throttled, dropped int64) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return
	}
	defer pool.Close()
	_, _ = pool.Exec(ctx, `UPDATE load_runs SET attempted=$2,produced=$3,bytes_produced=$4,throttled=$5,dropped=$6 WHERE id=$1`, runID, attempted, produced, bytesProduced, throttled, dropped)
}
