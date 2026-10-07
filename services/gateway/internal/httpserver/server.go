package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/enrichment"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/hunting"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/investigation"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

const maxRequestBodyBytes int64 = 1 << 20

type TelemetryIngestor interface {
	Ingest(context.Context, telemetry.IngestRequest) (telemetry.Event, error)
}

type EventReader interface {
	ListEvents(context.Context, database.EventQuery) ([]telemetry.Event, error)
}

type IntelligenceStore interface {
	ListIntelligenceIndicators(context.Context, database.IntelligenceIndicatorQuery) ([]enrichment.Indicator, error)
	GetEventEnrichments(context.Context, string) ([]database.EventEnrichmentRecord, error)
	ListRecentEventEnrichments(context.Context, int) ([]database.EventEnrichmentRecord, error)
	IntelligenceMetrics(context.Context) (database.IntelligenceMetrics, error)
}

type AnalystStore interface {
	ListFindings(context.Context, database.FindingQuery) ([]database.FindingRecord, error)
	GetFinding(context.Context, string) (database.FindingDetail, error)
	GetFindingEvidence(context.Context, string, int) (database.FindingEvidenceContext, error)
	UpdateFindingStatus(context.Context, string, string, string, string) (database.FindingDetail, error)
	ListDetections(context.Context) ([]database.DetectionRecord, error)
	DetectionMetrics(context.Context) ([]database.DetectionMetrics, error)
	SetDetectionEnabled(context.Context, string, bool, string, string) (database.DetectionRecord, error)
	CreateHunt(context.Context, string, string, string, hunting.Query, string) (hunting.Definition, error)
	ListHunts(context.Context) ([]hunting.Definition, error)
	GetHunt(context.Context, string) (hunting.Definition, error)
	RunHunt(context.Context, string, string, hunting.Query) (hunting.RunResult, error)
	UpdateHunt(context.Context, string, string, string, string, hunting.Query, string) (hunting.Definition, error)
	ListHuntVersions(context.Context, string) ([]hunting.VersionRecord, error)
	ListHuntRuns(context.Context, string, int) ([]hunting.RunRecord, error)
	HuntRunEvents(context.Context, int64) ([]database.EventEvidenceRecord, error)
	HuntMetrics(context.Context) (database.HuntMetricsSummary, error)
	GetAssetPivot(context.Context, string, int) (database.AssetPivot, error)
	GetIdentityPivot(context.Context, string, int) (database.IdentityPivot, error)
	CreateInvestigation(context.Context, database.InvestigationCreateInput) (database.InvestigationDetail, error)
	ListInvestigations(context.Context, string, int) ([]investigation.Record, error)
	GetInvestigation(context.Context, string) (database.InvestigationDetail, error)
	AddInvestigationNote(context.Context, string, string, string, string) (database.InvestigationDetail, error)
	UpdateInvestigationStatus(context.Context, string, string, string, string) (database.InvestigationDetail, error)
	AttachHuntRun(context.Context, string, int64, string, string) (database.InvestigationDetail, error)
	UpdateInvestigationMetadata(context.Context, string, *string, *string, string, string) (database.InvestigationDetail, error)
	InvestigationMetrics(context.Context) (database.InvestigationMetricsSummary, error)
}

type Server struct {
	mux       *http.ServeMux
	readiness *readiness.Checker
	ingestor  TelemetryIngestor
	events    EventReader
	analyst   AnalystStore
}

