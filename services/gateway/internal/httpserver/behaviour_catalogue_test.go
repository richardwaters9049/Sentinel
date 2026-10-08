package httpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
)

type catalogueReader struct {
	calls         int
	invalid, fail bool
}

func (c *catalogueReader) Catalogue(ctx context.Context) (behaviour.Catalogue, error) {
	c.calls++
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 3*time.Second {
		return behaviour.Catalogue{}, errors.New("missing request deadline")
	}
	if c.fail {
		return behaviour.Catalogue{}, errors.New("private upstream failure")
	}
	if c.invalid {
		return behaviour.Catalogue{}, nil
	}
	return behaviour.Catalogue{CatalogueVersion: "v1", ModelVersion: "model-v1", DatasetName: "synthetic-v1",
		Profiles: []behaviour.Profile{{ID: "BA-001", Title: "Timing", Description: "Timing profile", Features: []string{"weekend"}, RequiredTelemetry: []string{"timestamp"}, ContextWindow: "event", Limitations: []string{"Shift work"}, AnalystActions: []string{"Check schedule"}}},
		Validation: behaviour.CatalogueValidation{CatalogueVersion: "v1", ModelVersion: "model-v1", DatasetName: "synthetic-v1", Threshold: 65,
			Profiles: []behaviour.ProfileValidation{{ProfileID: "BA-001", ReferenceCount: 12, ChangedCount: 12, ChangedExplanationFeatures: []string{"weekend"}, Interpretation: "Synthetic context only"}}}}, nil
}

func TestBehaviourCatalogueAPI(t *testing.T) {
	for _, tc := range []struct {
		name, method, query string
		fail, invalid       bool
		status, calls       int
	}{
		{"available", "GET", "", false, false, 200, 1},
		{"no entity filters", "GET", "?entity_type=asset&entity_id=test", false, false, 400, 0},
		{"no threshold policy", "GET", "?threshold=71", false, false, 400, 0},
		{"unavailable", "GET", "", true, false, 503, 1},
		{"invalid dependency", "GET", "", false, true, 503, 1},
		{"read only", "POST", "", false, false, 405, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &catalogueReader{fail: tc.fail, invalid: tc.invalid}
			res := httptest.NewRecorder()
			NewWithBehaviourCatalogue(nil, nil, nil, nil, reader).Handler().ServeHTTP(res, httptest.NewRequest(tc.method, "/api/v1/behaviour/catalogue"+tc.query, nil))
			if res.Code != tc.status || reader.calls != tc.calls {
				t.Fatalf("status %d calls %d: %s", res.Code, reader.calls, res.Body.String())
			}
			if strings.Contains(res.Body.String(), "private upstream") {
				t.Fatal("leaked dependency error")
			}
		})
	}
	res := httptest.NewRecorder()
	New(nil, nil, nil, nil).Handler().ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/behaviour/catalogue", nil))
	if res.Code != 503 {
		t.Fatal("disabled catalogue accepted")
	}
}
