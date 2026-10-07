package behaviour

import (
	"context"
	"fmt"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type Scorer interface {
	Score(context.Context, ScoreRequest) (Score, error)
}

type Store interface {
	SaveBehaviourScore(context.Context, Score) error
}

type Processor struct {
	scorer Scorer
	store  Store
}

func NewProcessor(scorer Scorer, store Store) *Processor {
	return &Processor{scorer: scorer, store: store}
}

func (p *Processor) Process(ctx context.Context, event telemetry.Event) error {
	if p == nil || p.scorer == nil || p.store == nil {
		return nil
	}

	entityID := event.Source.Collector
	if event.Asset != nil && strings.TrimSpace(event.Asset.ID) != "" {
		entityID = event.Asset.ID
	}
	if event.Actor != nil && strings.TrimSpace(event.Actor.ID) != "" {
		entityID = event.Actor.ID
	}
	if strings.TrimSpace(entityID) == "" {
		return nil
	}

	request := ScoreRequest{
		EventID:   event.EventID,
		EntityID:  entityID,
		Timestamp: event.Timestamp,
		Category:  event.Event.Category,
		Action:    event.Event.Action,
		Outcome:   event.Event.Outcome,
	}
	if event.Asset != nil {
		request.SourceZone = event.Asset.Zone
	}
	if event.Network != nil {
		request.DestinationZone = event.Network.DestinationZone
		request.DestinationPort = event.Network.DestinationPort
	}
	if event.Actor != nil {
		request.ActorType = event.Actor.Type
	}

	score, err := p.scorer.Score(ctx, request)
	if err != nil {
		return fmt.Errorf("score behavioural event %s: %w", event.EventID, err)
	}
	if err := p.store.SaveBehaviourScore(ctx, score); err != nil {
		return fmt.Errorf("persist behavioural score %s: %w", event.EventID, err)
	}
	return nil
}
