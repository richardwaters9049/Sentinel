package behaviour

import "time"

type Explanation struct {
	Feature   string  `json:"feature"`
	Observed  float64 `json:"observed"`
	Baseline  float64 `json:"baseline"`
	Deviation float64 `json:"deviation"`
	Message   string  `json:"message"`
}

type ScoreRequest struct {
	EventID         string    `json:"event_id"`
	EntityID        string    `json:"entity_id"`
	Timestamp       time.Time `json:"timestamp"`
	Category        string    `json:"category"`
	Action          string    `json:"action"`
	Outcome         string    `json:"outcome"`
	SourceZone      string    `json:"source_zone"`
	DestinationZone string    `json:"destination_zone"`
	DestinationPort int       `json:"destination_port"`
	ActorType       string    `json:"actor_type"`
}

type Score struct {
	EventID      string        `json:"event_id"`
	EntityID     string        `json:"entity_id"`
	ModelVersion string        `json:"model_version"`
	ModelKind    string        `json:"model_kind"`
	AnomalyScore int           `json:"anomaly_score"`
	Severity     string        `json:"severity"`
	Anomalous    bool          `json:"anomalous"`
	Threshold    int           `json:"threshold"`
	Explanations []Explanation `json:"explanations"`
	ScoredAt     time.Time     `json:"scored_at,omitempty"`
}
