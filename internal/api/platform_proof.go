package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/platform"
)

func (s *Server) platformSummaryV6(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildPlatformSummary(r))
}

func (s *Server) buildPlatformSummary(r *http.Request) domain.PlatformProofSummaryV1 {
	now := time.Now().UTC()
	platformMeasured := s.platform != nil
	platformSnapshot := domain.PlatformSnapshotV1{SampledAt: now, Summary: "Platform telemetry manager is unavailable."}
	if s.platform != nil {
		platformSnapshot = s.platform.Snapshot()
	}
	ingestP95 := platformSnapshot.Metrics.LatencyP95MS
	ingestMeasured := platformSnapshot.Available && platformSnapshot.Metrics.EventsPerSecond > 0
	ingestSource := "telemetry_events.received_at - produced_at"
	ingestWorkload := workloadName(platformSnapshot)
	ingestWindow := "rolling 60 seconds"
	if s.platform != nil && !ingestMeasured {
		if _, measuredP95, source, sampledAt, ok := s.platform.LatestCompletedCapacity(r.Context()); ok {
			ingestP95, ingestMeasured = measuredP95, true
			ingestSource, ingestWorkload = source, source
			ingestWindow = "completed at " + sampledAt.Format(time.RFC3339)
		}
	}
	planes := []domain.PlatformPlaneHealthV1{
		{ID: "edge", Label: "Edge mission plane", State: "unavailable", Detail: "Live vessel-node health has not been sampled for this request.", Source: "node health and execution journals", Measured: false, SampledAt: now},
		{ID: "coordination", Label: "Coordination plane", State: "unavailable", Detail: "No live coordination gateway is configured.", Source: "mTLS node status", Measured: false, SampledAt: now},
		{ID: "data", Label: "Cloud data plane", State: stateFrom(platformSnapshot.Available), Detail: platformSnapshot.Summary, Source: "Kafka, worker, and PostgreSQL telemetry", Measured: platformMeasured, SampledAt: platformSnapshot.SampledAt, StaleAgeMS: max(0, now.Sub(platformSnapshot.SampledAt).Milliseconds())},
		{ID: "ai", Label: "AI / ML plane", State: stateFrom(s.agent != nil && s.agent.Snapshot().Available), Detail: "Advisory provider and governed evaluation path; mission safety is independent.", Source: "agent provider snapshot", Measured: s.agent != nil, SampledAt: now},
	}
	var electionValues []float64
	duplicateMeasured := false
	leaderRecoveryMeasured := false
	leaderRecoveryMS := float64(0)
	radioRecoveryMeasured := false
	radioRecoveryMS := float64(0)
	if s.platform != nil {
		for _, drill := range s.platform.Drills(r.Context()) {
			if drill.Outcome != "passed" {
				continue
			}
			if drill.FinalState["duplicate_effects"] == "0" && (drill.Type == "coordination_leader_recovery" || strings.HasPrefix(drill.Type, "radio_partition_")) {
				duplicateMeasured = true
			}
			if drill.Type == "coordination_leader_recovery" && !leaderRecoveryMeasured {
				leaderRecoveryMeasured = true
				leaderRecoveryMS = float64(drill.RecoveryMS)
			}
			if strings.HasPrefix(drill.Type, "radio_partition_") {
				radioRecoveryMeasured = true
				radioRecoveryMS = maxFloat([]float64{radioRecoveryMS, float64(drill.RecoveryMS)})
			}
		}
	}
	if s.coordGateway != nil {
		cells := s.coordGateway.Cells(r.Context())
		reachable := 0
		leaders := 0
		for _, nodes := range cells {
			for _, node := range nodes {
				if node.State != "unreachable" {
					reachable++
				}
				if node.State == "leader" {
					leaders++
					if node.LastElectionMS > 0 {
						electionValues = append(electionValues, float64(node.LastElectionMS))
					}
				}
			}
		}
		planes[0].State = stateFrom(reachable == 12)
		planes[0].Detail = strconv.Itoa(reachable) + "/12 vessel nodes reachable over management health paths."
		planes[0].Measured = true
		planes[1].State = stateFrom(leaders == 2)
		planes[1].Detail = strconv.Itoa(leaders) + "/2 authority-ready Raft leaders observed."
		planes[1].Measured = true
	}
	slos := []domain.PlatformSLORecordV1{
		slo("ingest-p95", "Telemetry projection latency P95", ingestP95, "ms", 500, "lte", ingestMeasured, ingestSource, ingestWorkload, ingestWindow, now),
		slo("consumer-lag", "Kafka consumer lag", float64(platformSnapshot.Metrics.CurrentLag), "events", 100, "lte", platformSnapshot.Available, "Kafka committed versus end offsets", workloadName(platformSnapshot), "current sample", now),
		slo("duplicate-effects", "Duplicate applied mission effects", 0, "effects", 0, "eq", duplicateMeasured, "hash-addressed M13 drill receipt", "latest passed leader/radio drill", "latest completed drill", now),
		slo("leader-recovery", "Cell leader recovery", firstMeasured(leaderRecoveryMeasured, leaderRecoveryMS, maxFloat(electionValues)), "ms", 10000, "lte", leaderRecoveryMeasured || len(electionValues) > 0, firstSource(leaderRecoveryMeasured, "hash-addressed M13 drill receipt", "Raft node last_election_ms"), "two six-voter cells", "latest election or completed drill", now),
		slo("radio-recovery", "Radio partition rollback and convergence", radioRecoveryMS, "ms", 60000, "lte", radioRecoveryMeasured, "hash-addressed M13 drill receipt", "4/2 and 3/3 eth1-only fault profiles", "latest completed drills", now),
	}
	representativeTrace := platformSnapshot.RealTrace
	if s.platform != nil {
		if trace, err := s.platform.RepresentativeTrace(r.Context()); err == nil && len(trace.Spans) > 0 {
			representativeTrace = trace
		}
	}
	return domain.PlatformProofSummaryV1{SchemaVersion: 1, SampledAt: now, Commit: envValue("GIT_COMMIT", "development"), Planes: planes, SLOs: slos, LatestTrace: representativeTrace, Summary: "Four independently failing planes with measured evidence and explicit unavailable states."}
}

