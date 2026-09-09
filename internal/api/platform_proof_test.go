package api

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/fourtytwo42/keelmesh/internal/agent"
	"github.com/fourtytwo42/keelmesh/internal/core"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func proofTestServer(managers ...any) *Server {
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	return New(core.New(), slog.Default(), web, managers...)
}

func TestPlatformSummaryLabelsUnavailableEvidence(t *testing.T) {
	server := proofTestServer()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v6/platform/summary", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var summary domain.PlatformProofSummaryV1
	if err := json.Unmarshal(recorder.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Planes) != 4 || summary.Planes[2].Measured || summary.Planes[2].State != "unavailable" {
		t.Fatalf("unavailable data-plane evidence was overstated: %+v", summary.Planes)
	}
	if summary.SLOs[2].Measured || summary.SLOs[2].State != "unavailable" {
		t.Fatalf("duplicate-effect SLO must remain unmeasured without a drill: %+v", summary.SLOs[2])
	}
}

func TestPlatformEvaluationIsSourceLinkedAndHumanGated(t *testing.T) {
	agentManager := agent.NewManager(agent.Config{Commit: "test-commit"}, slog.Default())
	server := proofTestServer(agentManager)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v6/platform/evaluations/episode-incident-vessel4-resilient-edge", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var evidence domain.PlatformEvaluationEvidenceV1
	if err := json.Unmarshal(recorder.Body.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if len(evidence.SourceEventIDs) == 0 || evidence.SourceStateChecksum == "" || evidence.ArtifactChecksum == "" {
		t.Fatalf("evaluation lacks source linkage: %+v", evidence)
	}
	if evidence.PromotionState != "awaiting_privileged_human_decision" || evidence.Stages[len(evidence.Stages)-1].State != "awaiting_human" {
		t.Fatalf("evaluation promotion boundary is not human-gated: %+v", evidence)
	}
}

func TestPlatformCapacityFailsClosedWithoutManager(t *testing.T) {
	server := proofTestServer()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v6/platform/capacity", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestCoordinationBarrierIsDisabledByDefault(t *testing.T) {
	t.Setenv("KEELMESH_PLATFORM_DRILLS_ENABLED", "")
	server := proofTestServer()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v6/coordination/cells/A/barriers", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
