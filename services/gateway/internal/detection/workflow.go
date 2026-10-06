package detection

import "fmt"

var findingTransitions = map[string]map[string]struct{}{
	"new": {
		"triaged": {},
	},
	"triaged": {
		"investigating":   {},
		"false_positive":  {},
		"benign_expected": {},
		"duplicate":       {},
	},
	"investigating": {
		"confirmed":       {},
		"false_positive":  {},
		"benign_expected": {},
		"duplicate":       {},
	},
	"confirmed": {
		"contained": {},
		"closed":    {},
	},
	"contained": {
		"closed": {},
	},
}

func IsFindingStatus(status string) bool {
	switch status {
	case "new", "triaged", "investigating", "false_positive", "benign_expected", "duplicate", "confirmed", "contained", "closed":
		return true
	default:
		return false
	}
}

func ValidateFindingTransition(from, to string) error {
	if !IsFindingStatus(from) {
		return fmt.Errorf("unknown current finding status %q", from)
	}
	if !IsFindingStatus(to) {
		return fmt.Errorf("unknown target finding status %q", to)
	}
	if from == to {
		return fmt.Errorf("finding is already %q", to)
	}

	allowed, ok := findingTransitions[from]
	if !ok {
		return fmt.Errorf("finding status %q is terminal", from)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("transition from %q to %q is not allowed", from, to)
	}

	return nil
}