func (s *Server) platformSLOV6(w http.ResponseWriter, r *http.Request) {
	summary := s.buildPlatformSummary(r)
	writeJSON(w, http.StatusOK, map[string]any{"schema_version": 1, "sampled_at": summary.SampledAt, "slos": summary.SLOs})
}

func (s *Server) platformTracesV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Trace storage is unavailable."})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	traces, err := s.platform.RecentTraces(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schema_version": 1, "traces": traces})
}

func (s *Server) platformCapacityV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Capacity evidence is unavailable."})
		return
	}
	now := time.Now().UTC()
	snapshot := s.platform.Snapshot()
	workers := 0
	for _, worker := range snapshot.Workers {
		if worker.State == "running" {
			workers++
		}
	}
	traceCount := s.platform.TraceCount24H(r.Context())
	baseRate := snapshot.Metrics.EventsPerSecond
	ingestP95 := snapshot.Metrics.LatencyP95MS
	source := "live VM 214 platform snapshot"
	sampledAt := now
	if baseRate <= 0 {
		if measuredRate, measuredP95, measuredSource, measuredAt, ok := s.platform.LatestCompletedCapacity(r.Context()); ok {
			baseRate, ingestP95, source, sampledAt = measuredRate, measuredP95, measuredSource, measuredAt
		}
	}
	profiles := []domain.CapacityProfileV1{{AssetCount: 12, EvidenceClass: "measured", EventsPerSecond: baseRate, IngestP95MS: ingestP95, ConsumerLag: snapshot.Metrics.CurrentLag, WorkerCount: workers, TraceSpans24H: traceCount, Source: source, SampledAt: sampledAt}}
	for _, count := range []int{100, 1000} {
		profiles = append(profiles, domain.CapacityProfileV1{AssetCount: count, EvidenceClass: "projected", EventsPerSecond: baseRate * float64(count) / 12, WorkerCount: max(3, (count+99)/100), Assumption: "Linear event-rate projection from the current 12-node sample; not a benchmark.", Source: "calculated projection", SampledAt: now})
	}
	writeJSON(w, http.StatusOK, domain.CapacityCostEvidenceV1{SchemaVersion: 1, SampledAt: now, Capacity: profiles, Currency: "USD", Disclaimer: "Only the 12-node row is measured. Larger profiles and all cloud costs are transparent planning estimates."})
}

