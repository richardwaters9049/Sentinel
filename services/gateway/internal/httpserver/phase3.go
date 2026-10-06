package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/hunting"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/investigation"
)

func (s *Server) handleHunts(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	hunts, err := s.analyst.ListHunts(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "hunts could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(hunts),
		"hunts": hunts,
	})
}

func (s *Server) handleCreateHunt(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Hypothesis  string        `json:"hypothesis"`
		Query       hunting.Query `json:"query"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	hunt, err := s.analyst.CreateHunt(
		r.Context(),
		request.Name,
		request.Description,
		request.Hypothesis,
		request.Query,
		actorID,
	)
	if err != nil {
		if errors.Is(err, hunting.ErrInvalidHunt) {
			writeAPIError(w, http.StatusBadRequest, "invalid_hunt", "hunt definition is invalid")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "create_failed", "hunt could not be created")
		return
	}

	writeJSON(w, http.StatusCreated, hunt)
}

func (s *Server) handleHuntDetail(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	hunt, err := s.analyst.GetHunt(r.Context(), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		if errors.Is(err, database.ErrHuntNotFound) {
			writeAPIError(w, http.StatusNotFound, "hunt_not_found", "hunt was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "hunt could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, hunt)
}

func (s *Server) handleUpdateHunt(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Hypothesis  string        `json:"hypothesis"`
		Query       hunting.Query `json:"query"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	hunt, err := s.analyst.UpdateHunt(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		request.Name,
		request.Description,
		request.Hypothesis,
		request.Query,
		actorID,
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrHuntNotFound):
			writeAPIError(w, http.StatusNotFound, "hunt_not_found", "hunt was not found")
		case errors.Is(err, hunting.ErrInvalidHunt):
			writeAPIError(w, http.StatusBadRequest, "invalid_hunt", "hunt definition is invalid")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "hunt could not be updated")
		}
		return
	}

	writeJSON(w, http.StatusOK, hunt)
}

