package behaviour

import "time"

type Explanation struct {
	Feature   string  `json:"feature"`
	Observed  float64 `json:"observed"`
	Baseline  float64 `json:"baseline"`
	Deviation float64 `json:"deviation"`
	Message   string  `json:"message"`
}

type BaselineContext struct {
	PriorEvents60m            int     `json:"prior_events_60m"`
	PriorEvents24h            int     `json:"prior_events_24h"`
	UniqueDestinationIPs24h   int     `json:"unique_destination_ips_24h"`
	UniqueDestinationPorts24h int     `json:"unique_destination_ports_24h"`
	AuthFailures60m           int     `json:"auth_failures_60m"`
	OTEvents24h               int     `json:"ot_events_24h"`
	EventRate60m              float64 `json:"event_rate_60m"`
	DestinationDiversity24h   float64 `json:"destination_diversity_24h"`
	AuthFailureRate60m        float64 `json:"auth_failure_rate_60m"`
	OTActivityRate24h         float64 `json:"ot_activity_rate_24h"`
}

type ScoreRequest struct {
	EventID         string          `json:"event_id"`
	EntityID        string          `json:"entity_id"`
	EntityType      string          `json:"entity_type"`
	Timestamp       time.Time       `json:"timestamp"`
	Category        string          `json:"category"`
	Action          string          `json:"action"`
	Outcome         string          `json:"outcome"`
	SourceZone      string          `json:"source_zone"`
	DestinationZone string          `json:"destination_zone"`
	DestinationPort int             `json:"destination_port"`
	ActorType       string          `json:"actor_type"`
	Baseline        BaselineContext `json:"baseline"`
}

type Score struct {
	EventID      string          `json:"event_id"`
	EntityID     string          `json:"entity_id"`
	EntityType   string          `json:"entity_type"`
	Baseline     BaselineContext `json:"baseline"`
	ModelVersion string          `json:"model_version"`
	ModelKind    string          `json:"model_kind"`
	AnomalyScore int             `json:"anomaly_score"`
	Severity     string          `json:"severity"`
	Anomalous    bool            `json:"anomalous"`
	Threshold    int             `json:"threshold"`
	Explanations []Explanation   `json:"explanations"`
	ScoredAt     time.Time       `json:"scored_at,omitempty"`
}