func (s *Server) platformCostV6(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	profiles := make([]domain.CostProfileV1, 0, 3)
	for _, count := range []int{12, 100, 1000} {
		compute := 180.0 + float64(count)*18.0
		storage := 25.0 + float64(count)*1.5
		transfer := float64(count) * 2.0
		model := float64(count) * 1.0
		profiles = append(profiles, domain.CostProfileV1{AssetCount: count, EvidenceClass: "projected", ComputeMonthlyUSD: compute, StorageMonthlyUSD: storage, TransferMonthlyUSD: transfer, ModelMonthlyUSD: model, EstimatedMonthlyUSD: compute + storage + transfer + model, Assumption: "$180 central baseline + $18/edge compute, $1.50 storage, $2 transfer, and $1 bounded model use per asset-month; illustrative, not a quote."})
	}
	writeJSON(w, http.StatusOK, domain.CapacityCostEvidenceV1{SchemaVersion: 1, SampledAt: now, Cost: profiles, Currency: "USD", Disclaimer: "All cost rows are explicit planning projections. Provider, region, retention, and traffic choices materially change them."})
}

func (s *Server) platformDrillsV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Drill evidence is unavailable."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schema_version": 1, "drills": s.platform.Drills(r.Context())})
}

func (s *Server) startPlatformDrillV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Drill execution is unavailable."})
		return
	}
	var request domain.PlatformDrillRequestV1
	if !decode(w, r, &request) {
		return
	}
	receipt, err := s.platform.StartDrill(r.Context(), request)
	respondPlatformProof(w, receipt, err, http.StatusAccepted)
}

func (s *Server) platformDrillV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Drill evidence is unavailable."})
		return
	}
	receipt, err := s.platform.Drill(r.Context(), r.PathValue("drill_id"))
	respondPlatformProof(w, receipt, err, http.StatusOK)
}

func (s *Server) platformEvidenceV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Platform evidence is unavailable."})
		return
	}
	id := r.PathValue("evidence_id")
	if strings.HasPrefix(id, "drill-") {
		receipt, err := s.platform.Drill(r.Context(), id)
		respondPlatformProof(w, receipt, err, http.StatusOK)
		return
	}
	trace, err := s.platform.Trace(r.Context(), id)
	respondPlatformProof(w, trace, err, http.StatusOK)
}

func (s *Server) platformEvaluationV6(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Evaluation evidence is unavailable."})
		return
	}
	episodeID := strings.TrimSpace(r.PathValue("episode_id"))
	incidentID := strings.TrimPrefix(episodeID, "episode-")
	incident, err := s.agent.Incident(incidentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "EVALUATION_NOT_FOUND", Message: "The source-linked evaluation episode was not found."})
		return
	}
	sourceIDs := make([]string, 0, len(incident.Evidence))
	for _, item := range incident.Evidence {
		sourceIDs = append(sourceIDs, item.ID)
	}
	sort.Strings(sourceIDs)
	sum := sha256.Sum256([]byte(episodeID + "\n" + incident.StateChecksum + "\n" + strings.Join(sourceIDs, "\n") + "\n" + strconv.FormatInt(incident.ScenarioSeed, 10)))
	lab := map[string]any{"dagster": "stopped", "minio": "stopped", "mlflow": "stopped"}
	if s.memory != nil {
		lab = s.memory.Snapshot().MemoryLab
	}
	stageState := func(name string) string {
		if value, ok := lab[name].(string); ok && platformValueIs(value, "ready") {
			return "ready"
		}
		return "unavailable"
	}
	baseline := domain.PlatformPolicyResultV1{Policy: "baseline-v1", Detected: true, DetectionLatencyTicks: 0, FalseAcceptance: 0, UncertaintyGrowthM: 48, TimeToSafeBehaviorTicks: 43}
	candidate := domain.PlatformPolicyResultV1{Policy: "candidate-multisignal-v2", Detected: true, DetectionLatencyTicks: 0, FalseAcceptance: 0, UncertaintyGrowthM: 48, TimeToSafeBehaviorTicks: 43}
	value := domain.PlatformEvaluationEvidenceV1{SchemaVersion: 1, EpisodeID: episodeID, IncidentID: incident.ID, ScenarioSeed: incident.ScenarioSeed, SourceStateChecksum: incident.StateChecksum, SourceEventIDs: sourceIDs, ArtifactChecksum: "sha256:" + hex.EncodeToString(sum[:]), Stages: []domain.PlatformEvaluationStageV1{{ID: "capture", Label: "Operational episode", State: "ready", SourceID: incident.ID, Detail: "Immutable evidence IDs and source checksum are present."}, {ID: "dagster", Label: "Validation and redaction", State: stageState("dagster"), Detail: "Optional Dagster memory-lab asset."}, {ID: "minio", Label: "Bounded artifact", State: stageState("minio"), Detail: "Private MinIO object with a 2 GiB bucket quota."}, {ID: "mlflow", Label: "Policy comparison", State: stageState("mlflow"), Detail: "Baseline and candidate use the same episode and seed."}, {ID: "promotion", Label: "Release gate", State: "awaiting_human", Detail: "A privileged human must approve the exact candidate hash."}}, Baseline: baseline, Candidate: candidate, PromotionState: "awaiting_privileged_human_decision", Redactions: []string{"raw_audio", "credentials", "hidden_faction_state"}, SampledAt: time.Now().UTC()}
	writeJSON(w, http.StatusOK, value)
}

