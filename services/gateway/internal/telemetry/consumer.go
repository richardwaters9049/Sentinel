package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrMalformedPersistedEvent = errors.New("malformed persisted telemetry event")
	ErrUnsupportedSchema       = errors.New("unsupported telemetry schema version")
)

type EventStore interface {
	SaveEvent(context.Context, Event) (bool, error)
}

type EventProcessor interface {
	Process(context.Context, Event) error
}

type permanentError struct {
	err error
}

func (e permanentError) Error() string {
	return e.err.Error()
}

func (e permanentError) Unwrap() error {
	return e.err
}

func (e permanentError) Permanent() bool {
	return true
}

func PersistenceHandler(
	store EventStore,
	processor EventProcessor,
) func(context.Context, []byte) error {
	return func(ctx context.Context, payload []byte) error {
		if store == nil {
			return fmt.Errorf("event store is not initialised")
		}

		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return permanentError{
				err: fmt.Errorf("%w: %v", ErrMalformedPersistedEvent, err),
			}
		}

		if event.SchemaVersion != SchemaVersion {
			return permanentError{
				err: fmt.Errorf(
					"%w %q",
					ErrUnsupportedSchema,
					event.SchemaVersion,
				),
			}
		}

		inserted, err := store.SaveEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("persist telemetry event %s: %w", event.EventID, err)
		}

		if !inserted || processor == nil {
			return nil
		}

		if err := processor.Process(ctx, event); err != nil {
			return fmt.Errorf("process persisted telemetry event %s: %w", event.EventID, err)
		}

		return nil
	}
}
