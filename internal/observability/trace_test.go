package observability

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTracerExportsOTLPWithBearerToken(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenPath, []byte("test-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Errorf("authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		received <- body
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	tracer := NewTracer(Config{Endpoint: server.URL, TokenFile: tokenPath, ServiceName: "test-service", QueueSize: 4}, nil)
	ctx := ContextWithParent(context.Background(), "legacy-correlation-id", "")
	_, span := tracer.Start(ctx, "test.operation", map[string]string{"entity.id": "vessel-1"})
	span.End("completed", nil)
	select {
	case body := <-received:
		spans, err := decodeOTLP(body)
		if err != nil {
			t.Fatal(err)
		}
		if spans[0].Service != "test-service" || spans[0].Name != "test.operation" || spans[0].Attributes["entity.id"] != "vessel-1" {
			t.Fatalf("unexpected span: %#v", spans[0])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exported span")
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tracer.Close(closeCtx)
}

func TestMiddlewarePropagatesTraceparentAndRecordsStatus(t *testing.T) {
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
	}))
	defer server.Close()
	tracer := NewTracer(Config{Endpoint: server.URL, ServiceName: "http-test", QueueSize: 4}, nil)
	handler := tracer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if TraceID(r.Context()) != "0123456789abcdef0123456789abcdef" {
			t.Errorf("trace context was not propagated")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || response.Header().Get("traceparent") == "" {
		t.Fatalf("status=%d traceparent=%q", response.Code, response.Header().Get("traceparent"))
	}
	select {
	case body := <-received:
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		spans, err := decodeOTLP(body)
		if err != nil {
			t.Fatal(err)
		}
		if spans[0].TraceID != "0123456789abcdef0123456789abcdef" || spans[0].State != "ok" || spans[0].Attributes["http.response.status_code"] != "201" {
			t.Fatalf("unexpected span: %#v", spans[0])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for middleware span")
	}
}

func TestDecodeOTLPRejectsMalformedAndAuthorizationIsExact(t *testing.T) {
	if _, err := decodeOTLP([]byte(`{"resourceSpans":[]}`)); err == nil {
		t.Fatal("empty payload accepted")
	}
	if authorized("Bearer secret-extra", "secret") || authorized("Bearer secret", "") || !authorized("Bearer secret", "secret") {
		t.Fatal("bearer authorization comparison is incorrect")
	}
	if _, _, ok := ParseTraceparent("00-not-a-trace-not-a-span-01"); ok {
		t.Fatal("invalid traceparent accepted")
	}
}
