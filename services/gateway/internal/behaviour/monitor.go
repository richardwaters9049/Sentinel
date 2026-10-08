package behaviour

import (
	"math"
	"time"
)

const MonitorSampleLimit = 2000
const MonitorMinimumSamples = 30
const MonitorShiftThreshold = 0.20

type HealthSignal struct {
	Status      string `json:"status"`
	Explanation string `json:"explanation"`
}
type MonitorWindow struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Samples     int       `json:"samples"`
	Histogram   [5]int    `json:"histogram"`
	ColdSamples int       `json:"cold_samples"`
	Truncated   bool      `json:"truncated"`
}
type Monitor struct {
	CheckedAt      time.Time     `json:"checked_at"`
	ModelVersion   string        `json:"model_version"`
	Current        MonitorWindow `json:"current"`
	Reference      MonitorWindow `json:"reference"`
	MinimumSamples int           `json:"minimum_samples"`
	ShiftThreshold float64       `json:"shift_threshold"`
	Distance       *float64      `json:"distance"`
	LastScoredAt   *time.Time    `json:"last_scored_at,omitempty"`
	Distribution   HealthSignal  `json:"distribution"`
	Baseline       HealthSignal  `json:"baseline"`
	Freshness      HealthSignal  `json:"freshness"`
	Availability   HealthSignal  `json:"availability"`
	Evaluation     HealthSignal  `json:"evaluation"`
}

// MonitorScores compares fixed score bands, independently of operational thresholds.
func MonitorScores(now time.Time, version string, current, reference []Score) Monitor {
	now = now.UTC()
	m := Monitor{CheckedAt: now, ModelVersion: version, MinimumSamples: MonitorMinimumSamples, ShiftThreshold: MonitorShiftThreshold,
		Current: MonitorWindow{Start: now.Add(-24 * time.Hour), End: now}, Reference: MonitorWindow{Start: now.Add(-48 * time.Hour), End: now.Add(-24 * time.Hour)}}
	fill := func(w *MonitorWindow, scores []Score) {
		for _, s := range scores {
			if s.ModelVersion != version || s.ScoredAt.Before(w.Start) || !s.ScoredAt.Before(w.End) {
				continue
			}
			if w.Samples == MonitorSampleLimit {
				w.Truncated = true
				break
			}
			w.Samples++
			band := s.AnomalyScore / 20
			if band > 4 {
				band = 4
			}
			if band < 0 {
				band = 0
			}
			w.Histogram[band]++
			if s.Baseline.PriorEvents24h < 5 {
				w.ColdSamples++
			}
			if m.LastScoredAt == nil || s.ScoredAt.After(*m.LastScoredAt) {
				t := s.ScoredAt.UTC()
				m.LastScoredAt = &t
			}
		}
	}
	fill(&m.Current, current)
	fill(&m.Reference, reference)
	m.Distribution = HealthSignal{"insufficient_samples", "At least 30 scores from the same model are required in each window."}
	if m.Current.Truncated || m.Reference.Truncated {
		m.Distribution = HealthSignal{"sample_limit", "A window exceeds 2,000 scores; distribution comparison is withheld to avoid sampling bias."}
	} else if m.Current.Samples >= MonitorMinimumSamples && m.Reference.Samples >= MonitorMinimumSamples {
		distance := 0.0
		for i := range m.Current.Histogram {
			distance += math.Abs(float64(m.Current.Histogram[i])/float64(m.Current.Samples)-float64(m.Reference.Histogram[i])/float64(m.Reference.Samples)) / 2
		}
		m.Distance = &distance
		m.Distribution = HealthSignal{"stable", "Score-band proportions differ by less than 0.20 total variation distance."}
		// Round only floating-point noise at the documented decision boundary.
		if distance+1e-12 >= MonitorShiftThreshold {
			m.Distribution = HealthSignal{"distribution_change", "Score-band proportions changed. Review telemetry composition and context; this does not establish model drift or compromise."}
		}
	}
	m.Baseline = HealthSignal{"insufficient_samples", "No recent scores are available to assess rolling context."}
	if m.Current.Samples > 0 {
		m.Baseline = HealthSignal{"context_available", "Recent scores each had at least five prior entity events in their persisted 24-hour context."}
		if m.Current.ColdSamples > 0 {
			m.Baseline = HealthSignal{"cold_context", "Some recent scores had fewer than five prior entity events. Context depth does not establish baseline quality."}
		}
	}
	m.Freshness = HealthSignal{"no_recent_scores", "No scores for this model and scope in the last 48 hours; telemetry absence and scoring failure cannot be distinguished."}
	if m.LastScoredAt != nil {
		m.Freshness = HealthSignal{"fresh", "Latest persisted score is within 15 minutes."}
		if now.Sub(*m.LastScoredAt) > 15*time.Minute {
			m.Freshness = HealthSignal{"stale", "Latest persisted score is older than 15 minutes; check telemetry activity and processing health."}
		}
	}
	m.Availability = HealthSignal{"unknown", "Live analytics availability has not been checked."}
	m.Evaluation = HealthSignal{"not_evaluated", "No matching synthetic evaluation is available for this model and current threshold."}
	return m
}
