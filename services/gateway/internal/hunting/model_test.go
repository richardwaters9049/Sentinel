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

func TestValidateQueryRejectsExcessiveLimit(t *testing.T) {
	t.Parallel()

	if !errors.Is(ValidateQuery(Query{Limit: 999}), ErrInvalidHunt) {
		t.Fatal("expected oversized hunt limit to be rejected")
	}
}
