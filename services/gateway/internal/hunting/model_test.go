package hunting

import (
	"errors"
	"testing"
	"time"
)

func TestValidateDefinition(t *testing.T) {
	t.Parallel()

	if err := ValidateDefinition("Suspicious auth", "Check unusual logins", Query{Limit: 100}); err != nil {
		t.Fatalf("expected valid hunt definition, got %v", err)
	}
}

func TestValidateDefinitionRejectsInvalidRange(t *testing.T) {
	t.Parallel()

	from := time.Now().UTC()
	to := from.Add(-time.Minute)

	err := ValidateDefinition("Bad range", "", Query{From: &from, To: &to})
	if !errors.Is(err, ErrInvalidHunt) {
		t.Fatalf("expected ErrInvalidHunt, got %v", err)
	}
}

func TestValidateQueryAllowsRicherOperators(t *testing.T) {
	t.Parallel()

	err := ValidateQuery(Query{
		Categories:       []string{"authentication", "network"},
		Actions:          []string{"login", "connection"},
		Outcomes:         []string{"success", "failure"},
		SourceZones:      []string{"corporate"},
		DestinationZones: []string{"ot"},
		DestinationPorts: []int{22, 443, 502},
		Labels:           map[string]string{"environment": "lab"},
		LastMinutes:      60,
		Limit:            250,
	})
	if err != nil {
		t.Fatalf("expected richer hunt query to be valid, got %v", err)
	}
}

func TestValidateQueryRejectsRelativeAndAbsoluteTimeTogether(t *testing.T) {
	t.Parallel()

	from := time.Now().UTC().Add(-time.Hour)
	if !errors.Is(ValidateQuery(Query{From: &from, LastMinutes: 30}), ErrInvalidHunt) {
		t.Fatal("expected mixed relative and absolute time filters to be rejected")
	}
}

func TestValidateQueryRejectsInvalidDestinationPort(t *testing.T) {
	t.Parallel()

	if !errors.Is(ValidateQuery(Query{DestinationPorts: []int{0}}), ErrInvalidHunt) {
		t.Fatal("expected invalid destination port to be rejected")
	}
}

func TestValidateQueryRejectsExcessiveLimit(t *testing.T) {
	t.Parallel()

	if !errors.Is(ValidateQuery(Query{Limit: 999}), ErrInvalidHunt) {
		t.Fatal("expected oversized hunt limit to be rejected")
	}
}
