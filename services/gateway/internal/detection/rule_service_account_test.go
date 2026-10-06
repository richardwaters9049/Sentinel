package detection

import (
	"context"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

func TestServiceAccountLoginRuleCreatesFinding(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_service_login",
		Timestamp: time.Date(2026, 10, 6, 22, 0, 0, 0, time.UTC),
		Actor: &telemetry.Actor{
			ID:   "svc-backup",
			Type: "service_account",
			Name: "Backup Service",
		},
		Event: telemetry.EventDetails{
			Category: "authentication",
			Action:   "login",
			Outcome:  "success",
		},
		Network: &telemetry.Network{SourceIP: "10.10.10.44"},
	}

	finding, err := NewServiceAccountLoginRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("expected rule to evaluate, got %v", err)
	}
	if finding == nil {
		t.Fatal("expected finding")
	}
	if finding.DetectionID != ServiceAccountLoginDetectionID {
		t.Fatalf("expected %q, got %q", ServiceAccountLoginDetectionID, finding.DetectionID)
	}
	if finding.Confidence != 90 || finding.Severity != "high" {
		t.Fatalf("unexpected confidence/severity: %d/%s", finding.Confidence, finding.Severity)
	}
	if finding.Evidence.ActorType != "service_account" {
		t.Fatalf("expected service_account evidence, got %q", finding.Evidence.ActorType)
	}
	if len(finding.EventIDs) != 1 || finding.EventIDs[0] != event.EventID {
		t.Fatalf("unexpected event evidence: %#v", finding.EventIDs)
	}
}

func TestServiceAccountLoginRuleIgnoresHumanLogin(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_human_login",
		Timestamp: time.Now().UTC(),
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
	}

	finding, err := NewServiceAccountLoginRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("expected rule to evaluate, got %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding, got %#v", finding)
	}
}
