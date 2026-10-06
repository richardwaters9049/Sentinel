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
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/hunting"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/investigation"
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

type stubFindingReader struct {
	findings         []database.FindingRecord
	detail           database.FindingDetail
	evidence         database.FindingEvidenceContext
	detections       []database.DetectionRecord
	metrics          []database.DetectionMetrics
	err              error
	query            database.FindingQuery
	statusID         string
	status           string
	actorID          string
	requestID        string
	detectionID      string
	detectionEnabled bool
	evidenceID       string
	contextMinutes   int
}

func (s *stubFindingReader) ListFindings(_ context.Context, query database.FindingQuery) ([]database.FindingRecord, error) {
	s.query = query
	return s.findings, s.err
}

func (s *stubFindingReader) GetFindingEvidence(_ context.Context, id string, contextMinutes int) (database.FindingEvidenceContext, error) {
	s.evidenceID = id
	s.contextMinutes = contextMinutes
	if s.err != nil {
		return database.FindingEvidenceContext{}, s.err
	}
	result := s.evidence
	result.FindingID = id
	result.ContextMinutes = contextMinutes
	return result, nil
}

func (s *stubFindingReader) GetFinding(_ context.Context, id string) (database.FindingDetail, error) {
	if s.err != nil {
		return database.FindingDetail{}, s.err
	}
	detail := s.detail
	if detail.ID == "" {
		detail.ID = id
	}
	return detail, nil
}

func (s *stubFindingReader) UpdateFindingStatus(
	_ context.Context,
	id string,
	status string,
	actorID string,
	requestID string,
) (database.FindingDetail, error) {
	s.statusID = id
	s.status = status
	s.actorID = actorID
	s.requestID = requestID
	if s.err != nil {
		return database.FindingDetail{}, s.err
	}
	detail := s.detail
	detail.ID = id
	detail.Status = status
	return detail, nil
}

func (s *stubFindingReader) ListDetections(context.Context) ([]database.DetectionRecord, error) {
	return s.detections, s.err
}

func (s *stubFindingReader) DetectionMetrics(context.Context) ([]database.DetectionMetrics, error) {
	return s.metrics, s.err
}

func (s *stubFindingReader) SetDetectionEnabled(
	_ context.Context,
	id string,
	enabled bool,
	actorID string,
	requestID string,
) (database.DetectionRecord, error) {
	s.detectionID = id
	s.detectionEnabled = enabled
	s.actorID = actorID
	s.requestID = requestID
	if s.err != nil {
		return database.DetectionRecord{}, s.err
	}
	return database.DetectionRecord{ID: id, Enabled: enabled}, nil
}

func (s *stubFindingReader) CreateHunt(
	context.Context,
	string,
	string,
	string,
	hunting.Query,
	string,
) (hunting.Definition, error) {
	return hunting.Definition{}, s.err
}

func (s *stubFindingReader) ListHunts(context.Context) ([]hunting.Definition, error) {
	return []hunting.Definition{}, s.err
}

func (s *stubFindingReader) GetHunt(context.Context, string) (hunting.Definition, error) {
	return hunting.Definition{}, s.err
}

func (s *stubFindingReader) RunHunt(context.Context, string, string, hunting.Query) (hunting.RunResult, error) {
	return hunting.RunResult{}, s.err
}

func (s *stubFindingReader) CreateInvestigation(
	context.Context,
	database.InvestigationCreateInput,
) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) ListInvestigations(
	context.Context,
	string,
	int,
) ([]investigation.Record, error) {
	return []investigation.Record{}, s.err
}

func (s *stubFindingReader) GetInvestigation(context.Context, string) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) AddInvestigationNote(
	context.Context,
	string,
	string,
	string,
	string,
) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) UpdateInvestigationStatus(
	context.Context,
	string,
	string,
	string,
	string,
) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) ListHuntRuns(
	context.Context,
	string,
	int,
) ([]hunting.RunRecord, error) {
	return []hunting.RunRecord{}, s.err
}

func (s *stubFindingReader) HuntRunEvents(context.Context, int64) ([]database.EventEvidenceRecord, error) {
	return []database.EventEvidenceRecord{}, s.err
}

func (s *stubFindingReader) GetAssetPivot(
	context.Context,
	string,
	int,
) (database.AssetPivot, error) {
	return database.AssetPivot{}, s.err
}

func (s *stubFindingReader) GetIdentityPivot(
	context.Context,
	string,
	int,
) (database.IdentityPivot, error) {
	return database.IdentityPivot{}, s.err
}

func (s *stubFindingReader) AttachHuntRun(
	context.Context,
	string,
	int64,
	string,
	string,
) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) UpdateInvestigationMetadata(
	context.Context,
	string,
	*string,
	*string,
	string,
	string,
) (database.InvestigationDetail, error) {
	return database.InvestigationDetail{}, s.err
}

func (s *stubFindingReader) UpdateHunt(
	context.Context,
	string,
	string,
	string,
	string,
	hunting.Query,
	string,
) (hunting.Definition, error) {
	return hunting.Definition{}, s.err
}

func (s *stubFindingReader) ListHuntVersions(
	context.Context,
	string,
) ([]hunting.VersionRecord, error) {
	return []hunting.VersionRecord{}, s.err
}

func (s *stubFindingReader) HuntMetrics(context.Context) (database.HuntMetricsSummary, error) {
	return database.HuntMetricsSummary{}, s.err
}

