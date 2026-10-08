package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"encoding/json"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
)

type entityBehaviourStore struct {
	stubFindingReader
	BehaviourStore
	query database.BehaviourQuery
	calls int
}

func (s *entityBehaviourStore) ListBehaviourScores(_ context.Context, q database.BehaviourQuery) ([]behaviour.Score, error) {
	s.query = q
	s.calls++
	return []behaviour.Score{}, nil
}

func (s *entityBehaviourStore) BehaviourMetrics(_ context.Context, q database.BehaviourQuery) (database.BehaviourMetrics, error) {
	s.query = q
	s.calls++
	return database.BehaviourMetrics{}, nil
}

func TestBehaviourEntityFilters(t *testing.T) {
	for _, endpoint := range []string{"scores", "metrics"} {
		for _, tc := range []struct {
			name, query, id, kind string
			status                int
		}{
			{"all", "", "", "", 200},
			{"identity", "?entity_type=identity&entity_id=shared", "shared", "identity", 200},
			{"asset namespace", "?entity_type=asset&entity_id=shared", "shared", "asset", 200},
			{"collector", "?entity_type=collector&entity_id=agent", "agent", "collector", 200},
			{"encoded ID", "?entity_type=identity&entity_id=svc%2Bworker%26ops", "svc+worker&ops", "identity", 200},
			{"missing type", "?entity_id=shared", "", "", 400},
			{"missing ID", "?entity_type=identity", "", "", 400},
			{"unknown type", "?entity_type=process&entity_id=shared", "", "", 400},
			{"oversized ID", "?entity_type=identity&entity_id=" + strings.Repeat("x", 257), "", "", 400},
			{"blank ID", "?entity_type=identity&entity_id=%20", "", "", 400},
		} {
			t.Run(endpoint+"/"+tc.name, func(t *testing.T) {
				store := &entityBehaviourStore{}
				req := localRequest(http.MethodGet, "/api/v1/behaviour/"+endpoint+tc.query, nil)
				res := httptest.NewRecorder()
				New(nil, nil, nil, store).Handler().ServeHTTP(res, req)
				if res.Code != tc.status {
					t.Fatalf("status %d: %s", res.Code, res.Body.String())
				}
				if tc.status == 400 {
					if store.calls != 0 {
						t.Fatal("invalid filter reached persistence")
					}
					return
				}
				if store.calls != 1 || store.query.EntityID != tc.id || store.query.EntityType != tc.kind {
					t.Fatalf("wrong entity query: %+v", store.query)
				}
			})
		}
	}
}

func TestBehaviourScoreLimit(t *testing.T) {
	for _, limit := range []string{"0", "201", "bogus"} {
		store := &entityBehaviourStore{}
		res := httptest.NewRecorder()
		New(nil, nil, nil, store).Handler().ServeHTTP(res, localRequest("GET", "/api/v1/behaviour/scores?limit="+limit, nil))
		if res.Code != 400 || store.calls != 0 {
			t.Fatalf("invalid limit accepted: %s", limit)
		}
	}
}

type monitorStore struct {
	entityBehaviourStore
	fail bool
}

func (s *monitorStore) BehaviourMonitor(_ context.Context, q database.BehaviourQuery, now time.Time) (behaviour.Monitor, error) {
	s.query = q
	s.calls++
	if s.fail {
		return behaviour.Monitor{}, errors.New("private dependency details")
	}
	return behaviour.MonitorScores(now, "v1", nil, nil), nil
}
func TestBehaviourMonitorAPI(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		fail       bool
		status     int
	}{
		{"scope", "?entity_type=asset&entity_id=shared", false, 200},
		{"invalid", "?entity_id=shared", false, 400},
		{"dependency failure", "", true, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &monitorStore{fail: tc.fail}
			res := httptest.NewRecorder()
			New(nil, nil, nil, store).Handler().ServeHTTP(res, localRequest("GET", "/api/v1/behaviour/monitor"+tc.path, nil))
			if res.Code != tc.status {
				t.Fatalf("%d %s", res.Code, res.Body.String())
			}
			if strings.Contains(res.Body.String(), "private dependency") {
				t.Fatal("leaked error")
			}
			if tc.status == 400 && store.calls != 0 {
				t.Fatal("invalid scope queried")
			}
			if tc.name == "scope" && (store.query.EntityType != "asset" || store.query.EntityID != "shared") {
				t.Fatal("scope lost")
			}
		})
	}
	res := httptest.NewRecorder()
	New(nil, nil, nil, nil).Handler().ServeHTTP(res, localRequest("GET", "/api/v1/behaviour/monitor", nil))
	if res.Code != 503 {
		t.Fatal("missing store accepted")
	}
}

func TestMonitorAvailability(t *testing.T) {
	for _, status := range []string{"available", "unavailable", "disabled"} {
		t.Run(status, func(t *testing.T) {
			checks := map[string]readiness.CheckFunc{}
			if status != "disabled" {
				checks["ml"] = func(context.Context) error {
					if status == "unavailable" {
						return errors.New("dependency down")
					}
					return nil
				}
			}
			res := httptest.NewRecorder()
			New(readiness.New(time.Second, checks), nil, nil, &monitorStore{}).Handler().ServeHTTP(res, localRequest("GET", "/api/v1/behaviour/monitor", nil))
			var monitor behaviour.Monitor
			if err := json.Unmarshal(res.Body.Bytes(), &monitor); err != nil {
				t.Fatal(err)
			}
			if monitor.Availability.Status != status {
				t.Fatalf("%+v", monitor.Availability)
			}
		})
	}
}
