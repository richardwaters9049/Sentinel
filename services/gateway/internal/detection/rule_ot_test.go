package detection

import (
	"context"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

func TestOTParameterChangeRuleCreatesFinding(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_ot_parameter_001",
		Timestamp: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		Asset: &telemetry.Asset{
			ID:       "asset-plc-sim-01",
			Hostname: "plc-sim-01",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "parameter_change",
			Outcome:  "success",
		},
		Labels: map[string]string{
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "setpoint_change",
			"ot.safety_impact": "potential_process_impact",
			"ot.simulated":     "true",
		},
	}

	finding, err := NewOTParameterChangeRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}
	if finding == nil {
		t.Fatal("expected finding")
	}
	if finding.DetectionID != OTParameterChangeDetectionID {
		t.Fatalf("expected %q, got %q", OTParameterChangeDetectionID, finding.DetectionID)
	}
	if finding.Evidence.OTDeviceType != "plc" {
		t.Fatalf("expected plc evidence, got %#v", finding.Evidence)
	}
	if finding.Evidence.OTOperation != "setpoint_change" {
		t.Fatalf("unexpected operation %q", finding.Evidence.OTOperation)
	}
}

func TestOTParameterChangeRuleIgnoresNonOTAsset(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_ot_parameter_corp",
		Timestamp: time.Now().UTC(),
		Asset: &telemetry.Asset{
			ID:       "asset-corporate-01",
			Hostname: "employee-workstation-01",
			Zone:     "corporate",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "parameter_change",
		},
		Labels: map[string]string{"ot.device_type": "plc"},
	}

	finding, err := NewOTParameterChangeRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding, got %#v", finding)
	}
}

func TestOTUnauthorizedCommandRuleCreatesFinding(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_ot_command_001",
		Timestamp: time.Date(2026, 10, 7, 12, 1, 0, 0, time.UTC),
		Asset: &telemetry.Asset{
			ID:       "asset-engineering-ws-01",
			Hostname: "engineering-workstation-01",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "command_message",
			Outcome:  "observed",
		},
		Network: &telemetry.Network{
			SourceIP:        "10.30.0.20",
			DestinationIP:   "10.30.0.40",
			DestinationPort: 502,
			DestinationZone: "ot",
			Protocol:        "tcp",
		},
		Labels: map[string]string{
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "write_request",
			"ot.authorized":    "false",
			"ot.safety_impact": "potential_process_impact",
			"ot.simulated":     "true",
		},
	}

	finding, err := NewOTUnauthorizedCommandRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}
	if finding == nil {
		t.Fatal("expected finding")
	}
	if finding.DetectionID != OTUnauthorizedCommandID {
		t.Fatalf("expected %q, got %q", OTUnauthorizedCommandID, finding.DetectionID)
	}
	if finding.Severity != "critical" {
		t.Fatalf("expected critical severity, got %q", finding.Severity)
	}
	if finding.Evidence.OTAuthorization != "false" {
		t.Fatalf("expected authorization evidence, got %#v", finding.Evidence)
	}
}

func TestOTUnauthorizedCommandRuleIgnoresAuthorizedCommand(t *testing.T) {
	t.Parallel()

	event := telemetry.Event{
		EventID:   "evt_ot_command_allowed",
		Timestamp: time.Now().UTC(),
		Asset: &telemetry.Asset{
			ID:       "asset-hmi-01",
			Hostname: "hmi-01",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "command_message",
		},
		Labels: map[string]string{
			"ot.device_type": "plc",
			"ot.authorized":  "true",
		},
	}

	finding, err := NewOTUnauthorizedCommandRule().Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding, got %#v", finding)
	}
}
