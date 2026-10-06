package investigation

import (
	"errors"
	"testing"
)

func TestValidateCreate(t *testing.T) {
	t.Parallel()

	if err := ValidateCreate("Investigate OT access", "high"); err != nil {
		t.Fatalf("expected valid investigation, got %v", err)
	}
}

func TestValidateCreateRejectsInvalidPriority(t *testing.T) {
	t.Parallel()

	if !errors.Is(ValidateCreate("Case", "urgent"), ErrInvalidInvestigation) {
		t.Fatal("expected invalid priority to be rejected")
	}
}

func TestValidateStatusTransition(t *testing.T) {
	t.Parallel()

	for _, transition := range [][2]string{
		{"open", "investigating"},
		{"investigating", "contained"},
		{"contained", "closed"},
		{"contained", "investigating"},
	} {
		if err := ValidateStatusTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("expected %s -> %s to be valid: %v", transition[0], transition[1], err)
		}
	}
}

func TestValidateStatusTransitionRejectsTerminalReopen(t *testing.T) {
	t.Parallel()

	if !errors.Is(ValidateStatusTransition("closed", "open"), ErrInvalidTransition) {
		t.Fatal("expected closed investigation to reject reopen")
	}
}