func New(
	readinessChecker *readiness.Checker,
	ingestor TelemetryIngestor,
	events EventReader,
	analyst AnalystStore,
) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		readiness: readinessChecker,
		ingestor:  ingestor,
		events:    events,
		analyst:   analyst,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(limitRequestBody(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", method(http.MethodGet, s.handleHealth))
	s.mux.HandleFunc("/ready", method(http.MethodGet, s.handleReady))
	s.mux.HandleFunc("/api/v1/telemetry", method(http.MethodPost, s.handleTelemetry))
	s.mux.HandleFunc("/api/v1/events", method(http.MethodGet, s.handleEvents))
	s.mux.HandleFunc("/api/v1/findings", method(http.MethodGet, s.handleFindings))
	s.mux.HandleFunc("GET /api/v1/findings/{id}", s.handleFindingDetail)
	s.mux.HandleFunc("GET /api/v1/findings/{id}/evidence", s.handleFindingEvidence)
	s.mux.HandleFunc("PATCH /api/v1/findings/{id}/status", s.handleFindingStatus)
	s.mux.HandleFunc("GET /api/v1/detections", s.handleDetections)
	s.mux.HandleFunc("GET /api/v1/detections/metrics", s.handleDetectionMetrics)
	s.mux.HandleFunc("GET /api/v1/intelligence/indicators", s.handleIntelligenceIndicators)
	s.mux.HandleFunc("GET /api/v1/intelligence/matches", s.handleIntelligenceMatches)
	s.mux.HandleFunc("GET /api/v1/intelligence/metrics", s.handleIntelligenceMetrics)
	s.mux.HandleFunc("GET /api/v1/events/{id}/enrichments", s.handleEventEnrichments)
	s.mux.HandleFunc("PATCH /api/v1/detections/{id}", s.handleDetectionState)
	s.mux.HandleFunc("GET /api/v1/hunts", s.handleHunts)
	s.mux.HandleFunc("POST /api/v1/hunts", s.handleCreateHunt)
	s.mux.HandleFunc("GET /api/v1/hunts/{id}", s.handleHuntDetail)
	s.mux.HandleFunc("PATCH /api/v1/hunts/{id}", s.handleUpdateHunt)
	s.mux.HandleFunc("GET /api/v1/hunts/{id}/versions", s.handleHuntVersions)
	s.mux.HandleFunc("POST /api/v1/hunts/{id}/run", s.handleRunHunt)
	s.mux.HandleFunc("GET /api/v1/hunts/{id}/runs", s.handleHuntRuns)
	s.mux.HandleFunc("GET /api/v1/hunts/metrics", s.handleHuntMetrics)
	s.mux.HandleFunc("GET /api/v1/assets/{id}/pivot", s.handleAssetPivot)
	s.mux.HandleFunc("GET /api/v1/identities/{id}/pivot", s.handleIdentityPivot)
	s.mux.HandleFunc("GET /api/v1/investigations", s.handleInvestigations)
	s.mux.HandleFunc("POST /api/v1/investigations", s.handleCreateInvestigation)
	s.mux.HandleFunc("GET /api/v1/investigations/{id}", s.handleInvestigationDetail)
	s.mux.HandleFunc("POST /api/v1/investigations/{id}/notes", s.handleInvestigationNote)
	s.mux.HandleFunc("PATCH /api/v1/investigations/{id}/status", s.handleInvestigationStatus)
	s.mux.HandleFunc("PATCH /api/v1/investigations/{id}", s.handleInvestigationMetadata)
	s.mux.HandleFunc("POST /api/v1/investigations/{id}/hunt-runs/{run_id}", s.handleAttachHuntRun)
	s.mux.HandleFunc("GET /api/v1/investigations/metrics", s.handleInvestigationMetrics)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "gateway",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.readiness == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"ready":  false,
		})
		return
	}

	result := s.readiness.Check(r.Context())
	status := http.StatusOK
	statusText := "ready"

	if !result.Ready {
		status = http.StatusServiceUnavailable
		statusText = "not_ready"
	}

	writeJSON(w, status, map[string]interface{}{
		"status":       statusText,
		"ready":        result.Ready,
		"dependencies": result.Dependencies,
	})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if s.ingestor == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "telemetry_unavailable", "telemetry ingestion is unavailable")
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request telemetry.IngestRequest
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must contain one valid telemetry event")
		return
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON object")
		return
	}

	event, err := s.ingestor.Ingest(r.Context(), request)
	if err != nil {
		if errors.Is(err, telemetry.ErrInvalidEvent) {
			writeAPIError(w, http.StatusBadRequest, "invalid_event", err.Error())
			return
		}

		writeAPIError(w, http.StatusServiceUnavailable, "publish_failed", "telemetry event could not be accepted")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"accepted":       true,
		"event_id":       event.EventID,
		"schema_version": event.SchemaVersion,
		"received_at":    event.ReceivedAt,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.events == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "events_unavailable", "event storage is unavailable")
		return
	}

	query, err := parseEventQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	events, err := s.events.ListEvents(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "events could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":  len(events),
		"events": events,
	})
}

