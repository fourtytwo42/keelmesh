package observability

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(value []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(value)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *statusWriter) Push(target string, options *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, options)
	}
	return http.ErrNotSupported
}

func (t *Tracer) Middleware(next http.Handler) http.Handler {
	if !t.Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if traceID, parentID, ok := ParseTraceparent(r.Header.Get("traceparent")); ok {
			ctx = ContextWithParent(ctx, traceID, parentID)
		}
		ctx, span := t.Start(ctx, "http "+r.Method+" "+routeName(r), map[string]string{
			"http.request.method": r.Method,
			"url.path":            r.URL.Path,
		})
		writer := &statusWriter{ResponseWriter: w}
		writer.Header().Set("traceparent", "00-"+span.traceID+"-"+span.spanID+"-01")
		next.ServeHTTP(writer, r.WithContext(ctx))
		if writer.status == 0 {
			writer.status = http.StatusOK
		}
		state := "ok"
		if writer.status >= 500 {
			state = "error"
		}
		span.End(state, map[string]string{"http.response.status_code": strconv.Itoa(writer.status)})
	})
}

func ParseTraceparent(value string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" || !validHex(parts[1], 32) || !validHex(parts[2], 16) || len(parts[3]) != 2 {
		return "", "", false
	}
	return strings.ToLower(parts[1]), strings.ToLower(parts[2]), true
}

func routeName(r *http.Request) string {
	if pattern := r.Pattern; pattern != "" {
		return pattern
	}
	return r.URL.Path
}
