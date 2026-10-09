package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTraceContextValidation(t *testing.T) {
	parent := "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01"
	ctx, tr, err := NewTrace(context.Background(), parent)
	if err != nil || tr.ID != "0123456789abcdef0123456789abcdef" || tr.Parent != "0123456789abcdef" || Header(ctx) == parent {
		t.Fatal("parent propagation failed")
	}
	for _, bad := range []string{"", "00-00000000000000000000000000000000-0123456789abcdef-01", strings.ToUpper(parent), parent + "-extra"} {
		_, tr, err := NewTrace(context.Background(), bad)
		if err != nil || tr.Parent != "" || len(tr.ID) != 32 {
			t.Fatal("invalid parent accepted", bad)
		}
	}
}
func TestExportAndMetrics(t *testing.T) {
	received := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid export contract")
		}
		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Error(err)
		}
		received <- data
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	ops := New(nil, server.URL, "test-version")
	_, tr, _ := NewTrace(context.Background(), "")
	ops.Complete(tr, "GET /api/v1/events", "GET", 200)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ops.Close(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-received:
		if len(data["resourceSpans"].([]any)) != 1 {
			t.Fatal("missing span")
		}
	default:
		t.Fatal("export not delivered")
	}
	var out bytes.Buffer
	ops.Metrics(&out)
	if !strings.Contains(out.String(), `sentinel_http_requests_total{status_class="2xx"} 1`) || ops.TraceFailures.Load() != 0 {
		t.Fatal(out.String())
	}
}
func TestExporterFailureIsObservable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", 503) }))
	defer server.Close()
	ops := New(nil, server.URL, "test")
	_, tr, _ := NewTrace(context.Background(), "")
	ops.Complete(tr, "unmatched", "GET", 503)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ops.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if ops.TraceFailures.Load() != 1 {
		t.Fatal("export failure hidden")
	}
}

func TestShutdownConcurrentWithCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer server.Close()
	ops := New(nil, server.URL, "test")
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			_, tr, _ := NewTrace(context.Background(), "")
			ops.Complete(tr, "unmatched", "GET", 200)
		}
	})
	if err := ops.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if err := ops.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
