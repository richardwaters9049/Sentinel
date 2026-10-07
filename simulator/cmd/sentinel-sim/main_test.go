package main

import (
	"testing"
	"time"
)

func TestScenarioEventsIncludesPhaseTwoCatalogueScenarios(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 6, 22, 0, 0, 0, time.UTC)

	serviceEvents, err := scenarioEvents("service-account-login", start)
	if err != nil {
		t.Fatalf("service-account scenario: %v", err)
	}
	if len(serviceEvents) != 1 {
		t.Fatalf("expected one service-account event, got %d", len(serviceEvents))
	}

	serviceActor, ok := serviceEvents[0]["actor"].(map[string]interface{})
	if !ok || serviceActor["type"] != "service_account" {
		t.Fatalf("expected service_account actor, got %#v", serviceEvents[0]["actor"])
	}

	networkEvents, err := scenarioEvents("it-to-ot-connection", start)
	if err != nil {
		t.Fatalf("IT-to-OT scenario: %v", err)
	}
	if len(networkEvents) != 1 {
		t.Fatalf("expected one IT-to-OT event, got %d", len(networkEvents))
	}

	asset, ok := networkEvents[0]["asset"].(map[string]interface{})
	if !ok || asset["zone"] != "corporate" {
		t.Fatalf("expected corporate source asset, got %#v", networkEvents[0]["asset"])
	}

	network, ok := networkEvents[0]["network"].(map[string]interface{})
	if !ok || network["destination_zone"] != "ot" {
		t.Fatalf("expected OT destination zone, got %#v", networkEvents[0]["network"])
	}
}

func TestScenarioEventsRejectsUnknownScenario(t *testing.T) {
	t.Parallel()

	if _, err := scenarioEvents("does-not-exist", time.Now().UTC()); err == nil {
		t.Fatal("expected unknown scenario to fail")
	}
}

func TestScenarioEventsIncludesPhaseFiveOTScenarios(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	baseline, err := scenarioEvents("ot-hmi-read-baseline", start)
	if err != nil {
		t.Fatalf("baseline OT scenario: %v", err)
	}
	if len(baseline) != 1 {
		t.Fatalf("expected one baseline OT event, got %d", len(baseline))
	}
	baselineLabels, ok := baseline[0]["labels"].(map[string]string)
	if !ok {
		t.Fatalf("expected string labels, got %#v", baseline[0]["labels"])
	}
	if baselineLabels["ot.authorized"] != "true" || baselineLabels["ot.simulated"] != "true" {
		t.Fatalf("unexpected baseline labels: %#v", baselineLabels)
	}

	parameter, err := scenarioEvents("ot-plc-parameter-change", start)
	if err != nil {
		t.Fatalf("parameter-change scenario: %v", err)
	}
	parameterEvent, ok := parameter[0]["event"].(map[string]interface{})
	if !ok || parameterEvent["action"] != "parameter_change" {
		t.Fatalf("expected parameter_change event, got %#v", parameter[0]["event"])
	}
	parameterLabels, ok := parameter[0]["labels"].(map[string]string)
	if !ok || parameterLabels["ot.device_type"] != "plc" {
		t.Fatalf("expected PLC metadata, got %#v", parameter[0]["labels"])
	}

	command, err := scenarioEvents("ot-unauthorized-command", start)
	if err != nil {
		t.Fatalf("unauthorized-command scenario: %v", err)
	}
	commandLabels, ok := command[0]["labels"].(map[string]string)
	if !ok || commandLabels["ot.authorized"] != "false" {
		t.Fatalf("expected unauthorized OT metadata, got %#v", command[0]["labels"])
	}
	if commandLabels["ot.safety_impact"] != "potential_process_impact" {
		t.Fatalf("expected safety annotation, got %#v", commandLabels)
	}
}
