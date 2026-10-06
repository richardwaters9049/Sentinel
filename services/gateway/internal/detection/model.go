package detection

import "time"

const (
	AuthBurstDetectionID      = "DET-AUTH-001"
	AuthBurstDetectionVersion = 1
)

type Evidence struct {
	EventIDs       []string `json:"event_ids"`
	FailureCount   int      `json:"failure_count"`
	IdentityID     string   `json:"identity_id"`
	SourceIP       string   `json:"source_ip"`
	WindowSeconds  int      `json:"window_seconds"`
	SuccessEventID string   `json:"success_event_id"`
}

type Finding struct {
	ID               string
	DetectionID      string
	DetectionVersion int
	DedupKey         string
	Title            string
	Severity         string
	Confidence       int
	Status           string
	FirstObservedAt  time.Time
	LastObservedAt   time.Time
	Evidence         Evidence
	EventIDs         []string
}

type AuthFailure struct {
	EventID   string
	Timestamp time.Time
}
