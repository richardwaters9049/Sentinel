package enrichment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type fakeRepository struct {
	indicators []Indicator
	lookupErr  error
	saveErr    error
	revision   time.Time
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

func (f *fakeRepository) IntelligenceRevision(context.Context) (time.Time, error) {
	if f.revision.IsZero() {
		return time.Unix(1, 0).UTC(), nil
	}
	return f.revision, nil
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

func TestEngineEnrichesDomainAndSHA256Labels(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		indicators: []Indicator{
			{
				ID:               "ioc-domain",
				SourceID:         "source-1",
				SourceConfidence: 85,
				Confidence:       92,
			},
		},
	}
	engine := New(repo)

	domainEvent := telemetry.Event{
		EventID:   "evt-domain",
		Timestamp: time.Now().UTC(),
		Labels: map[string]string{
			"dns.query": "Telemetry-Sync.Example.",
		},
	}
	if err := engine.Process(context.Background(), domainEvent); err != nil {
		t.Fatalf("process domain enrichment: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("expected one domain enrichment, got %d", len(repo.saved))
	}
	if repo.saved[0].EventField != "labels.dns.query" {
		t.Fatalf("unexpected domain field %q", repo.saved[0].EventField)
	}
	if repo.saved[0].ObservedValue != "telemetry-sync.example" {
		t.Fatalf("unexpected normalized domain %q", repo.saved[0].ObservedValue)
	}

	repo.saved = nil
	repo.indicators = []Indicator{
		{
			ID:               "ioc-sha",
			SourceID:         "source-1",
			SourceConfidence: 85,
			Confidence:       90,
		},
	}

	hash := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	hashEvent := telemetry.Event{
		EventID:   "evt-hash",
		Timestamp: time.Now().UTC(),
		Labels: map[string]string{
			"file.sha256": hash,
		},
	}
	if err := engine.Process(context.Background(), hashEvent); err != nil {
		t.Fatalf("process hash enrichment: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("expected one hash enrichment, got %d", len(repo.saved))
	}
	if repo.saved[0].EventField != "labels.file.sha256" {
		t.Fatalf("unexpected hash field %q", repo.saved[0].EventField)
	}
	if repo.saved[0].ObservedValue != strings.ToLower(hash) {
		t.Fatalf("unexpected normalized hash %q", repo.saved[0].ObservedValue)
	}
}

func TestNormaliseDomainAndSHA256RejectInvalidValues(t *testing.T) {
	t.Parallel()

	if got := normaliseDomain(" telemetry-sync.example. "); got != "telemetry-sync.example" {
		t.Fatalf("unexpected domain normalization %q", got)
	}
	if got := normaliseDomain("not a domain"); got != "" {
		t.Fatalf("expected invalid domain rejection, got %q", got)
	}
	if got := normaliseDomain("localhost"); got != "" {
		t.Fatalf("expected single-label domain rejection, got %q", got)
	}

	validHash := strings.Repeat("a", 64)
	if got := normaliseSHA256(strings.ToUpper(validHash)); got != validHash {
		t.Fatalf("unexpected hash normalization %q", got)
	}
	if got := normaliseSHA256("deadbeef"); got != "" {
		t.Fatalf("expected short hash rejection, got %q", got)
	}
}

func TestEngineInvalidatesCacheWhenIntelligenceRevisionChanges(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		revision: time.Unix(10, 0).UTC(),
		indicators: []Indicator{
			{
				ID:               "ioc-revision-a",
				SourceID:         "source-a",
				SourceConfidence: 80,
				Confidence:       90,
			},
		},
	}
	engine := New(repo)
	event := telemetry.Event{
		EventID:   "evt-revision-a",
		Timestamp: time.Now().UTC(),
		Network: &telemetry.Network{
			DestinationIP: "198.51.100.66",
		},
	}

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("first enrichment: %v", err)
	}
	if repo.lookups != 1 {
		t.Fatalf("expected first repository lookup, got %d", repo.lookups)
	}

	event.EventID = "evt-revision-b"
	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("cached enrichment: %v", err)
	}
	if repo.lookups != 1 {
		t.Fatalf("expected cached lookup, got %d repository queries", repo.lookups)
	}

	repo.revision = repo.revision.Add(time.Second)
	repo.indicators = nil
	event.EventID = "evt-revision-c"
	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("post-revision enrichment: %v", err)
	}
	if repo.lookups != 2 {
		t.Fatalf("expected cache invalidation after revision change, got %d lookups", repo.lookups)
	}
}
