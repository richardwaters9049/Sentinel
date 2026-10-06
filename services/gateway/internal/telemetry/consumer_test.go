package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type recordingStore struct {
	event    Event
	inserted bool
	err      error
}

func (s *recordingStore) SaveEvent(_ context.Context, event Event) (bool, error) {
	s.event = event
	return s.inserted, s.err
}

func TestPersistenceHandlerStoresEvent(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inserted: true}
	handler := PersistenceHandler(store)

	event, err := Normalize(
		minimalRequest(time.Now().UTC()),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("expected handler to persist event, got %v", err)
	}

	if store.event.EventID != event.EventID {
		t.Fatalf("expected event id %q, got %q", event.EventID, store.event.EventID)
	}
}

func TestPersistenceHandlerRejectsUnknownSchema(t *testing.T) {
	t.Parallel()

	store := &recordingStore{}
	handler := PersistenceHandler(store)

	payload := []byte(`{"event_id":"evt_test","schema_version":"99.0.0"}`)
	err := handler(context.Background(), payload)
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("expected ErrUnsupportedSchema, got %v", err)
	}

	permanent, ok := err.(interface{ Permanent() bool })
	if !ok || !permanent.Permanent() {
		t.Fatalf("expected unsupported schema error to be permanent, got %T", err)
	}
}

func TestPersistenceHandlerReturnsStoreFailure(t *testing.T) {
	t.Parallel()

	store := &recordingStore{err: errors.New("database unavailable")}
	handler := PersistenceHandler(store)

	event, err := Normalize(
		minimalRequest(time.Now().UTC()),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	if err := handler(context.Background(), payload); err == nil {
		t.Fatal("expected storage error")
	}
}
