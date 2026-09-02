package domain

import (
	"encoding/json"
	"time"
)

type EventEnvelopeV1 struct {
	SchemaVersion int             `json:"schema_version"`
	EventID       string          `json:"event_id"`
	LogicalKey    string          `json:"logical_key"`
	FleetID       string          `json:"fleet_id"`
	VesselID      string          `json:"vessel_id"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	PayloadSchema int             `json:"payload_schema_version"`
	TraceID       string          `json:"trace_id"`
	RunID         string          `json:"run_id"`
	ProducedAt    time.Time       `json:"produced_at"`
	Payload       json.RawMessage `json:"payload"`
	Checksum      string          `json:"checksum"`
}

type ServiceNodeV1 struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}
type PartitionAssignmentV1 struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	WorkerID  string `json:"worker_id"`
	Lag       int64  `json:"lag"`
}
type TopicSnapshotV1 struct {
	Name            string  `json:"name"`
	Partitions      int     `json:"partitions"`
	EventsPerSecond float64 `json:"events_per_second"`
	BytesPerSecond  float64 `json:"bytes_per_second"`
	CurrentLag      int64   `json:"current_lag"`
	PeakLag         int64   `json:"peak_lag"`
}
type WorkerSnapshotV1 struct {
	ID                 string    `json:"id"`
	PID                int       `json:"pid"`
	State              string    `json:"state"`
	AssignedPartitions []int32   `json:"assigned_partitions"`
	CPUPercent         float64   `json:"cpu_percent"`
	RSSBytes           int64     `json:"rss_bytes"`
	BatchRate          float64   `json:"batch_rate"`
	RebalanceEpoch     int64     `json:"rebalance_epoch"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
}
type PipelineMetricsV1 struct {
	Attempted            int64   `json:"attempted"`
	Produced             int64   `json:"produced"`
	UniqueInserted       int64   `json:"unique_inserted"`
	DuplicatesSuppressed int64   `json:"duplicates_suppressed"`
	OutOfOrder           int64   `json:"out_of_order"`
	Quarantined          int64   `json:"quarantined"`
	Replayed             int64   `json:"replayed"`
	Throttled            int64   `json:"throttled"`
	Dropped              int64   `json:"dropped"`
	EventsPerSecond      float64 `json:"events_per_second"`
	BytesPerSecond       float64 `json:"bytes_per_second"`
	LatencyP50MS         float64 `json:"latency_p50_ms"`
	LatencyP95MS         float64 `json:"latency_p95_ms"`
	LatencyP99MS         float64 `json:"latency_p99_ms"`
	DBWriteP95MS         float64 `json:"db_write_p95_ms"`
	CurrentLag           int64   `json:"current_lag"`
	PeakLag              int64   `json:"peak_lag"`
	RebalanceCount       int64   `json:"rebalance_count"`
	RecoverySeconds      float64 `json:"recovery_seconds"`
}
type LoadRunV1 struct {
	ID          string     `json:"id"`
	Profile     string     `json:"profile"`
	Seed        int64      `json:"seed"`
	VesselCount int        `json:"vessel_count"`
	RateHz      float64    `json:"rate_hz"`
	State       string     `json:"state"`
	StartedAt   time.Time  `json:"started_at"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
}
type PlatformFaultCommandV1 struct {
	RequestID                    string `json:"request_id"`
	IdempotencyKey               string `json:"idempotency_key"`
	ExpectedPlatformStateVersion int64  `json:"expected_platform_state_version"`
	Kind                         string `json:"kind"`
	TargetID                     string `json:"target_id"`
	Signature                    string `json:"signature,omitempty"`
}
type PlatformMutationV1 struct {
	RequestID                    string `json:"request_id"`
	IdempotencyKey               string `json:"idempotency_key"`
	ExpectedPlatformStateVersion int64  `json:"expected_platform_state_version"`
}
type LoadRunRequestV1 struct {
	PlatformMutationV1
	Profile string `json:"profile"`
	Seed    int64  `json:"seed"`
}
type QuarantineRecordV1 struct {
	ID                string    `json:"id"`
	EventID           string    `json:"event_id"`
	Reason            string    `json:"reason"`
	OriginalTopic     string    `json:"original_topic"`
	OriginalPartition int32     `json:"original_partition"`
	OriginalOffset    int64     `json:"original_offset"`
	Checksum          string    `json:"checksum"`
	RepairState       string    `json:"repair_state"`
	CreatedAt         time.Time `json:"created_at"`
}
type ReplayRunV1 struct {
	ID             string     `json:"id"`
	SourceRunID    string     `json:"source_run_id"`
	State          string     `json:"state"`
	LiveCount      int64      `json:"live_count"`
	ShadowCount    int64      `json:"shadow_count"`
	LiveChecksum   string     `json:"live_checksum"`
	ShadowChecksum string     `json:"shadow_checksum"`
	Matches        bool       `json:"matches"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
type RetrievalHitV1 struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary"`
	Similarity float64 `json:"similarity"`
	Provenance string  `json:"provenance"`
	Fixture    bool    `json:"fixture"`
}
type TraceStageV1 struct {
	EventID string    `json:"event_id"`
	Stage   string    `json:"stage"`
	At      time.Time `json:"at"`
	Service string    `json:"service"`
	Detail  string    `json:"detail,omitempty"`
}
type EvidenceReportV1 struct {
	RunID       string             `json:"run_id"`
	Commit      string             `json:"commit"`
	ImageDigest string             `json:"image_digest"`
	Hardware    string             `json:"hardware"`
	GeneratedAt time.Time          `json:"generated_at"`
	Metrics     PipelineMetricsV1  `json:"metrics"`
	Workers     []WorkerSnapshotV1 `json:"workers"`
	Replay      *ReplayRunV1       `json:"replay,omitempty"`
}
type PlatformSnapshotV1 struct {
	SchemaVersion int                     `json:"schema_version"`
	StateVersion  int64                   `json:"state_version"`
	Available     bool                    `json:"available"`
	Phase         string                  `json:"phase"`
	SampledAt     time.Time               `json:"sampled_at"`
	Services      []ServiceNodeV1         `json:"services"`
	Topics        []TopicSnapshotV1       `json:"topics"`
	Workers       []WorkerSnapshotV1      `json:"workers"`
	Assignments   []PartitionAssignmentV1 `json:"assignments"`
	Metrics       PipelineMetricsV1       `json:"metrics"`
	ActiveRun     *LoadRunV1              `json:"active_run,omitempty"`
	Quarantine    []QuarantineRecordV1    `json:"quarantine"`
	Replay        *ReplayRunV1            `json:"replay,omitempty"`
	SelectedTrace []TraceStageV1          `json:"selected_trace"`
	Retrieval     []RetrievalHitV1        `json:"retrieval"`
	Summary       string                  `json:"summary"`
}
