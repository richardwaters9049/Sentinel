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

func TestScenarioEventsIncludesExpandedOTInventory(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 18, 30, 0, 0, time.UTC)

	sequence, err := scenarioEvents("ot-change-sequence", start)
	if err != nil {
		t.Fatalf("OT change sequence: %v", err)
	}
	if len(sequence) != 2 {
		t.Fatalf("expected two correlated OT events, got %d", len(sequence))
	}
	firstAsset, ok := sequence[0]["asset"].(map[string]interface{})
	if !ok || firstAsset["hostname"] != "plc-sim-02" {
		t.Fatalf("expected plc-sim-02 mode-change asset, got %#v", sequence[0]["asset"])
	}
	secondAsset, ok := sequence[1]["asset"].(map[string]interface{})
	if !ok || secondAsset["hostname"] != "plc-sim-02" {
		t.Fatalf("expected plc-sim-02 parameter-change asset, got %#v", sequence[1]["asset"])
	}
	firstEvent, ok := sequence[0]["event"].(map[string]interface{})
	if !ok || firstEvent["action"] != "controller_mode_change" {
		t.Fatalf("expected controller_mode_change, got %#v", sequence[0]["event"])
	}
	secondEvent, ok := sequence[1]["event"].(map[string]interface{})
	if !ok || secondEvent["action"] != "parameter_change" {
		t.Fatalf("expected parameter_change, got %#v", sequence[1]["event"])
	}

	historian, err := scenarioEvents("ot-historian-read-baseline", start)
	if err != nil {
		t.Fatalf("historian baseline: %v", err)
	}
	historianAsset, ok := historian[0]["asset"].(map[string]interface{})
	if !ok || historianAsset["hostname"] != "historian-01" || historianAsset["zone"] != "dmz" {
		t.Fatalf("unexpected historian asset: %#v", historian[0]["asset"])
	}

	sensor, err := scenarioEvents("ot-sensor-telemetry-baseline", start)
	if err != nil {
		t.Fatalf("sensor baseline: %v", err)
	}
	sensorAsset, ok := sensor[0]["asset"].(map[string]interface{})
	if !ok || sensorAsset["hostname"] != "sensor-sim-01" || sensorAsset["zone"] != "ot" {
		t.Fatalf("unexpected sensor asset: %#v", sensor[0]["asset"])
	}
}

func TestScenarioEventsIncludesIntelligenceIOCMatch(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	events, err := scenarioEvents("intel-ioc-match", start)
	if err != nil {
		t.Fatalf("intel IOC scenario: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}

	network, ok := events[0]["network"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected network payload, got %#v", events[0]["network"])
	}
	if network["destination_ip"] != "198.51.100.66" {
		t.Fatalf("unexpected IOC destination: %#v", network["destination_ip"])
	}

	labels, ok := events[0]["labels"].(map[string]string)
	if !ok || labels["scenario"] != "intel-ioc-match" || labels["simulated"] != "true" {
		t.Fatalf("unexpected intelligence labels: %#v", events[0]["labels"])
	}
}

func TestScenarioEventsIncludesDomainAndSHA256IntelligenceMatches(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 21, 0, 0, 0, time.UTC)

	domainEvents, err := scenarioEvents("intel-domain-match", start)
	if err != nil {
		t.Fatalf("domain intelligence scenario: %v", err)
	}
	if len(domainEvents) != 1 {
		t.Fatalf("expected one domain event, got %d", len(domainEvents))
	}
	domainLabels, ok := domainEvents[0]["labels"].(map[string]string)
	if !ok || domainLabels["dns.query"] != "telemetry-sync.example" {
		t.Fatalf("unexpected domain labels: %#v", domainEvents[0]["labels"])
	}

	hashEvents, err := scenarioEvents("intel-sha256-match", start)
	if err != nil {
		t.Fatalf("sha256 intelligence scenario: %v", err)
	}
	if len(hashEvents) != 1 {
		t.Fatalf("expected one sha256 event, got %d", len(hashEvents))
	}
	hashLabels, ok := hashEvents[0]["labels"].(map[string]string)
	if !ok ||
		hashLabels["file.sha256"] !=
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected sha256 labels: %#v", hashEvents[0]["labels"])
	}
}

func TestScenarioEventsIncludesFindingEnrichmentMatch(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 21, 15, 0, 0, time.UTC)
	events, err := scenarioEvents("intel-finding-match", start)
	if err != nil {
		t.Fatalf("finding enrichment scenario: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}

	network, ok := events[0]["network"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected network payload, got %#v", events[0]["network"])
	}
	if network["source_ip"] != "10.10.0.25" ||
		network["destination_zone"] != "ot" ||
		network["destination_port"] != 502 {
		t.Fatalf("unexpected finding-match network context: %#v", network)
	}

	asset, ok := events[0]["asset"].(map[string]interface{})
	if !ok || asset["zone"] != "corporate" {
		t.Fatalf("expected corporate source asset, got %#v", events[0]["asset"])
	}
}

func TestScenarioEventsIncludesBehaviouralAnalyticsFixtures(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)

	normal, err := scenarioEvents("behaviour-normal", start)
	if err != nil {
		t.Fatalf("normal behaviour scenario: %v", err)
	}
	if len(normal) != 1 {
		t.Fatalf("expected one normal behaviour event, got %d", len(normal))
	}
	normalNetwork := normal[0]["network"].(map[string]interface{})
	if normalNetwork["destination_zone"] != "corporate" {
		t.Fatalf("unexpected normal destination zone: %#v", normalNetwork)
	}

	anomaly, err := scenarioEvents("behaviour-anomaly", start)
	if err != nil {
		t.Fatalf("anomaly behaviour scenario: %v", err)
	}
	if len(anomaly) != 1 {
		t.Fatalf("expected one anomaly behaviour event, got %d", len(anomaly))
	}
	actor := anomaly[0]["actor"].(map[string]interface{})
	network := anomaly[0]["network"].(map[string]interface{})
	event := anomaly[0]["event"].(map[string]interface{})
	if actor["type"] != "service_account" {
		t.Fatalf("expected service account anomaly fixture, got %#v", actor)
	}
	if network["destination_zone"] != "ot" || network["destination_port"] != 502 {
		t.Fatalf("unexpected anomaly network context: %#v", network)
	}
	if event["outcome"] != "failure" {
		t.Fatalf("expected failed authentication anomaly, got %#v", event)
	}
}
