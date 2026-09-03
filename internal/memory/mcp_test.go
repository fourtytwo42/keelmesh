package memory

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryMCPRejectsMissingToken(t *testing.T) {
	manager := testManager()
	request := httptest.NewRequest(http.MethodPost, "/mcp/memory", nil)
	response := httptest.NewRecorder()
	manager.MCPHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if response.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("missing bearer challenge")
	}
}
