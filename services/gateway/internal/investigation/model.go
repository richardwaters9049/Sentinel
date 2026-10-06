package investigation

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInvestigation = errors.New("invalid investigation")
	ErrInvalidTransition    = errors.New("invalid investigation status transition")
)

type Record struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	OwnerID     *string   `json:"owner_id,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Note struct {
	ID              int64     `json:"id"`
	InvestigationID string    `json:"investigation_id"`
	ActorID         string    `json:"actor_id"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
}

func ValidateCreate(title, priority string) error {
	if strings.TrimSpace(title) == "" || len(strings.TrimSpace(title)) > 200 {
		return ErrInvalidInvestigation
	}
	switch strings.TrimSpace(priority) {
	case "", "low", "medium", "high", "critical":
		return nil
	default:
		return ErrInvalidInvestigation
	}
}

func ValidateStatusTransition(from, to string) error {
	if from == to {
		return ErrInvalidTransition
	}

	switch from {
	case "open":
		if to == "investigating" || to == "closed" {
			return nil
		}
	case "investigating":
		if to == "contained" || to == "closed" {
			return nil
		}
	case "contained":
		if to == "closed" || to == "investigating" {
			return nil
		}
	case "closed":
		return ErrInvalidTransition
	}

	return ErrInvalidTransition
}
