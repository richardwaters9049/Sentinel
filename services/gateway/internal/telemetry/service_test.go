package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type recordingPublisher struct {
	eventID string
	payload []byte
	err     error
}

func (p *recordingPublisher) PublishTelemetry(_ context.Context, eventID string, payload []byte) error {
	p.eventID = eventID
	p.payload = append([]byte(nil), payload...)
	return p.err
}

func TestServiceIngestPublishesNormalisedEvent(t *testing.T) {
	t.Parallel()

	publisher := &recordingPublisher{}
	service := NewService(publisher)
	service.now = func() time.Time {
		return time.Date(2026, 10, 6, 22, 0, 0, 0, time.UTC)
	}

	event, err := service.Ingest(context.Background(), minimalRequest(
		time.Date(2026, 10, 6, 21, 59, 0, 0, time.UTC),
	))
	if err != nil {
		t.Fatalf("expected ingestion to succeed, got %v", err)
	}

	if publisher.eventID != event.EventID {
		t.Fatalf("expected publisher event id %q, got %q", event.EventID, publisher.eventID)
	}

	var published Event
	if err := json.Unmarshal(publisher.payload, &published); err != nil {
		t.Fatalf("decode published payload: %v", err)
	}
	if published.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, published.SchemaVersion)
	}
}

func TestServiceIngestDoesNotPublishInvalidEvent(t *testing.T) {
	t.Parallel()

	publisher := &recordingPublisher{}
	service := NewService(publisher)

	_, err := service.Ingest(context.Background(), IngestRequest{})
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("expected ErrInvalidEvent, got %v", err)
	}
	if len(publisher.payload) != 0 {
		t.Fatal("invalid event should not be published")
	}
}

func TestServiceIngestReturnsPublisherFailure(t *testing.T) {
	t.Parallel()

	publisher := &recordingPublisher{err: errors.New("queue unavailable")}
	service := NewService(publisher)

	_, err := service.Ingest(context.Background(), minimalRequest(time.Now().UTC()))
	if err == nil {
		t.Fatal("expected publisher error")
	}
}
