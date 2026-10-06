package detection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type fakeRepository struct {
	failures      []AuthFailure
	queryErr      error
	createErr     error
	created       []Finding
	queryIdentity string
	querySourceIP string
	queryBefore   time.Time
	queryWindow   time.Duration
	queryLimit    int
}

func (f *fakeRepository) RecentAuthenticationFailures(
	_ context.Context,
	identityID string,
	sourceIP string,
	before time.Time,
	window time.Duration,
	limit int,
) ([]AuthFailure, error) {
	f.queryIdentity = identityID
	f.querySourceIP = sourceIP
	f.queryBefore = before
	f.queryWindow = window
	f.queryLimit = limit
	return f.failures, f.queryErr
}

func (f *fakeRepository) CreateFinding(_ context.Context, finding Finding) (bool, error) {
	if f.createErr != nil {
		return false, f.createErr
	}
	f.created = append(f.created, finding)
	return true, nil
}

func TestProcessCreatesAuthBurstFinding(t *testing.T) {
	t.Parallel()

	successTime := time.Date(2026, 10, 6, 22, 10, 0, 0, time.UTC)
	repo := &fakeRepository{
		failures: []AuthFailure{
			{EventID: "evt_fail_4", Timestamp: successTime.Add(-20 * time.Second)},
			{EventID: "evt_fail_2", Timestamp: successTime.Add(-40 * time.Second)},
			{EventID: "evt_fail_1", Timestamp: successTime.Add(-50 * time.Second)},
			{EventID: "evt_fail_3", Timestamp: successTime.Add(-30 * time.Second)},
		},
	}

	engine := New(repo)
	event := successfulLogin(successTime)

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("expected finding to be created, got %v", err)
	}

	if len(repo.created) != 1 {
		t.Fatalf("expected one finding, got %d", len(repo.created))
	}

	finding := repo.created[0]
	if finding.DetectionID != AuthBurstDetectionID {
		t.Fatalf("expected detection %q, got %q", AuthBurstDetectionID, finding.DetectionID)
	}
	if finding.Severity != "high" || finding.Confidence != 85 {
		t.Fatalf("unexpected severity/confidence: %s/%d", finding.Severity, finding.Confidence)
	}
	if finding.Evidence.FailureCount != 4 {
		t.Fatalf("expected four failures, got %d", finding.Evidence.FailureCount)
	}
	if got, want := len(finding.EventIDs), 5; got != want {
		t.Fatalf("expected %d evidence events, got %d", want, got)
	}
	if finding.EventIDs[0] != "evt_fail_1" || finding.EventIDs[4] != "evt_success" {
		t.Fatalf("expected chronological evidence ordering, got %#v", finding.EventIDs)
	}
	if repo.queryIdentity != "user-01" || repo.querySourceIP != "10.0.0.44" {
		t.Fatalf("unexpected correlation keys: %q %q", repo.queryIdentity, repo.querySourceIP)
	}
	if repo.queryWindow != 5*time.Minute || repo.queryLimit != 4 {
		t.Fatalf("unexpected detection query: %s limit=%d", repo.queryWindow, repo.queryLimit)
	}
}

func TestProcessIgnoresNonSuccessfulLogin(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	engine := New(repo)

	event := successfulLogin(time.Now().UTC())
	event.Event.Outcome = "failure"

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("expected failure event to be ignored, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no findings, got %d", len(repo.created))
	}
}

func TestProcessRequiresThreshold(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := &fakeRepository{
		failures: []AuthFailure{
			{EventID: "evt_1", Timestamp: now.Add(-time.Minute)},
			{EventID: "evt_2", Timestamp: now.Add(-45 * time.Second)},
			{EventID: "evt_3", Timestamp: now.Add(-30 * time.Second)},
		},
	}

	if err := New(repo).Process(context.Background(), successfulLogin(now)); err != nil {
		t.Fatalf("expected below-threshold event to succeed without finding, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no findings below threshold, got %d", len(repo.created))
	}
}

func TestProcessSkipsMissingCorrelationContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	engine := New(repo)
	event := successfulLogin(time.Now().UTC())
	event.Network = nil

	if err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("expected event without network context to be skipped, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no finding, got %d", len(repo.created))
	}
}

func TestProcessPropagatesRepositoryErrors(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{queryErr: errors.New("database unavailable")}
	err := New(repo).Process(context.Background(), successfulLogin(time.Now().UTC()))
	if err == nil {
		t.Fatal("expected query error")
	}
}

func successfulLogin(timestamp time.Time) telemetry.Event {
	return telemetry.Event{
		EventID:       "evt_success",
		SchemaVersion: telemetry.SchemaVersion,
		Timestamp:     timestamp,
		ReceivedAt:    timestamp,
		Source: telemetry.Source{
			Type:      "identity",
			Collector: "test",
		},
		Actor: &telemetry.Actor{
			ID:   "user-01",
			Type: "human",
			Name: "Demo User",
		},
		Event: telemetry.EventDetails{
			Category: "authentication",
			Action:   "login",
			Outcome:  "success",
		},
		Network: &telemetry.Network{
			SourceIP: "10.0.0.44",
		},
	}
}
