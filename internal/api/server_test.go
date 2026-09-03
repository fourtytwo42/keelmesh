package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/fourtytwo42/keelmesh/internal/core"
)

func TestHandlerRegistersVersionedActionRoutes(t *testing.T) {
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0o644)}}
	server := New(core.New(), slog.Default(), web)
	if handler := server.Handler(); handler == nil {
		t.Fatal("handler is nil")
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
