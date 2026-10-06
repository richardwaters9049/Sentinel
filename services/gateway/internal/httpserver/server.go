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

type FindingReader interface {
	ListFindings(context.Context, database.FindingQuery) ([]database.FindingRecord, error)
}

type Server struct {
	mux       *http.ServeMux
	readiness *readiness.Checker
	ingestor  TelemetryIngestor
	events    EventReader
	findings  FindingReader
}

func New(
	readinessChecker *readiness.Checker,
	ingestor TelemetryIngestor,
	events EventReader,
	findings FindingReader,
) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		readiness: readinessChecker,
		ingestor:  ingestor,
		events:    events,
		findings:  findings,
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

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	if s.findings == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "findings_unavailable", "finding storage is unavailable")
		return
	}

	query, err := parseFindingQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	findings, err := s.findings.ListFindings(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "findings could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(findings),
		"findings": findings,
	})
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