func (s *Server) handleIntelligenceIndicators(w http.ResponseWriter, r *http.Request) {
	store, ok := s.analyst.(IntelligenceStore)
	if !ok || store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "intelligence_unavailable", "threat intelligence is unavailable")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}

	indicatorType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	switch indicatorType {
	case "", "ip", "domain", "sha256":
	default:
		writeAPIError(w, http.StatusBadRequest, "invalid_query", "type must be ip, domain, or sha256")
		return
	}

	indicators, err := store.ListIntelligenceIndicators(
		r.Context(),
		database.IntelligenceIndicatorQuery{
			Limit:         limit,
			IndicatorType: indicatorType,
			Value:         strings.TrimSpace(r.URL.Query().Get("value")),
		},
	)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "threat indicators could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(indicators),
		"indicators": indicators,
	})
}

func (s *Server) handleIntelligenceMatches(w http.ResponseWriter, r *http.Request) {
	store, ok := s.analyst.(IntelligenceStore)
	if !ok || store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "intelligence_unavailable", "threat intelligence is unavailable")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}

	matches, err := store.ListRecentEventEnrichments(r.Context(), limit)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "enrichment matches could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(matches),
		"matches": matches,
	})
}

func (s *Server) handleIntelligenceMetrics(w http.ResponseWriter, r *http.Request) {
	store, ok := s.analyst.(IntelligenceStore)
	if !ok || store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "intelligence_unavailable", "threat intelligence is unavailable")
		return
	}

	metrics, err := store.IntelligenceMetrics(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "intelligence metrics could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleEventEnrichments(w http.ResponseWriter, r *http.Request) {
	store, ok := s.analyst.(IntelligenceStore)
	if !ok || store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "intelligence_unavailable", "event enrichment is unavailable")
		return
	}

	eventID := strings.TrimSpace(r.PathValue("id"))
	if eventID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_event_id", "event id is required")
		return
	}

	enrichments, err := store.GetEventEnrichments(r.Context(), eventID)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "event enrichments could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_id":    eventID,
		"count":       len(enrichments),
		"enrichments": enrichments,
	})
}

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "findings_unavailable", "finding storage is unavailable")
		return
	}

	query, err := parseFindingQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	findings, err := s.analyst.ListFindings(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "findings could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(findings),
		"findings": findings,
	})
}

func (s *Server) handleFindingDetail(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "findings_unavailable", "finding storage is unavailable")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_finding_id", "finding id is required")
		return
	}

	finding, err := s.analyst.GetFinding(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrFindingNotFound) {
			writeAPIError(w, http.StatusNotFound, "finding_not_found", "finding was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "finding could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, finding)
}

func (s *Server) handleFindingEvidence(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "findings_unavailable", "finding storage is unavailable")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_finding_id", "finding id is required")
		return
	}

	contextMinutes := 5
	if raw := strings.TrimSpace(r.URL.Query().Get("context_minutes")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || value > 60 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "context_minutes must be an integer between 0 and 60")
			return
		}
		contextMinutes = value
	}

	evidence, err := s.analyst.GetFindingEvidence(r.Context(), id, contextMinutes)
	if err != nil {
		if errors.Is(err, database.ErrFindingNotFound) {
			writeAPIError(w, http.StatusNotFound, "finding_not_found", "finding was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "finding evidence could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, evidence)
}

func (s *Server) handleDetectionMetrics(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "detections_unavailable", "detection storage is unavailable")
		return
	}

	metrics, err := s.analyst.DetectionMetrics(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "detection metrics could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(metrics),
		"metrics": metrics,
	})
}

