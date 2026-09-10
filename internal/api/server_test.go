package api

import (
	"bytes"
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

func TestCombatV8ArmAndDisarmControlledVessel(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	manager := fleetops.New("", slog.Default())
	server := New(core.New(), slog.Default(), web, manager)
	vesselID := manager.Snapshot().Vessels[0].ID

	for _, action := range []struct {
		suffix string
		armed  bool
	}{
		{suffix: ":arm", armed: true},
		{suffix: ":disarm", armed: false},
	} {
		body := []byte(`{"request_id":"test-` + action.suffix[1:] + `","idempotency_key":"key-` + action.suffix[1:] + `","actor_identity":"operator"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v8/combat/vessels/"+vesselID+action.suffix, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body = %s", action.suffix, recorder.Code, recorder.Body.String())
		}
		var entity domain.CombatEntityStateV1
		if err := json.Unmarshal(recorder.Body.Bytes(), &entity); err != nil {
			t.Fatal(err)
		}
		if entity.Armed != action.armed {
			t.Fatalf("%s armed = %t", action.suffix, entity.Armed)
		}
	}
}

func TestCombatV8AutoDefenseDefaultsOffAndCanBeEnabled(t *testing.T) {
	t.Setenv("KEELMESH_FLEET_PROFILE", "vm12")
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	manager := fleetops.New("", slog.Default())
	server := New(core.New(), slog.Default(), web, manager)
	vesselID := manager.Snapshot().Vessels[0].ID
	before, _ := manager.CombatEntity(vesselID)
	if before.AutoDefense {
		t.Fatal("auto defense must default disabled")
	}
	body := []byte(`{"request_id":"defense","idempotency_key":"defense-key","actor_identity":"operator","enabled":true,"response":"retreat"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v8/combat/vessels/"+vesselID+":defense", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("defense status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var entity domain.CombatEntityStateV1
	if err := json.Unmarshal(recorder.Body.Bytes(), &entity); err != nil {
		t.Fatal(err)
	}
	if !entity.AutoDefense || entity.AutoDefenseResponse != "retreat" {
		t.Fatalf("unexpected defense state: %#v", entity)
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
