package api

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/fourtytwo42/keelmesh/internal/core"
	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/fourtytwo42/keelmesh/internal/fleetops"
)

func TestHandlerRegistersVersionedActionRoutes(t *testing.T) {
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	server := New(core.New(), slog.Default(), web)
	if handler := server.Handler(); handler == nil {
		t.Fatal("handler is nil")
	}
}

func TestCombatV8ReadRoutesExposeBlackwake(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	manager := fleetops.New("", slog.Default())
	server := New(core.New(), slog.Default(), web, manager)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v8/combat", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("combat snapshot status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var snapshot domain.CombatSnapshotV1
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entities) != 45 || snapshot.Disclaimer == "" {
		t.Fatalf("unexpected combat snapshot: entities=%d disclaimer=%q", len(snapshot.Entities), snapshot.Disclaimer)
	}
	entity := httptest.NewRecorder()
	server.Handler().ServeHTTP(entity, httptest.NewRequest(http.MethodGet, "/api/v8/combat/entities/HOSTILE-0001", nil))
	if entity.Code != http.StatusOK || !json.Valid(entity.Body.Bytes()) {
		t.Fatalf("Blackwake lookup failed: status=%d body=%s", entity.Code, entity.Body.String())
	}
}

func TestSPAIndexIsNeverCachedAcrossDeployments(t *testing.T) {
	web := fstest.MapFS{
		"index.html":         &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)},
		"assets/app-dead.js": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)},
	}
	index := httptest.NewRecorder()
	spaHandler(web).ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := index.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("index cache policy = %q", got)
	}
	asset := httptest.NewRecorder()
	spaHandler(web).ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/app-dead.js", nil))
	if got := asset.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache policy = %q", got)
	}
}
