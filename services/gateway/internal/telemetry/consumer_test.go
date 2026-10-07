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

type recordingProcessor struct {
	event Event
	calls int
	err   error
}

func (p *recordingProcessor) Process(_ context.Context, event Event) error {
	p.calls++
	p.event = event
	return p.err
}

func TestPersistenceHandlerStoresAndProcessesNewEvent(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inserted: true}
	processor := &recordingProcessor{}
	handler := PersistenceHandler(store, processor)

	event := normalizedFixture(t)
	payload := marshalEvent(t, event)

	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("expected handler to persist event, got %v", err)
	}

	if store.event.EventID != event.EventID {
		t.Fatalf("expected event id %q, got %q", event.EventID, store.event.EventID)
	}
	if processor.calls != 1 || processor.event.EventID != event.EventID {
		t.Fatalf("expected persisted event to be processed once")
	}
}

func TestPersistenceHandlerSkipsProcessorForDuplicateEvent(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inserted: false}
	processor := &recordingProcessor{}
	handler := PersistenceHandler(store, processor)

	if err := handler(context.Background(), marshalEvent(t, normalizedFixture(t))); err != nil {
		t.Fatalf("expected duplicate event to be harmless, got %v", err)
	}

	if processor.calls != 0 {
		t.Fatalf("expected duplicate event not to be processed, got %d calls", processor.calls)
	}
}

func TestPersistenceHandlerRejectsUnknownSchema(t *testing.T) {
	t.Parallel()

	store := &recordingStore{}
	handler := PersistenceHandler(store, nil)

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
	handler := PersistenceHandler(store, nil)

	if err := handler(context.Background(), marshalEvent(t, normalizedFixture(t))); err == nil {
		t.Fatal("expected storage error")
	}
}

func TestPersistenceHandlerReturnsProcessorFailure(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inserted: true}
	processor := &recordingProcessor{err: errors.New("detection unavailable")}
	handler := PersistenceHandler(store, processor)

	if err := handler(context.Background(), marshalEvent(t, normalizedFixture(t))); err == nil {
		t.Fatal("expected processor error")
	}
}

func normalizedFixture(t *testing.T) Event {
	t.Helper()

	event, err := Normalize(
		minimalRequest(time.Now().UTC()),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}
	return event
}

func marshalEvent(t *testing.T, event Event) []byte {
	t.Helper()

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return payload
}

type processorRecorder struct {
	name  string
	order *[]string
	err   error
}

func (p processorRecorder) Process(_ context.Context, _ Event) error {
	*p.order = append(*p.order, p.name)
	return p.err
}

func TestProcessorChainRunsInOrder(t *testing.T) {
	t.Parallel()

	var order []string
	chain := ProcessorChain{
		processorRecorder{name: "detection", order: &order},
		processorRecorder{name: "enrichment", order: &order},
	}

	if err := chain.Process(context.Background(), Event{EventID: "evt-chain"}); err != nil {
		t.Fatalf("process chain: %v", err)
	}

	if got, want := len(order), 2; got != want {
		t.Fatalf("expected %d processors, got %d", want, got)
	}
	if order[0] != "detection" || order[1] != "enrichment" {
		t.Fatalf("unexpected processor order: %#v", order)
	}
}

func TestProcessorChainStopsOnFailure(t *testing.T) {
	t.Parallel()

	var order []string
	chain := ProcessorChain{
		processorRecorder{name: "detection", order: &order, err: errors.New("boom")},
		processorRecorder{name: "enrichment", order: &order},
	}

	if err := chain.Process(context.Background(), Event{EventID: "evt-chain-error"}); err == nil {
		t.Fatal("expected processor failure")
	}
	if len(order) != 1 || order[0] != "detection" {
		t.Fatalf("expected chain to stop after failure, got %#v", order)
	}
}
