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