func (s *Server) handleHuntVersions(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	versions, err := s.analyst.ListHuntVersions(r.Context(), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		if errors.Is(err, database.ErrHuntNotFound) {
			writeAPIError(w, http.StatusNotFound, "hunt_not_found", "hunt was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "hunt versions could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(versions),
		"versions": versions,
	})
}

func (s *Server) handleHuntMetrics(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	metrics, err := s.analyst.HuntMetrics(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "hunt metrics could not be queried")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleRunHunt(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Override hunting.Query `json:"override"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	result, err := s.analyst.RunHunt(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		actorID,
		request.Override,
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrHuntNotFound):
			writeAPIError(w, http.StatusNotFound, "hunt_not_found", "hunt was not found")
		case errors.Is(err, hunting.ErrInvalidHunt):
			writeAPIError(w, http.StatusBadRequest, "invalid_hunt", "hunt query is invalid")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "hunt_failed", "hunt could not be executed")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleHuntRuns(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "hunts_unavailable", "hunt storage is unavailable")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "limit must be an integer between 1 and 200")
			return
		}
		limit = value
	}

	huntID := strings.TrimSpace(r.PathValue("id"))
	runs, err := s.analyst.ListHuntRuns(r.Context(), huntID, limit)
	if err != nil {
		if errors.Is(err, database.ErrHuntNotFound) {
			writeAPIError(w, http.StatusNotFound, "hunt_not_found", "hunt was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "hunt runs could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(runs),
		"runs":  runs,
	})
}

func (s *Server) handleAssetPivot(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "pivots_unavailable", "asset pivot storage is unavailable")
		return
	}

	limit, ok := parsePivotLimit(w, r)
	if !ok {
		return
	}

	pivot, err := s.analyst.GetAssetPivot(r.Context(), strings.TrimSpace(r.PathValue("id")), limit)
	if err != nil {
		if errors.Is(err, database.ErrAssetNotFound) {
			writeAPIError(w, http.StatusNotFound, "asset_not_found", "asset was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "asset pivot could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, pivot)
}

func (s *Server) handleIdentityPivot(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "pivots_unavailable", "identity pivot storage is unavailable")
		return
	}

	limit, ok := parsePivotLimit(w, r)
	if !ok {
		return
	}

	pivot, err := s.analyst.GetIdentityPivot(r.Context(), strings.TrimSpace(r.PathValue("id")), limit)
	if err != nil {
		if errors.Is(err, database.ErrIdentityNotFound) {
			writeAPIError(w, http.StatusNotFound, "identity_not_found", "identity was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "identity pivot could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, pivot)
}

func parsePivotLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "limit must be an integer between 1 and 200")
			return 0, false
		}
		limit = value
	}
	return limit, true
}

func (s *Server) handleInvestigationMetrics(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	metrics, err := s.analyst.InvestigationMetrics(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "investigation metrics could not be queried")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleInvestigations(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			writeAPIError(w, http.StatusBadRequest, "invalid_query", "limit must be an integer between 1 and 200")
			return
		}
		limit = value
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	switch status {
	case "", "open", "investigating", "contained", "closed":
	default:
		writeAPIError(w, http.StatusBadRequest, "invalid_query", "status is not supported")
		return
	}

	records, err := s.analyst.ListInvestigations(r.Context(), status, limit)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "investigations could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":          len(records),
		"investigations": records,
	})
}

func (s *Server) handleCreateInvestigation(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Priority    string   `json:"priority"`
		OwnerID     string   `json:"owner_id"`
		FindingIDs  []string `json:"finding_ids"`
		EventIDs    []string `json:"event_ids"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	record, err := s.analyst.CreateInvestigation(r.Context(), database.InvestigationCreateInput{
		Title:       request.Title,
		Description: request.Description,
		Priority:    request.Priority,
		OwnerID:     request.OwnerID,
		CreatedBy:   actorID,
		FindingIDs:  request.FindingIDs,
		EventIDs:    request.EventIDs,
		RequestID:   strings.TrimSpace(r.Header.Get("X-Request-ID")),
	})
	if err != nil {
		if errors.Is(err, investigation.ErrInvalidInvestigation) {
			writeAPIError(w, http.StatusBadRequest, "invalid_investigation", "investigation definition is invalid")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "create_failed", "investigation could not be created")
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleInvestigationDetail(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	record, err := s.analyst.GetInvestigation(r.Context(), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		if errors.Is(err, database.ErrInvestigationNotFound) {
			writeAPIError(w, http.StatusNotFound, "investigation_not_found", "investigation was not found")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "query_failed", "investigation could not be queried")
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleInvestigationNote(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Body string `json:"body"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	record, err := s.analyst.AddInvestigationNote(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		actorID,
		request.Body,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvestigationNotFound):
			writeAPIError(w, http.StatusNotFound, "investigation_not_found", "investigation was not found")
		case errors.Is(err, investigation.ErrInvalidInvestigation):
			writeAPIError(w, http.StatusBadRequest, "invalid_note", "note must contain between 1 and 5000 characters")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "investigation note could not be added")
		}
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleInvestigationMetadata(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		OwnerID  *string `json:"owner_id"`
		Priority *string `json:"priority"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if request.OwnerID == nil && request.Priority == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_investigation", "owner_id or priority is required")
		return
	}

	record, err := s.analyst.UpdateInvestigationMetadata(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		request.OwnerID,
		request.Priority,
		actorID,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvestigationNotFound):
			writeAPIError(w, http.StatusNotFound, "investigation_not_found", "investigation was not found")
		case errors.Is(err, investigation.ErrInvalidInvestigation):
			writeAPIError(w, http.StatusBadRequest, "invalid_investigation", "investigation metadata is invalid")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "investigation metadata could not be updated")
		}
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleAttachHuntRun(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	runID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("run_id")), 10, 64)
	if err != nil || runID < 1 {
		writeAPIError(w, http.StatusBadRequest, "invalid_hunt_run_id", "hunt run id must be a positive integer")
		return
	}

	record, err := s.analyst.AttachHuntRun(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		runID,
		actorID,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvestigationNotFound):
			writeAPIError(w, http.StatusNotFound, "investigation_not_found", "investigation was not found")
		case errors.Is(err, database.ErrHuntRunNotFound):
			writeAPIError(w, http.StatusNotFound, "hunt_run_not_found", "hunt run was not found")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "hunt run could not be attached")
		}
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleInvestigationStatus(w http.ResponseWriter, r *http.Request) {
	if s.analyst == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "investigations_unavailable", "investigation storage is unavailable")
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
	if actorID == "" {
		writeAPIError(w, http.StatusBadRequest, "actor_required", "X-Sentinel-Actor header is required")
		return
	}

	var request struct {
		Status string `json:"status"`
	}
	if err := decodeStrictJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	record, err := s.analyst.UpdateInvestigationStatus(
		r.Context(),
		strings.TrimSpace(r.PathValue("id")),
		request.Status,
		actorID,
		strings.TrimSpace(r.Header.Get("X-Request-ID")),
	)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvestigationNotFound):
			writeAPIError(w, http.StatusNotFound, "investigation_not_found", "investigation was not found")
		case errors.Is(err, investigation.ErrInvalidTransition):
			writeAPIError(w, http.StatusConflict, "invalid_transition", "investigation status transition is not allowed")
		default:
			writeAPIError(w, http.StatusServiceUnavailable, "update_failed", "investigation status could not be updated")
		}
		return
	}

	writeJSON(w, http.StatusOK, record)
}
