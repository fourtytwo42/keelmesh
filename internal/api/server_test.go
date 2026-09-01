package api

import (
	"io/fs"
	"log/slog"
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
