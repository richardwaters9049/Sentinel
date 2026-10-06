package detection

import "testing"

func TestValidateFindingTransitionAllowsExpectedWorkflow(t *testing.T) {
	t.Parallel()

	valid := [][2]string{
		{"new", "triaged"},
		{"triaged", "investigating"},
		{"triaged", "false_positive"},
		{"investigating", "confirmed"},
		{"confirmed", "contained"},
		{"confirmed", "closed"},
		{"contained", "closed"},
	}

	for _, transition := range valid {
		if err := ValidateFindingTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("expected %q -> %q to be valid: %v", transition[0], transition[1], err)
		}
	}
}

func TestValidateFindingTransitionRejectsInvalidWorkflow(t *testing.T) {
	t.Parallel()

	invalid := [][2]string{
		{"new", "confirmed"},
		{"new", "new"},
		{"false_positive", "triaged"},
		{"closed", "investigating"},
		{"unknown", "triaged"},
		{"new", "unknown"},
	}

	for _, transition := range invalid {
		if err := ValidateFindingTransition(transition[0], transition[1]); err == nil {
			t.Fatalf("expected %q -> %q to be rejected", transition[0], transition[1])
		}
	}
}
