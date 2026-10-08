package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/database"
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
				req := httptest.NewRequest(http.MethodGet, "/api/v1/behaviour/"+endpoint+tc.query, nil)
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
		New(nil, nil, nil, store).Handler().ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/behaviour/scores?limit="+limit, nil))
		if res.Code != 400 || store.calls != 0 {
			t.Fatalf("invalid limit accepted: %s", limit)
		}
	}
}
