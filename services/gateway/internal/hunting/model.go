package hunting

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidHunt = errors.New("invalid hunt")

type Query struct {
	Category         string            `json:"category,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	Action           string            `json:"action,omitempty"`
	Actions          []string          `json:"actions,omitempty"`
	Outcome          string            `json:"outcome,omitempty"`
	Outcomes         []string          `json:"outcomes,omitempty"`
	AssetID          string            `json:"asset_id,omitempty"`
	IdentityID       string            `json:"identity_id,omitempty"`
	SourceIP         string            `json:"source_ip,omitempty"`
	DestinationIP    string            `json:"destination_ip,omitempty"`
	SourceZones      []string          `json:"source_zones,omitempty"`
	DestinationZone  string            `json:"destination_zone,omitempty"`
	DestinationZones []string          `json:"destination_zones,omitempty"`
	DestinationPorts []int             `json:"destination_ports,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	From             *time.Time        `json:"from,omitempty"`
	To               *time.Time        `json:"to,omitempty"`
	LastMinutes      int               `json:"last_minutes,omitempty"`
	Limit            int               `json:"limit,omitempty"`
}

type Definition struct {
	ID          string    `json:"id"`
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Hypothesis  string    `json:"hypothesis"`
	Query       Query     `json:"query"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type VersionRecord struct {
	HuntID      string    `json:"hunt_id"`
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Hypothesis  string    `json:"hypothesis"`
	Query       Query     `json:"query"`
	ChangedBy   string    `json:"changed_by"`
	ChangedAt   time.Time `json:"changed_at"`
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
	if query.LastMinutes < 0 || query.LastMinutes > 10080 {
		return ErrInvalidHunt
	}
	if query.LastMinutes > 0 && (query.From != nil || query.To != nil) {
		return ErrInvalidHunt
	}
	if query.From != nil && query.To != nil && query.From.After(*query.To) {
		return ErrInvalidHunt
	}
	if len(query.Categories) > 20 ||
		len(query.Actions) > 20 ||
		len(query.Outcomes) > 20 ||
		len(query.SourceZones) > 20 ||
		len(query.DestinationZones) > 20 ||
		len(query.DestinationPorts) > 50 ||
		len(query.Labels) > 20 {
		return ErrInvalidHunt
	}
	for _, port := range query.DestinationPorts {
		if port < 1 || port > 65535 {
			return ErrInvalidHunt
		}
	}
	for key, value := range query.Labels {
		if strings.TrimSpace(key) == "" || len(key) > 64 || len(value) > 256 {
			return ErrInvalidHunt
		}
	}
	return nil
}
