package detection

import (
	"context"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

func TestCorporateToOTRuleCreatesFinding(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_itot_001",
		Timestamp: time.Date(2026, 10, 6, 22, 0, 0, 0, time.UTC),
		Asset: &telemetry.Asset{
			ID:       "asset-corp-01",
			Hostname: "employee-workstation-01",
			Zone:     "corporate",
		},
		Event: telemetry.EventDetails{
			Category: "network",
			Action:   "connection",
			Outcome:  "success",
		},
		Network: &telemetry.Network{
			SourceIP:        "10.10.0.20",
			DestinationIP:   "10.30.0.10",
			DestinationPort: 502,
			DestinationZone: "ot",
			Protocol:        "tcp",
		},
	}

	finding, err := NewCorporateToOTRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("expected rule to evaluate, got %v", err)
	}
	if finding == nil {
		t.Fatal("expected finding")
	}
	if finding.DetectionID != CorporateToOTDetectionID {
		t.Fatalf("expected %q, got %q", CorporateToOTDetectionID, finding.DetectionID)
	}
	if finding.Evidence.SourceZone != "corporate" || finding.Evidence.DestinationZone != "ot" {
		t.Fatalf("unexpected zone evidence: %#v", finding.Evidence)
	}
	if finding.Evidence.DestinationIP != "10.30.0.10" {
		t.Fatalf("unexpected destination IP %q", finding.Evidence.DestinationIP)
	}
}

func TestCorporateToOTRuleIgnoresOTToOTConnection(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_otot_001",
		Timestamp: time.Now().UTC(),
		Asset: &telemetry.Asset{
			ID:       "asset-ot-01",
			Hostname: "engineering-workstation-01",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "network",
			Action:   "connection",
		},
		Network: &telemetry.Network{DestinationZone: "ot"},
	}

	finding, err := NewCorporateToOTRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("expected rule to evaluate, got %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding, got %#v", finding)
	}
}
