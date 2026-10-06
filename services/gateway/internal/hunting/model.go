package hunting

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidHunt = errors.New("invalid hunt")

type Query struct {
	Category        string     `json:"category,omitempty"`
	Action          string     `json:"action,omitempty"`
	Outcome         string     `json:"outcome,omitempty"`
	AssetID         string     `json:"asset_id,omitempty"`
	IdentityID      string     `json:"identity_id,omitempty"`
	SourceIP        string     `json:"source_ip,omitempty"`
	DestinationZone string     `json:"destination_zone,omitempty"`
	From            *time.Time `json:"from,omitempty"`
	To              *time.Time `json:"to,omitempty"`
	Limit           int        `json:"limit,omitempty"`
}

type Definition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Hypothesis  string    `json:"hypothesis"`
	Query       Query     `json:"query"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RunRecord struct {
	ID          int64     `json:"id"`
	HuntID      string    `json:"hunt_id"`
	ActorID     string    `json:"actor_id"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	ResultCount int       `json:"result_count"`
	Parameters  Query     `json:"parameters"`
}

type RunResult struct {
	RunID       int64       `json:"run_id"`
	HuntID      string      `json:"hunt_id"`
	ResultCount int         `json:"result_count"`
	Events      interface{} `json:"events"`
	ExecutedAt  time.Time   `json:"executed_at"`
}

func ValidateDefinition(name, hypothesis string, query Query) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidHunt
	}
	if len(strings.TrimSpace(name)) > 160 {
		return ErrInvalidHunt
	}
	if len(strings.TrimSpace(hypothesis)) > 2000 {
		return ErrInvalidHunt
	}
	return ValidateQuery(query)
}

func ValidateQuery(query Query) error {
	if query.Limit < 0 || query.Limit > 500 {
		return ErrInvalidHunt
	}
	if query.From != nil && query.To != nil && query.From.After(*query.To) {
		return ErrInvalidHunt
	}
	return nil
}
