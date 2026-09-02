package platform

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:embed schema.sql
var schemaSQL string

func Migrate(ctx context.Context, cfg Config) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func InitTopics(ctx context.Context, cfg Config) error {
	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...))
	if err != nil {
		return err
	}
	defer client.Close()
	admin := kadm.NewClient(client)
	topics := []struct {
		name       string
		partitions int32
		compact    bool
	}{
		{RawTopic, 12, false}, {AuditTopic, 3, false}, {RetryTopic, 3, false}, {QuarantineTopic, 3, false}, {WorkerStatusTopic, 3, true}, {ControlTopic, 1, true},
	}
	for _, topic := range topics {
		configs := map[string]*string{}
		if topic.compact {
			v := "compact"
			configs["cleanup.policy"] = &v
		} else {
			retention := "1800000"
			bytes := "134217728"
			configs["retention.ms"] = &retention
			configs["retention.bytes"] = &bytes
		}
		responses, err := admin.CreateTopics(ctx, topic.partitions, 1, configs, topic.name)
		if err != nil {
			return err
		}
		for _, response := range responses {
			if response.Err != nil && !strings.Contains(strings.ToUpper(response.Err.Error()), "ALREADY") {
				return fmt.Errorf("create %s: %w", response.Topic, response.Err)
			}
		}
	}
	return nil
}
