package behaviour

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testCatalogue() Catalogue {
	return Catalogue{CatalogueVersion: "catalogue-v1", ModelVersion: "model-v1", DatasetName: "synthetic-v1",
		Profiles: []Profile{{ID: "BA-001", Title: "Timing", Description: "Synthetic timing reference", Features: []string{"weekend"}, RequiredTelemetry: []string{"timestamp"}, ContextWindow: "event", Limitations: []string{"Shift work"}, AnalystActions: []string{"Check schedule"}}},
		Validation: CatalogueValidation{CatalogueVersion: "catalogue-v1", ModelVersion: "model-v1", DatasetName: "synthetic-v1", Threshold: 65,
			Profiles: []ProfileValidation{{ProfileID: "BA-001", ReferenceCount: 12, ChangedCount: 12, ChangedFlagged: 6, ChangedFlagRate: 0.5, ReferenceMeanScore: 20, ChangedMeanScore: 60, ChangedExplanationFeatures: []string{"weekend"}, Interpretation: "Synthetic only"}}}}
}

func TestCatalogueValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Catalogue)
	}{
		{"model boundary", func(c *Catalogue) { c.Validation.ModelVersion = "other" }},
		{"threshold", func(c *Catalogue) { c.Validation.Threshold = 100 }},
		{"missing validation", func(c *Catalogue) { c.Validation.Profiles = nil }},
		{"missing profile", func(c *Catalogue) { c.Profiles = nil }},
		{"duplicate profile", func(c *Catalogue) { c.Profiles = append(c.Profiles, c.Profiles[0]) }},
		{"unknown profile result", func(c *Catalogue) { c.Validation.Profiles[0].ProfileID = "other" }},
		{"bad counts", func(c *Catalogue) { c.Validation.Profiles[0].ChangedFlagged = 13 }},
		{"inconsistent rates", func(c *Catalogue) { c.Validation.Profiles[0].ChangedFlagRate = 0.2 }},
		{"non finite", func(c *Catalogue) { c.Validation.Profiles[0].ChangedMeanScore = math.NaN() }},
		{"oversized text", func(c *Catalogue) { c.Profiles[0].Description = strings.Repeat("x", 1025) }},
		{"missing limitations", func(c *Catalogue) { c.Profiles[0].Limitations = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testCatalogue()
			tc.mutate(&c)
			if c.Validate() == nil {
				t.Fatal("invalid catalogue accepted")
			}
		})
	}
	if err := testCatalogue().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogueClient(t *testing.T) {
	valid, _ := json.Marshal(testCatalogue())
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"valid", string(valid), 200, false},
		{"unavailable", "private upstream details", 503, true},
		{"malformed", "{", 200, true},
		{"empty", "{}", 200, true},
		{"oversized", strings.Repeat(" ", int(maxResponseBytes)+1), 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/v1/catalogue" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			_, err := NewClient(upstream.URL, time.Second).Catalogue(context.Background())
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewClient("http://127.0.0.1:1", time.Second).Catalogue(ctx); err == nil {
		t.Fatal("cancelled request accepted")
	}
	var disabled *Client
	if _, err := disabled.Catalogue(context.Background()); err == nil {
		t.Fatal("disabled client accepted")
	}
}
