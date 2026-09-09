package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Endpoint    string
	TokenFile   string
	ServiceName string
	QueueSize   int
}

func ConfigFromEnv(service string) Config {
	return Config{
		Endpoint:    strings.TrimSpace(os.Getenv("KEELMESH_OTLP_ENDPOINT")),
		TokenFile:   strings.TrimSpace(os.Getenv("KEELMESH_OTLP_TOKEN_FILE")),
		ServiceName: service,
		QueueSize:   512,
	}
}

type traceContext struct{ TraceID, SpanID string }
type contextKey struct{}

type recordedSpan struct {
	TraceID, SpanID, ParentSpanID, Name, Service, State string
	StartedAt, EndedAt                                  time.Time
	Attributes                                          map[string]string
}

type Span struct {
	tracer                    *Tracer
	traceID, spanID, parentID string
	name                      string
	startedAt                 time.Time
	attributes                map[string]string
	once                      sync.Once
}

type Tracer struct {
	cfg     Config
	logger  *slog.Logger
	client  *http.Client
	queue   chan recordedSpan
	dropped atomic.Int64
	closed  chan struct{}
	wg      sync.WaitGroup
}

func NewTracer(cfg Config, logger *slog.Logger) *Tracer {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 512
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		cfg.ServiceName = "keelmesh"
	}
	t := &Tracer{cfg: cfg, logger: logger, client: &http.Client{Timeout: 2 * time.Second}, queue: make(chan recordedSpan, cfg.QueueSize), closed: make(chan struct{})}
	if cfg.Endpoint != "" {
		t.wg.Add(1)
		go t.exportLoop()
	}
	return t
}

func (t *Tracer) Enabled() bool { return t != nil && t.cfg.Endpoint != "" }

func (t *Tracer) Dropped() int64 {
	if t == nil {
		return 0
	}
	return t.dropped.Load()
}

func (t *Tracer) Start(ctx context.Context, name string, attributes map[string]string) (context.Context, *Span) {
	parent, _ := ctx.Value(contextKey{}).(traceContext)
	traceID := parent.TraceID
	if traceID == "" {
		traceID = randomHex(16)
	}
	spanID := randomHex(8)
	span := &Span{tracer: t, traceID: traceID, spanID: spanID, parentID: parent.SpanID, name: name, startedAt: time.Now().UTC(), attributes: cloneAttributes(attributes)}
	return context.WithValue(ctx, contextKey{}, traceContext{TraceID: traceID, SpanID: spanID}), span
}

func ContextWithParent(ctx context.Context, traceID, spanID string) context.Context {
	if !validHex(traceID, 32) {
		traceID = NormalizeTraceID(traceID)
	}
	if !validHex(spanID, 16) {
		spanID = ""
	}
	return context.WithValue(ctx, contextKey{}, traceContext{TraceID: traceID, SpanID: spanID})
}

// NormalizeTraceID preserves valid W3C trace IDs and deterministically maps
// legacy correlation identifiers into a valid 16-byte trace ID.
func NormalizeTraceID(value string) string {
	if validHex(value, 32) {
		return strings.ToLower(value)
	}
	return deterministicTraceID(value)
}

func TraceID(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(traceContext)
	return value.TraceID
}

func (s *Span) TraceID() string {
	if s == nil {
		return ""
	}
	return s.traceID
}

func (s *Span) Traceparent() string {
	if s == nil {
		return ""
	}
	return "00-" + s.traceID + "-" + s.spanID + "-01"
}

func (s *Span) End(state string, attributes map[string]string) {
	if s == nil || s.tracer == nil {
		return
	}
	s.once.Do(func() {
		merged := cloneAttributes(s.attributes)
		for key, value := range attributes {
			merged[key] = value
		}
		if state == "" {
			state = "ok"
		}
		s.tracer.enqueue(recordedSpan{TraceID: s.traceID, SpanID: s.spanID, ParentSpanID: s.parentID, Name: s.name, Service: s.tracer.cfg.ServiceName, State: state, StartedAt: s.startedAt, EndedAt: time.Now().UTC(), Attributes: merged})
	})
}

func (t *Tracer) enqueue(span recordedSpan) {
	if !t.Enabled() {
		return
	}
	select {
	case t.queue <- span:
	default:
		t.dropped.Add(1)
	}
}

func (t *Tracer) Close(ctx context.Context) {
	if !t.Enabled() {
		return
	}
	select {
	case <-t.closed:
		return
	default:
		close(t.closed)
	}
	done := make(chan struct{})
	go func() { t.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (t *Tracer) exportLoop() {
	defer t.wg.Done()
	for {
		select {
		case span := <-t.queue:
			t.export(span)
		case <-t.closed:
			for {
				select {
				case span := <-t.queue:
					t.export(span)
				default:
					return
				}
			}
		}
	}
}

func (t *Tracer) export(span recordedSpan) {
	body, err := json.Marshal(makeOTLPPayload(span))
	if err != nil {
		t.dropped.Add(1)
		return
	}
	req, err := http.NewRequest(http.MethodPost, t.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		t.dropped.Add(1)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token := readSecret(t.cfg.TokenFile); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		t.dropped.Add(1)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		t.dropped.Add(1)
		if t.logger != nil {
			t.logger.Warn("OTLP span export rejected", "status", resp.StatusCode, "service", t.cfg.ServiceName)
		}
	}
}

func makeOTLPPayload(span recordedSpan) map[string]any {
	attributes := make([]map[string]any, 0, len(span.Attributes))
	for key, value := range span.Attributes {
		attributes = append(attributes, map[string]any{"key": key, "value": map[string]any{"stringValue": value}})
	}
	statusCode := 1
	if span.State != "ok" && span.State != "accepted" && span.State != "completed" {
		statusCode = 2
	}
	wireSpan := map[string]any{
		"traceId": span.TraceID, "spanId": span.SpanID, "name": span.Name, "kind": 2,
		"startTimeUnixNano": span.StartedAt.UnixNano(), "endTimeUnixNano": span.EndedAt.UnixNano(),
		"attributes": attributes, "status": map[string]any{"code": statusCode, "message": span.State},
	}
	if span.ParentSpanID != "" {
		wireSpan["parentSpanId"] = span.ParentSpanID
	}
	return map[string]any{"resourceSpans": []any{map[string]any{
		"resource":   map[string]any{"attributes": []any{map[string]any{"key": "service.name", "value": map[string]any{"stringValue": span.Service}}}},
		"scopeSpans": []any{map[string]any{"scope": map[string]any{"name": "keelmesh"}, "spans": []any{wireSpan}}},
	}}}
}

func randomHex(size int) string {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return deterministicTraceID(time.Now().UTC().String())[:size*2]
	}
	return hex.EncodeToString(value)
}

func deterministicTraceID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

func validHex(value string, length int) bool {
	if len(value) != length || strings.Trim(value, "0") == "" {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func cloneAttributes(value map[string]string) map[string]string {
	out := make(map[string]string, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func readSecret(path string) string {
	if path == "" {
		return ""
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(value))
}
