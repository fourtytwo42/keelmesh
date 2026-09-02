package platform

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Brokers       []string
	DatabaseURL   string
	ControlSecret string
	WorkerID      string
	HTTPAddress   string
}

func ConfigFromEnv() Config {
	brokers := strings.Split(env("KEELMESH_KAFKA_BROKERS", "kafka:9092"), ",")
	return Config{Brokers: brokers, DatabaseURL: env("KEELMESH_DATABASE_URL", "postgres://keelmesh:keelmesh@postgres:5432/keelmesh?sslmode=disable"), ControlSecret: env("KEELMESH_CONTROL_SECRET", "development-only-control-key"), WorkerID: env("KEELMESH_WORKER_ID", "worker-1"), HTTPAddress: env("KEELMESH_PLATFORM_HTTP", ":8090")}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err == nil {
		return v
	}
	return fallback
}

const (
	RawTopic          = "telemetry.raw.v1"
	AuditTopic        = "mission.audit.v1"
	RetryTopic        = "telemetry.retry.v1"
	QuarantineTopic   = "telemetry.quarantine.v1"
	WorkerStatusTopic = "platform.worker-status.v1"
	ControlTopic      = "platform.control.v1"
)
