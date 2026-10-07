package behaviour

import (
	"context"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type fakeScorer struct {
	request ScoreRequest
	score   Score
	err     error
}

func (f *fakeScorer) Score(_ context.Context, request ScoreRequest) (Score, error) {
	f.request = request
	if f.score.EventID == "" {
		f.score = Score{
			EventID:      request.EventID,
			EntityID:     request.EntityID,
			ModelVersion: "test-v1",
			ModelKind:    "IsolationForest",
			AnomalyScore: 72,
			Severity:     "medium",
			Anomalous:    true,
			Threshold:    65,
			ScoredAt:     time.Now().UTC(),
		}
	}
	return f.score, f.err
}

type fakeStore struct {
	saved Score
	err   error
}

func (f *fakeStore) SaveBehaviourScore(_ context.Context, score Score) error {
	f.saved = score
	return f.err
}

func TestProcessorUsesActorAsPrimaryEntityAndPersistsScore(t *testing.T) {
	t.Parallel()

	scorer := &fakeScorer{}
	store := &fakeStore{}
	processor := NewProcessor(scorer, store)

	event := telemetry.Event{
		EventID:   "evt-behaviour-1",
		Timestamp: time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC),
		Source: telemetry.Source{
			Type:      "identity",
			Collector: "collector-1",
		},
		Asset: &telemetry.Asset{
			ID:   "asset-1",
			Zone: "corporate",
		},
		Actor: &telemetry.Actor{
			ID:   "svc-1",
			Type: "service_account",
			Name: "Service",
		},
		Event: telemetry.EventDetails{
			Category: "authentication",
			Action:   "login",
			Outcome:  "failure",
		},
		Network: &telemetry.Network{
			DestinationZone: "ot",
			DestinationPort: 502,
		},
	}

	if err := processor.Process(context.Background(), event); err != nil {
		t.Fatalf("process behaviour: %v", err)
	}

	if scorer.request.EntityID != "svc-1" {
		t.Fatalf("expected actor entity, got %q", scorer.request.EntityID)
	}
	if scorer.request.SourceZone != "corporate" {
		t.Fatalf("unexpected source zone %q", scorer.request.SourceZone)
	}
	if scorer.request.DestinationZone != "ot" || scorer.request.DestinationPort != 502 {
		t.Fatalf("unexpected destination context: %#v", scorer.request)
	}
	if store.saved.EventID != event.EventID {
		t.Fatalf("expected score persistence for %q, got %q", event.EventID, store.saved.EventID)
	}
}

func TestProcessorFallsBackToAssetThenCollector(t *testing.T) {
	t.Parallel()

	scorer := &fakeScorer{}
	store := &fakeStore{}
	processor := NewProcessor(scorer, store)

	event := telemetry.Event{
		EventID:   "evt-behaviour-asset",
		Timestamp: time.Now().UTC(),
		Source: telemetry.Source{
			Type:      "endpoint",
			Collector: "collector-fallback",
		},
		Asset: &telemetry.Asset{
			ID:   "asset-fallback",
			Zone: "corporate",
		},
		Event: telemetry.EventDetails{Category: "process", Action: "start"},
	}

	if err := processor.Process(context.Background(), event); err != nil {
		t.Fatalf("process asset fallback: %v", err)
	}
	if scorer.request.EntityID != "asset-fallback" {
		t.Fatalf("expected asset fallback, got %q", scorer.request.EntityID)
	}

	event.EventID = "evt-behaviour-collector"
	event.Asset = nil
	if err := processor.Process(context.Background(), event); err != nil {
		t.Fatalf("process collector fallback: %v", err)
	}
	if scorer.request.EntityID != "collector-fallback" {
		t.Fatalf("expected collector fallback, got %q", scorer.request.EntityID)
	}
}