func (s *stubFindingReader) InvestigationMetrics(context.Context) (database.InvestigationMetricsSummary, error) {
	return database.InvestigationMetricsSummary{}, s.err
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

	New(checker, nil, nil, nil).Handler().ServeHTTP(res, req)

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

func TestFindingsReturnsStoredFindings(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		findings: []database.FindingRecord{
			{
				ID:          "fnd_test_001",
				DetectionID: "DET-AUTH-001",
				Severity:    "high",
				Status:      "new",
			},
		},
	}

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings?limit=10&severity=high&status=new", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if reader.query.Limit != 10 || reader.query.Severity != "high" || reader.query.Status != "new" {
		t.Fatalf("unexpected finding query: %#v", reader.query)
	}
	if !strings.Contains(res.Body.String(), "DET-AUTH-001") {
		t.Fatalf("expected finding in response, got %s", res.Body.String())
	}
}

func TestFindingsRejectsInvalidSeverity(t *testing.T) {
	t.Parallel()

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings?severity=extreme", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, &stubFindingReader{}).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestFindingDetailReturnsEvidenceAndAudit(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		detail: database.FindingDetail{
			FindingRecord: database.FindingRecord{
				ID:          "fnd_test_001",
				DetectionID: "DET-AUTH-001",
				Status:      "new",
			},
			Events: []database.EventEvidenceRecord{{ID: "evt_1"}},
			Audit:  []database.AuditRecord{{Action: "finding.status_changed"}},
		},
	}

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings/fnd_test_001", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "\"evt_1\"") {
		t.Fatalf("expected evidence event in response, got %s", res.Body.String())
	}
}

func TestFindingStatusRequiresActor(t *testing.T) {
	t.Parallel()

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/findings/fnd_test_001/status",
		strings.NewReader(`{"status":"triaged"}`),
	)
	res := httptest.NewRecorder()

	New(checker, nil, nil, &stubFindingReader{}).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
	if !strings.Contains(res.Body.String(), "actor_required") {
		t.Fatalf("expected actor_required response, got %s", res.Body.String())
	}
}

func TestFindingStatusUpdatesAuditedWorkflow(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/findings/fnd_test_001/status",
		strings.NewReader(`{"status":"triaged"}`),
	)
	req.Header.Set("X-Sentinel-Actor", "analyst-richard")
	req.Header.Set("X-Request-ID", "req-001")
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if reader.statusID != "fnd_test_001" || reader.status != "triaged" {
		t.Fatalf("unexpected status update: %q %q", reader.statusID, reader.status)
	}
	if reader.actorID != "analyst-richard" || reader.requestID != "req-001" {
		t.Fatalf("audit context was not forwarded")
	}
}

func TestFindingStatusMapsInvalidTransitionToConflict(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		err: database.ErrInvalidTransition,
	}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/findings/fnd_test_001/status",
		strings.NewReader(`{"status":"confirmed"}`),
	)
	req.Header.Set("X-Sentinel-Actor", "analyst-richard")
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, res.Code)
	}
}

func TestDetectionsReturnsDetectionMetadata(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		detections: []database.DetectionRecord{
			{ID: "DET-AUTH-001", Enabled: true, Severity: "high"},
		},
	}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/detections", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if !strings.Contains(res.Body.String(), "DET-AUTH-001") {
		t.Fatalf("expected detection metadata, got %s", res.Body.String())
	}
}

func TestDetectionStateUpdateRequiresActorAndBoolean(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/detections/DET-AUTH-001",
		strings.NewReader(`{"enabled":false}`),
	)
	req.Header.Set("X-Sentinel-Actor", "analyst-richard")
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if reader.detectionID != "DET-AUTH-001" || reader.detectionEnabled {
		t.Fatalf("expected DET-AUTH-001 to be disabled")
	}
}

func TestFindingEvidenceReturnsLinkedAndContextEvents(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		evidence: database.FindingEvidenceContext{
			LinkedEvents:  []database.EventEvidenceRecord{{ID: "evt_linked"}},
			ContextEvents: []database.EventEvidenceRecord{{ID: "evt_context"}},
		},
	}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings/fnd_test/evidence?context_minutes=10", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if reader.evidenceID != "fnd_test" || reader.contextMinutes != 10 {
		t.Fatalf("unexpected evidence query: %q %d", reader.evidenceID, reader.contextMinutes)
	}
	if !strings.Contains(res.Body.String(), "evt_linked") || !strings.Contains(res.Body.String(), "evt_context") {
		t.Fatalf("expected linked and context events, got %s", res.Body.String())
	}
}

func TestFindingEvidenceRejectsInvalidContextWindow(t *testing.T) {
	t.Parallel()

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings/fnd_test/evidence?context_minutes=90", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, &stubFindingReader{}).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestDetectionMetricsReturnsQualityData(t *testing.T) {
	t.Parallel()

	reader := &stubFindingReader{
		metrics: []database.DetectionMetrics{
			{
				DetectionID:        "DET-AUTH-001",
				HitCount:           10,
				FalsePositiveCount: 2,
				FalsePositiveRate:  0.2,
			},
		},
	}
	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/detections/metrics", nil)
	res := httptest.NewRecorder()

	New(checker, nil, nil, reader).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "\"false_positive_rate\":0.2") {
		t.Fatalf("expected quality metrics, got %s", res.Body.String())
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

	return New(checker, ingestor, events, nil)
}
