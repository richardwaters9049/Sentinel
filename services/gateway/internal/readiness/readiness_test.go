package readiness

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckerReadyWhenDependenciesPass(t *testing.T) {
	t.Parallel()

	checker := New(time.Second, map[string]CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})

	result := checker.Check(context.Background())

	if !result.Ready {
		t.Fatal("expected readiness check to pass")
	}

	if result.Dependencies["postgres"] != "ready" {
		t.Fatalf("expected postgres to be ready, got %q", result.Dependencies["postgres"])
	}
}

func TestCheckerFailsWhenDependencyFails(t *testing.T) {
	t.Parallel()

	checker := New(time.Second, map[string]CheckFunc{
		"postgres": func(context.Context) error { return errors.New("database unavailable") },
	})

	result := checker.Check(context.Background())

	if result.Ready {
		t.Fatal("expected readiness check to fail")
	}

	if result.Dependencies["postgres"] != "unavailable" {
		t.Fatalf("expected postgres to be unavailable, got %q", result.Dependencies["postgres"])
	}
}

func TestCheckerTimesOutSlowDependency(t *testing.T) {
	t.Parallel()

	checker := New(10*time.Millisecond, map[string]CheckFunc{
		"slow": func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})

	result := checker.Check(context.Background())

	if result.Ready {
		t.Fatal("expected timed-out dependency to fail readiness")
	}
}
