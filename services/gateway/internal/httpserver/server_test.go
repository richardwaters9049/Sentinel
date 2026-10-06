package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type stubIngestor struct {
	event telemetry.Event
	err   error
}

func (s stubIngestor) Ingest(context.Context, telemetry.IngestRequest) (telemetry.Event, error) {
	return s.event, s.err
}

type stubEventReader struct {
	events []telemetry.Event
	err    error
	query  database.EventQuery
}

func (s *stubEventReader) ListEvents(_ context.Context, query database.EventQuery) ([]telemetry.Event, error) {
	s.query = query
	return s.events, s.err
}

func TestHealth(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	if !strings.Contains(res.Body.String(), "\"service\":\"gateway\"") {
		t.Fatalf("expected gateway service in response, got %s", res.Body.String())
	}
}

func TestReadyReturnsOKWhenDependenciesPass(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if !strings.Contains(res.Body.String(), "\"ready\":true") {
		t.Fatalf("expected ready=true, got %s", res.Body.String())
	}
}

func TestReadyReturnsUnavailableWhenDependencyFails(t *testing.T) {
	t.Parallel()

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return errors.New("down") },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}

	if !strings.Contains(res.Body.String(), "\"ready\":false") {
		t.Fatalf("expected ready=false, got %s", res.Body.String())
	}
}

func TestTelemetryAcceptsValidEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 21, 0, 0, 0, time.UTC)
	ingestor := stubIngestor{
		event: telemetry.Event{
			EventID:       "evt_test_001",
			SchemaVersion: telemetry.SchemaVersion,
			ReceivedAt:    now,
		},
	}

	body := `{
		"timestamp": "2026-10-06T20:59:00Z",
		"source": {"type": "identity", "collector": "test"},
		"event": {"category": "authentication", "action": "login", "outcome": "success"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", strings.NewReader(body))
	res := httptest.NewRecorder()

	newTestServer(ingestor, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, res.Code, res.Body.String())
	}

	if !strings.Contains(res.Body.String(), "\"event_id\":\"evt_test_001\"") {
		t.Fatalf("expected event id in response, got %s", res.Body.String())
	}
}

func TestTelemetryRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	body := `{
		"timestamp": "2026-10-06T20:59:00Z",
		"source": {"type": "identity", "collector": "test"},
		"event": {"category": "authentication", "action": "login"},
		"unexpected": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", strings.NewReader(body))
	res := httptest.NewRecorder()

	newTestServer(stubIngestor{}, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestTelemetryReturnsValidationFailure(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", strings.NewReader(`{
		"timestamp": "2026-10-06T20:59:00Z",
		"source": {"type": "identity", "collector": "test"},
		"event": {"category": "authentication", "action": "login"}
	}`))
	res := httptest.NewRecorder()

	ingestor := stubIngestor{
		err: telemetry.ErrInvalidEvent,
	}

	newTestServer(ingestor, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	if !strings.Contains(res.Body.String(), "invalid_event") {
		t.Fatalf("expected invalid_event response, got %s", res.Body.String())
	}
}

func TestEventsReturnsStoredEvents(t *testing.T) {
	t.Parallel()

	reader := &stubEventReader{
		events: []telemetry.Event{
			{
				EventID:       "evt_test_001",
				SchemaVersion: telemetry.SchemaVersion,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=25&category=authentication&asset_id=asset-01", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	if reader.query.Limit != 25 {
		t.Fatalf("expected limit 25, got %d", reader.query.Limit)
	}
	if reader.query.Category != "authentication" {
		t.Fatalf("expected category filter, got %q", reader.query.Category)
	}
	if reader.query.AssetID != "asset-01" {
		t.Fatalf("expected asset filter, got %q", reader.query.AssetID)
	}
}

func TestEventsRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=999", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, &stubEventReader{}).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, nil).Handler().ServeHTTP(res, req)

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected DENY frame policy, got %q", got)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	res := httptest.NewRecorder()

	newTestServer(nil, nil).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}
}

func newTestServer(
	ingestor TelemetryIngestor,
	events EventReader,
) *Server {
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	return New(checker, ingestor, events)
}
