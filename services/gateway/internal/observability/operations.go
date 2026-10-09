package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type traceKey struct{}
type Trace struct {
	ID, Span, Parent string
	Started          time.Time
}

func NewTrace(ctx context.Context, parent string) (context.Context, Trace, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ctx, Trace{}, err
	}
	t := Trace{ID: hex.EncodeToString(raw[:16]), Span: hex.EncodeToString(raw[16:]), Started: time.Now()}
	parts := strings.Split(parent, "-")
	if len(parts) == 4 && parts[0] == "00" && validHex(parts[1], 32) && validHex(parts[2], 16) && (parts[3] == "00" || parts[3] == "01") {
		t.ID = parts[1]
		t.Parent = parts[2]
	}
	return context.WithValue(ctx, traceKey{}, t), t, nil
}
func validHex(s string, n int) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(s) == n && s == strings.ToLower(s) && !bytes.Equal(b, make([]byte, n/2))
}
func Current(ctx context.Context) Trace { t, _ := ctx.Value(traceKey{}).(Trace); return t }
func Header(ctx context.Context) string {
	t := Current(ctx)
	if t.ID == "" {
		return ""
	}
	return "00-" + t.ID + "-" + t.Span + "-01"
}

type span struct {
	Trace         Trace
	Route, Method string
	Status        int
	Finished      time.Time
}
type Operations struct {
	logger                                                                  *slog.Logger
	endpoint, version                                                       string
	queue                                                                   chan span
	done                                                                    chan struct{}
	mu                                                                      sync.Mutex
	closed                                                                  bool
	requests                                                                [6]uint64
	buckets                                                                 [6]uint64
	seconds                                                                 float64
	AuditFailures, RateDenied, SignatureDenied, TraceDropped, TraceFailures atomic.Uint64
}

func New(logger *slog.Logger, endpoint, version string) *Operations {
	o := &Operations{logger: logger, endpoint: strings.TrimRight(endpoint, "/"), version: version, queue: make(chan span, 256), done: make(chan struct{})}
	go o.export()
	return o
}
func (o *Operations) Complete(t Trace, route, method string, status int) {
	finished := time.Now()
	elapsed := finished.Sub(t.Started).Seconds()
	o.mu.Lock()
	o.requests[min(max(status/100, 0), 5)]++
	o.seconds += elapsed
	for i, b := range []float64{.005, .01, .05, .1, .5, 1} {
		if elapsed <= b {
			o.buckets[i]++
		}
	}
	o.mu.Unlock()
	if o.logger != nil {
		o.logger.Info("HTTP request completed", "trace_id", t.ID, "span_id", t.Span, "route", route, "method", method, "status", status, "duration_ms", elapsed*1000, "version", o.version)
	}
	if o.endpoint != "" {
		o.mu.Lock()
		if o.closed {
			o.TraceDropped.Add(1)
		} else {
			select {
			case o.queue <- span{t, route, method, status, finished}:
			default:
				o.TraceDropped.Add(1)
			}
		}
		o.mu.Unlock()
	}
}
func (o *Operations) Metrics(w io.Writer) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var total uint64
	for i, n := range o.requests {
		total += n
		fmt.Fprintf(w, "sentinel_http_requests_total{status_class=%q} %d\n", fmt.Sprintf("%dxx", i), n)
	}
	for i, b := range []string{"0.005", "0.01", "0.05", "0.1", "0.5", "1"} {
		fmt.Fprintf(w, "sentinel_http_duration_seconds_bucket{le=%q} %d\n", b, o.buckets[i])
	}
	fmt.Fprintf(w, "sentinel_http_duration_seconds_bucket{le=\"+Inf\"} %d\nsentinel_http_duration_seconds_count %d\nsentinel_http_duration_seconds_sum %g\n", total, total, o.seconds)
	fmt.Fprintf(w, "sentinel_rate_denied_total %d\nsentinel_audit_failures_total %d\nsentinel_signature_denied_total %d\nsentinel_trace_dropped_total %d\nsentinel_trace_failures_total %d\nsentinel_trace_queue_depth %d\n", o.RateDenied.Load(), o.AuditFailures.Load(), o.SignatureDenied.Load(), o.TraceDropped.Load(), o.TraceFailures.Load(), len(o.queue))
}
func (o *Operations) Close(ctx context.Context) error {
	o.mu.Lock()
	if !o.closed {
		o.closed = true
		close(o.queue)
	}
	o.mu.Unlock()
	select {
	case <-o.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func attribute(key, value string) map[string]any {
	return map[string]any{"key": key, "value": map[string]string{"stringValue": value}}
}
func (o *Operations) export() {
	defer close(o.done)
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	for s := range o.queue {
		data, err := json.Marshal(map[string]any{"resourceSpans": []any{map[string]any{
			"resource": map[string]any{"attributes": []any{attribute("service.name", "sentinel-gateway"), attribute("service.version", o.version)}},
			"scopeSpans": []any{map[string]any{"scope": map[string]string{"name": "sentinel.http"}, "spans": []any{map[string]any{
				"traceId": s.Trace.ID, "spanId": s.Trace.Span, "parentSpanId": s.Trace.Parent, "name": s.Route, "kind": 2,
				"startTimeUnixNano": fmt.Sprint(s.Trace.Started.UnixNano()), "endTimeUnixNano": fmt.Sprint(s.Finished.UnixNano()),
				"attributes": []any{attribute("http.request.method", s.Method), map[string]any{"key": "http.response.status_code", "value": map[string]string{"intValue": fmt.Sprint(s.Status)}}},
				"status": map[string]int{"code": func() int {
					if s.Status >= 500 {
						return 2
					}
					return 0
				}()},
			}}}},
		}}})
		if err != nil {
			o.TraceFailures.Add(1)
			continue
		}
		req, err := http.NewRequest(http.MethodPost, o.endpoint+"/v1/traces", bytes.NewReader(data))
		if err != nil {
			o.TraceFailures.Add(1)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			o.TraceFailures.Add(1)
			continue
		}
		// Bound the receiver's response too; export failures must never block request handling.
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
		response.Body.Close()
		var result struct {
			PartialSuccess struct {
				RejectedSpans string `json:"rejectedSpans"`
			} `json:"partialSuccess"`
		}
		if response.StatusCode != 200 || readErr != nil || len(body) > 65536 || (len(body) > 0 && json.Unmarshal(body, &result) != nil) || (result.PartialSuccess.RejectedSpans != "" && result.PartialSuccess.RejectedSpans != "0") {
			o.TraceFailures.Add(1)
		}
	}
}
