package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Publisher interface {
	PublishTelemetry(context.Context, string, []byte) error
}

type Service struct {
	publisher Publisher
	now       func() time.Time
}

func NewService(publisher Publisher) *Service {
	return &Service{
		publisher: publisher,
		now:       time.Now,
	}
}

func (s *Service) Ingest(ctx context.Context, request IngestRequest) (Event, error) {
	if s == nil || s.publisher == nil {
		return Event{}, fmt.Errorf("telemetry service is not initialised")
	}

	event, err := Normalize(request, s.now().UTC())
	if err != nil {
		return Event{}, err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return Event{}, fmt.Errorf("marshal normalized event: %w", err)
	}

	if err := s.publisher.PublishTelemetry(ctx, event.EventID, payload); err != nil {
		return Event{}, fmt.Errorf("publish normalized event: %w", err)
	}

	return event, nil
}
