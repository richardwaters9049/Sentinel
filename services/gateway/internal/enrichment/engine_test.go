package enrichment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type fakeRepository struct {
	indicators []Indicator
	lookupErr  error
	saveErr    error
	lookups    int
	saved      []Match
}

func (f *fakeRepository) LookupIndicators(
	context.Context,
	string,
	string,
	time.Time,
) ([]Indicator, error) {
	f.lookups++
	return append([]Indicator(nil), f.indicators...), f.lookupErr
}

func (f *fakeRepository) SaveEventEnrichments(
	_ context.Context,
	_ string,
	matches []Match,
) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, matches...)
	return nil
}

func TestEngineEnrichesMatchingNetworkIOC(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		indicators: []Indicator{
			{
				ID:               "ioc-1",
				SourceID:         "source-1",
				SourceConfidence: 85,
				Confidence:       95,
				Provenance: map[string]interface{}{
					"origin": "local fixture",
				},
			},
		},
	}
	engine := New(repo)
	event := telemetry.Event{
		EventID:   "evt-ioc-1",
		Timestamp: time.Now().UTC(),
		Network: &telemetry.Network{
			SourceIP: "198.51.100.66",
		},
	}

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("process enrichment: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("expected one saved enrichment, got %d", len(repo.saved))
	}

	match := repo.saved[0]
	if match.EventField != "network.source_ip" {
		t.Fatalf("unexpected field %q", match.EventField)
	}
	if match.ObservedValue != "198.51.100.66" {
		t.Fatalf("unexpected observed value %q", match.ObservedValue)
	}
	if match.EffectiveConfidence != 81 {
		t.Fatalf("expected confidence 81, got %d", match.EffectiveConfidence)
	}
}

func TestEngineCachesPositiveAndNegativeLookups(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	engine := New(repo)
	event := telemetry.Event{
		EventID:   "evt-cache-1",
		Timestamp: time.Now().UTC(),
		Network: &telemetry.Network{
			DestinationIP: "203.0.113.200",
		},
	}

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("first process: %v", err)
	}
	event.EventID = "evt-cache-2"
	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("second process: %v", err)
	}

	if repo.lookups != 1 {
		t.Fatalf("expected one repository lookup due to cache, got %d", repo.lookups)
	}
}

func TestEnginePropagatesLookupFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{lookupErr: errors.New("intel store unavailable")}
	engine := New(repo)

	err := engine.Process(context.Background(), telemetry.Event{
		EventID:   "evt-error",
		Timestamp: time.Now().UTC(),
		Network: &telemetry.Network{
			SourceIP: "198.51.100.66",
		},
	})
	if err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestEngineIgnoresEventsWithoutNetworkContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	if err := New(repo).Process(context.Background(), telemetry.Event{
		EventID: "evt-no-network",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lookups != 0 {
		t.Fatalf("expected no lookups, got %d", repo.lookups)
	}
}

func TestEffectiveConfidence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source    int
		indicator int
		want      int
	}{
		{source: 100, indicator: 100, want: 100},
		{source: 85, indicator: 95, want: 81},
		{source: 80, indicator: 80, want: 64},
		{source: 120, indicator: 200, want: 100},
		{source: -10, indicator: 90, want: 0},
	}

	for _, tc := range cases {
		if got := effectiveConfidence(tc.source, tc.indicator); got != tc.want {
			t.Fatalf(
				"effectiveConfidence(%d, %d)=%d want %d",
				tc.source,
				tc.indicator,
				got,
				tc.want,
			)
		}
	}
}
