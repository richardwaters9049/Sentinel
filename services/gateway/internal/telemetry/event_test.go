package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeCreatesCanonicalEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 21, 0, 0, 0, time.UTC)
	req := IngestRequest{
		Timestamp: now.Add(-time.Minute),
		Source: Source{
			Type:      " Identity ",
			Vendor:    "Sentinel Sim",
			Collector: " collector-01 ",
		},
		Asset: &Asset{
			ID:       "asset-01",
			Hostname: " workstation-01 ",
			Zone:     " Corporate ",
		},
		Actor: &Actor{
			ID:   "user-01",
			Type: " Human ",
			Name: " Demo User ",
		},
		Event: EventDetails{
			Category: " Authentication ",
			Action:   " Login ",
			Outcome:  " Success ",
		},
		Network: &Network{
			SourceIP:        "10.0.0.10",
			DestinationIP:   "10.0.0.20",
			DestinationPort: 443,
			DestinationZone: " OT ",
			Protocol:        " TCP ",
		},
		Labels: map[string]string{
			" scenario ": " auth-demo ",
		},
	}

	event, err := Normalize(req, now)
	if err != nil {
		t.Fatalf("expected request to normalize, got %v", err)
	}

	if !strings.HasPrefix(event.EventID, "evt_") {
		t.Fatalf("expected generated event id, got %q", event.EventID)
	}
	if event.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, event.SchemaVersion)
	}
	if event.Source.Type != "identity" {
		t.Fatalf("expected normalized source type, got %q", event.Source.Type)
	}
	if event.Asset == nil || event.Asset.Zone != "corporate" {
		t.Fatalf("expected normalized asset zone, got %#v", event.Asset)
	}
	if event.Actor == nil || event.Actor.Type != "human" {
		t.Fatalf("expected normalized actor type, got %#v", event.Actor)
	}
	if event.Event.Category != "authentication" || event.Event.Action != "login" {
		t.Fatalf("unexpected normalized event: %#v", event.Event)
	}
	if event.Network == nil || event.Network.Protocol != "tcp" {
		t.Fatalf("expected normalized network protocol, got %#v", event.Network)
	}
	if event.Network.DestinationZone != "ot" {
		t.Fatalf("expected normalized destination zone, got %#v", event.Network)
	}
	if event.Labels["scenario"] != "auth-demo" {
		t.Fatalf("expected normalized label, got %#v", event.Labels)
	}
}

func TestNormalizePreservesProvidedEventID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	req := minimalRequest(now)
	req.EventID = "evt_fixture_001"

	event, err := Normalize(req, now)
	if err != nil {
		t.Fatalf("expected request to normalize, got %v", err)
	}

	if event.EventID != "evt_fixture_001" {
		t.Fatalf("expected provided event id, got %q", event.EventID)
	}
}

func TestNormalizeRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := Normalize(IngestRequest{}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected validation error")
	}

	message := err.Error()
	for _, expected := range []string{
		"timestamp is required",
		"source.type is required",
		"source.collector is required",
		"event.category is required",
		"event.action is required",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got %q", expected, message)
		}
	}
}

func TestNormalizeRejectsInvalidNetworkAddress(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	req := minimalRequest(now)
	req.Network = &Network{SourceIP: "999.999.1.1"}

	_, err := Normalize(req, now)
	if err == nil || !strings.Contains(err.Error(), "network.source_ip") {
		t.Fatalf("expected invalid source IP error, got %v", err)
	}
}

func TestNormalizeRejectsFutureTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	req := minimalRequest(now)
	req.Timestamp = now.Add(6 * time.Minute)

	_, err := Normalize(req, now)
	if err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("expected future timestamp error, got %v", err)
	}
}

func minimalRequest(timestamp time.Time) IngestRequest {
	return IngestRequest{
		Timestamp: timestamp,
		Source: Source{
			Type:      "identity",
			Collector: "test-collector",
		},
		Event: EventDetails{
			Category: "authentication",
			Action:   "login",
		},
	}
}
