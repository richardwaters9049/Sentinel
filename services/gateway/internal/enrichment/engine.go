package enrichment

import (
	"context"
	"fmt"
	"net"
	"regexp"
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
	IntelligenceRevision(context.Context) (time.Time, error)
	SaveEventEnrichments(context.Context, string, []Match) error
}

type cacheEntry struct {
	indicators []Indicator
	expiresAt  time.Time
	revision   time.Time
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
	if e == nil || e.repository == nil {
		return nil
	}

	type candidate struct {
		indicatorType string
		field         string
		value         string
		normalise     func(string) string
	}

	candidates := make([]candidate, 0, 4)
	if event.Network != nil {
		candidates = append(candidates,
			candidate{
				indicatorType: "ip",
				field:         "network.source_ip",
				value:         event.Network.SourceIP,
				normalise:     normaliseIP,
			},
			candidate{
				indicatorType: "ip",
				field:         "network.destination_ip",
				value:         event.Network.DestinationIP,
				normalise:     normaliseIP,
			},
		)
	}

	if event.Labels != nil {
		candidates = append(candidates,
			candidate{
				indicatorType: "domain",
				field:         "labels.dns.query",
				value:         event.Labels["dns.query"],
				normalise:     normaliseDomain,
			},
			candidate{
				indicatorType: "sha256",
				field:         "labels.file.sha256",
				value:         event.Labels["file.sha256"],
				normalise:     normaliseSHA256,
			},
		)
	}

	matches := make([]Match, 0, 4)
	seen := make(map[string]struct{})

	for _, candidate := range candidates {
		value := candidate.normalise(candidate.value)
		if value == "" {
			continue
		}

		indicators, err := e.lookup(
			ctx,
			candidate.indicatorType,
			value,
			event.Timestamp,
		)
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

	revision, err := e.repository.IntelligenceRevision(ctx)
	if err != nil {
		return nil, fmt.Errorf("read intelligence revision: %w", err)
	}

	e.mu.Lock()
	if entry, ok := e.cache[key]; ok &&
		now.Before(entry.expiresAt) &&
		entry.revision.Equal(revision) {
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
		revision:   revision,
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

func normaliseDomain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimSuffix(value, ".")
	if value == "" || len(value) > 253 || strings.Contains(value, " ") {
		return ""
	}

	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return ""
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return ""
		}
		for index, r := range label {
			isLetter := r >= 'a' && r <= 'z'
			isDigit := r >= '0' && r <= '9'
			if isLetter || isDigit || (r == '-' && index > 0 && index < len(label)-1) {
				continue
			}
			return ""
		}
	}
	return value
}

var sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func normaliseSHA256(value string) string {
	value = strings.TrimSpace(value)
	if !sha256Pattern.MatchString(value) {
		return ""
	}
	return strings.ToLower(value)
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
