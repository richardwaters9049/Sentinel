package behaviour

import (
	"testing"
	"time"
)

func TestMonitorScores(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	samples := func(n, score, prior int, age time.Duration, model string) []Score {
		result := make([]Score, n)
		for i := range result {
			result[i] = Score{ModelVersion: model, AnomalyScore: score, Threshold: 65, ScoredAt: now.Add(-age), Baseline: BaselineContext{PriorEvents24h: prior}}
		}
		return result
	}
	for _, tc := range []struct {
		name                              string
		current, reference                []Score
		distribution, baseline, freshness string
	}{
		{"normal", samples(30, 40, 5, time.Minute, "v1"), samples(30, 40, 5, 25*time.Hour, "v1"), "stable", "context_available", "fresh"},
		{"shift", samples(30, 100, 5, time.Minute, "v1"), samples(30, 0, 5, 25*time.Hour, "v1"), "distribution_change", "context_available", "fresh"},
		{"empty", nil, nil, "insufficient_samples", "insufficient_samples", "no_recent_scores"},
		{"minimum boundary", samples(29, 40, 5, time.Minute, "v1"), samples(30, 40, 5, 25*time.Hour, "v1"), "insufficient_samples", "context_available", "fresh"},
		{"cold stale", samples(30, 40, 4, time.Hour, "v1"), samples(30, 40, 5, 25*time.Hour, "v1"), "stable", "cold_context", "stale"},
		{"model boundary", samples(30, 40, 5, time.Minute, "v1"), samples(30, 40, 5, 25*time.Hour, "v2"), "insufficient_samples", "context_available", "fresh"},
		{"cap", samples(2001, 40, 5, time.Minute, "v1"), samples(30, 40, 5, 25*time.Hour, "v1"), "sample_limit", "context_available", "fresh"},
		{"wrong windows", samples(30, 40, 5, 25*time.Hour, "v1"), samples(30, 40, 5, 49*time.Hour, "v1"), "insufficient_samples", "insufficient_samples", "no_recent_scores"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := MonitorScores(now, "v1", tc.current, tc.reference)
			if m.Distribution.Status != tc.distribution || m.Baseline.Status != tc.baseline || m.Freshness.Status != tc.freshness {
				t.Fatalf("unexpected monitor: %+v", m)
			}
			if m.Current.Samples > MonitorSampleLimit {
				t.Fatal("unbounded window")
			}
			if tc.distribution == "distribution_change" && (m.Distance == nil || *m.Distance != 1) {
				t.Fatal("expected full separation")
			}
		})
	}
	current := samples(30, 40, 5, time.Minute, "v1")
	ref := samples(30, 40, 5, 25*time.Hour, "v1")
	for i := range current {
		current[i].Threshold = 99
		current[i].Anomalous = false
		ref[i].Threshold = 1
		ref[i].Anomalous = true
	}
	if m := MonitorScores(now, "v1", current, ref); m.Distribution.Status != "stable" {
		t.Fatal("threshold policy treated as drift")
	}
	// Exactly 20% of mass moved to another band reaches the documented boundary.
	for i := 0; i < 6; i++ {
		current[i].AnomalyScore = 80
	}
	if m := MonitorScores(now, "v1", current, ref); m.Distribution.Status != "distribution_change" {
		t.Fatalf("boundary missed: %+v", m)
	}
}