func (s *Server) handleFindingStatus(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "findings_unavailable", "finding storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_finding_id", "finding id is required")
		return
	}

	var request struct {
		Status string `json:"status"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	request.Status = strings.TrimSpace(request.Status)
	if request.Status == "" {
		writeAPIError(w, http.StatusBadRequest, "status_required", "status is required")
		return
	}

	finding, err := s.analyst.UpdateFindingStatus(
		r.Context(),
		id,
		request.Status,
		actorID,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrFindingNotFound):
			writeAPIError(w, http.StatusNotFound, "finding_not_found", "finding was not found")
		case errors.Is(err, database.ErrInvalidTransition):
			writeAPIError(w, http.StatusConflict, "invalid_transition", err.Error())
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "finding status could not be updated")
		}
		return
	}

	writeJSON(w, http.StatusOK, finding)
}

func (s *Server) handleDetections(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "detections_unavailable", "detection storage is unavailable")
		return
	}

	detections, err := s.analyst.ListDetections(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "detections could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(detections),
		"detections": detections,
	})
}

func (s *Server) handleDetectionState(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "detections_unavailable", "detection storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_detection_id", "detection id is required")
		return
	}

	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "enabled_required", "enabled is required")
		return
	}

	detectionRecord, err := s.analyst.SetDetectionEnabled(
		r.Context(),
		id,
		*request.Enabled,
		actorID,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDetectionNotFound):
			writeAPIError(w, http.StatusNotFound, "detection_not_found", "detection was not found")
		case errors.Is(err, database.ErrDetectionUnchanged):
			writeAPIError(w, http.StatusConflict, "detection_unchanged", "detection already has the requested enabled state")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "detection state could not be updated")
		}
		return
	}

	writeJSON(w, http.StatusOK, detectionRecord)
}

func decodeStrictJSON(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return errors.New("request body must contain valid JSON")
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}

	return nil
}

func parseFindingQuery(r *http.Request) (database.FindingQuery, error) {
	values := r.URL.Query()
	query := database.FindingQuery{
		Limit:    50,
		Status:   strings.TrimSpace(values.Get("status")),
		Severity: strings.TrimSpace(values.Get("severity")),
	}

	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 200 {
			return database.FindingQuery{}, errors.New("limit must be an integer between 1 and 200")
		}
		query.Limit = limit
	}

	switch query.Status {
	case "", "new", "triaged", "investigating", "false_positive", "benign_expected", "duplicate", "confirmed", "contained", "closed":
	default:
		return database.FindingQuery{}, errors.New("status is not supported")
	}

	switch query.Severity {
	case "", "informational", "low", "medium", "high", "critical":
	default:
		return database.FindingQuery{}, errors.New("severity is not supported")
	}

	return query, nil
}

func parseEventQuery(r *http.Request) (database.EventQuery, error) {
	values := r.URL.Query()
	query := database.EventQuery{
		Limit:      50,
		Category:   strings.TrimSpace(values.Get("category")),
		AssetID:    strings.TrimSpace(values.Get("asset_id")),
		IdentityID: strings.TrimSpace(values.Get("identity_id")),
	}

	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 200 {
			return database.EventQuery{}, errors.New("limit must be an integer between 1 and 200")
		}
		query.Limit = limit
	}

	if raw := strings.TrimSpace(values.Get("before")); raw != "" {
		before, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return database.EventQuery{}, errors.New("before must be an RFC3339 timestamp")
		}
		query.Before = &before
	}

	return query, nil
}

func method(allowed string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowed {
			w.Header().Set("Allow", allowed)
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
