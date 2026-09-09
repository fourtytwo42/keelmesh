package observability

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxTracePayload = 2 << 20

type CollectorConfig struct {
	Address     string
	DatabaseURL string
	TokenFile   string
	Retention   time.Duration
}

func RunCollector(ctx context.Context, cfg CollectorConfig, logger *slog.Logger) error {
	if cfg.Address == "" {
		cfg.Address = ":4318"
	}
	if cfg.Retention <= 0 {
		cfg.Retention = 24 * time.Hour
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect trace store: %w", err)
	}
	defer pool.Close()
	collector := &collectorHandler{pool: pool, tokenFile: cfg.TokenFile, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", collector.health)
	mux.HandleFunc("POST /v1/traces", collector.traces)
	server := &http.Server{Addr: cfg.Address, Handler: mux, ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = pool.Exec(context.Background(), `DELETE FROM otel_spans WHERE started_at < now() - make_interval(secs => $1)`, int64(cfg.Retention.Seconds()))
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	logger.Info("private OTLP/HTTP collector listening", "address", cfg.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type collectorHandler struct {
	pool      *pgxpool.Pool
	tokenFile string
	logger    *slog.Logger
}

func (c *collectorHandler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func (c *collectorHandler) traces(w http.ResponseWriter, r *http.Request) {
	if !authorized(r.Header.Get("Authorization"), readSecret(c.tokenFile)) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxTracePayload))
	if err != nil {
		http.Error(w, "trace payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	spans, err := decodeOTLP(body)
	if err != nil {
		http.Error(w, "invalid OTLP payload", http.StatusBadRequest)
		return
	}
	for _, span := range spans {
		attributes, _ := json.Marshal(span.Attributes)
		_, err = c.pool.Exec(r.Context(), `INSERT INTO otel_spans(trace_id,span_id,parent_span_id,service,name,state,started_at,duration_ms,attributes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(trace_id,span_id) DO NOTHING`, span.TraceID, span.SpanID, span.ParentSpanID, span.Service, span.Name, span.State, span.StartedAt, span.EndedAt.Sub(span.StartedAt).Seconds()*1000, attributes)
		if err != nil {
			c.logger.Error("persist OTLP span", "error", err)
			http.Error(w, "trace store unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{}`)
}

func authorized(header, token string) bool {
	provided := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || len(provided) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

type otlpPayload struct {
	ResourceSpans []struct {
		Resource struct {
			Attributes []otlpAttribute `json:"attributes"`
		} `json:"resource"`
		ScopeSpans []struct {
			Spans []struct {
				TraceID, SpanID, ParentSpanID, Name string
				StartTimeUnixNano                   json.Number
				EndTimeUnixNano                     json.Number
				Attributes                          []otlpAttribute
				Status                              struct {
					Code    int
					Message string
				}
			} `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}

type otlpAttribute struct {
	Key   string `json:"key"`
	Value struct {
		StringValue string `json:"stringValue"`
	} `json:"value"`
}

func decodeOTLP(body []byte) ([]recordedSpan, error) {
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	var payload otlpPayload
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	var result []recordedSpan
	for _, resource := range payload.ResourceSpans {
		service := attributeValue(resource.Resource.Attributes, "service.name")
		for _, scope := range resource.ScopeSpans {
			for _, wire := range scope.Spans {
				start, startErr := wire.StartTimeUnixNano.Int64()
				end, endErr := wire.EndTimeUnixNano.Int64()
				if startErr != nil || endErr != nil || end < start || !validHex(wire.TraceID, 32) || !validHex(wire.SpanID, 16) || service == "" || wire.Name == "" {
					return nil, errors.New("invalid span")
				}
				state := wire.Status.Message
				if state == "" {
					state = "ok"
				}
				attrs := make(map[string]string, len(wire.Attributes))
				for _, attr := range wire.Attributes {
					attrs[attr.Key] = attr.Value.StringValue
				}
				result = append(result, recordedSpan{TraceID: wire.TraceID, SpanID: wire.SpanID, ParentSpanID: wire.ParentSpanID, Name: wire.Name, Service: service, State: state, StartedAt: time.Unix(0, start).UTC(), EndedAt: time.Unix(0, end).UTC(), Attributes: attrs})
			}
		}
	}
	if len(result) == 0 {
		return nil, errors.New("empty trace payload")
	}
	return result, nil
}

func attributeValue(attributes []otlpAttribute, key string) string {
	for _, attribute := range attributes {
		if attribute.Key == key {
			return attribute.Value.StringValue
		}
	}
	return ""
}
