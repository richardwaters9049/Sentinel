package enrichment

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

const (
	defaultCacheTTL = 5 * time.Minute
	maxCacheEntries = 1024
)

type Repository interface {
	LookupIndicators(context.Context, string, string, time.Time) ([]Indicator, error)
	SaveEventEnrichments(context.Context, string, []Match) error
}

type cacheEntry struct {
	indicators []Indicator
	expiresAt  time.Time
}

type Engine struct {
	repository Repository
	cacheTTL   time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry
}

func New(repository Repository) *Engine {
	return &Engine{
		repository: repository,
		cacheTTL:   defaultCacheTTL,
		cache:      make(map[string]cacheEntry),
	}
}

func (e *Engine) Process(ctx context.Context, event telemetry.Event) error {
	if e == nil || e.repository == nil || event.Network == nil {
		return nil
	}

	candidates := []struct {
		field string
		value string
	}{
		{field: "network.source_ip", value: event.Network.SourceIP},
		{field: "network.destination_ip", value: event.Network.DestinationIP},
	}

	matches := make([]Match, 0, 2)
	seen := make(map[string]struct{})

	for _, candidate := range candidates {
		value := normaliseIP(candidate.value)
		if value == "" {
			continue
		}

		indicators, err := e.lookup(ctx, "ip", value, event.Timestamp)
		if err != nil {
			return fmt.Errorf("lookup %s enrichment for %s: %w", candidate.field, event.EventID, err)
		}

		for _, indicator := range indicators {
			key := candidate.field + ":" + indicator.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			matches = append(matches, Match{
				EventID:             event.EventID,
				IndicatorID:         indicator.ID,
				SourceID:            indicator.SourceID,
				EventField:          candidate.field,
				ObservedValue:       value,
				SourceConfidence:    indicator.SourceConfidence,
				IndicatorConfidence: indicator.Confidence,
				EffectiveConfidence: effectiveConfidence(
					indicator.SourceConfidence,
					indicator.Confidence,
				),
				Provenance: indicator.Provenance,
				MatchedAt:  time.Now().UTC(),
			})
		}
	}

	if len(matches) == 0 {
		return nil
	}

	if err := e.repository.SaveEventEnrichments(ctx, event.EventID, matches); err != nil {
		return fmt.Errorf("persist event enrichments for %s: %w", event.EventID, err)
	}

	return nil
}

func (e *Engine) lookup(
	ctx context.Context,
	indicatorType string,
	value string,
	observedAt time.Time,
) ([]Indicator, error) {
	key := indicatorType + ":" + strings.ToLower(strings.TrimSpace(value))
	now := time.Now().UTC()

	e.mu.Lock()
	if entry, ok := e.cache[key]; ok && now.Before(entry.expiresAt) {
		result := append([]Indicator(nil), entry.indicators...)
		e.mu.Unlock()
		return result, nil
	}
	e.mu.Unlock()

	indicators, err := e.repository.LookupIndicators(
		ctx,
		indicatorType,
		value,
		observedAt.UTC(),
	)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	if len(e.cache) >= maxCacheEntries {
		for existingKey := range e.cache {
			delete(e.cache, existingKey)
			break
		}
	}
	e.cache[key] = cacheEntry{
		indicators: append([]Indicator(nil), indicators...),
		expiresAt:  now.Add(e.cacheTTL),
	}
	e.mu.Unlock()

	return indicators, nil
}

func normaliseIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}

func effectiveConfidence(sourceConfidence, indicatorConfidence int) int {
	if sourceConfidence < 0 {
		sourceConfidence = 0
	}
	if sourceConfidence > 100 {
		sourceConfidence = 100
	}
	if indicatorConfidence < 0 {
		indicatorConfidence = 0
	}
	if indicatorConfidence > 100 {
		indicatorConfidence = 100
	}

	return (sourceConfidence*indicatorConfidence + 50) / 100
}