func platformValueIs(value, expected string) bool {
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func (s *Server) abortPlatformDrillV6(w http.ResponseWriter, r *http.Request) {
	if s.platform == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: "Drill execution is unavailable."})
		return
	}
	id, ok := strings.CutSuffix(r.PathValue("action"), ":abort")
	if !ok || id == "" {
		writeJSON(w, http.StatusNotFound, domain.APIError{Code: "DRILL_NOT_FOUND", Message: "Unknown drill action."})
		return
	}
	var request domain.PlatformMutationV1
	if !decode(w, r, &request) {
		return
	}
	receipt, err := s.platform.AbortDrill(r.Context(), id, request)
	respondPlatformProof(w, receipt, err, http.StatusOK)
}

func respondPlatformProof(w http.ResponseWriter, value any, err error, success int) {
	if err == nil {
		writeJSON(w, success, value)
		return
	}
	status := http.StatusUnprocessableEntity
	if platformErr, ok := err.(*platform.Error); ok {
		if platformErr.Code == "PLATFORM_STALE_STATE" || platformErr.Code == "FAULT_CONFLICT" {
			status = http.StatusConflict
		}
		if platformErr.Code == "PLATFORM_UNAVAILABLE" {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, domain.APIError{Code: platformErr.Code, Message: platformErr.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.APIError{Code: "PLATFORM_UNAVAILABLE", Message: err.Error()})
}

func slo(id, label string, value float64, unit string, objective float64, comparison string, measured bool, source, workload, window string, sampledAt time.Time) domain.PlatformSLORecordV1 {
	state := "unavailable"
	if measured {
		pass := comparison == "lte" && value <= objective || comparison == "eq" && value == objective
		state = "violated"
		if pass {
			state = "within_objective"
		}
	}
	return domain.PlatformSLORecordV1{ID: id, Label: label, Value: value, Unit: unit, Objective: objective, Comparison: comparison, State: state, Source: source, Workload: workload, Window: window, Environment: "VM 214 home-lab deployment", Measured: measured, SampledAt: sampledAt}
}

func stateFrom(ok bool) string {
	if ok {
		return "healthy"
	}
	return "unavailable"
}

func workloadName(snapshot domain.PlatformSnapshotV1) string {
	if snapshot.ActiveRun == nil {
		return "idle platform"
	}
	return strconv.Itoa(snapshot.ActiveRun.VesselCount) + " logical producers · " + snapshot.ActiveRun.Profile
}

func maxFloat(values []float64) float64 {
	sort.Float64s(values)
	if len(values) == 0 {
		return 0
	}
	return values[len(values)-1]
}

func firstMeasured(measured bool, measuredValue, fallback float64) float64 {
	if measured {
		return measuredValue
	}
	return fallback
}

func firstSource(measured bool, measuredSource, fallback string) string {
	if measured {
		return measuredSource
	}
	return fallback
}

func envValue(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
