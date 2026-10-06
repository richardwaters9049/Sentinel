package detection

import "time"

const (
	AuthBurstDetectionID           = "DET-AUTH-001"
	AuthBurstDetectionVersion      = 1
	ServiceAccountLoginDetectionID = "DET-AUTH-002"
	ServiceAccountLoginVersion     = 1
	CorporateToOTDetectionID       = "DET-NET-001"
	CorporateToOTDetectionVersion  = 1
)

type Evidence struct {
	EventIDs        []string `json:"event_ids"`
	FailureCount    int      `json:"failure_count,omitempty"`
	IdentityID      string   `json:"identity_id,omitempty"`
	ActorType       string   `json:"actor_type,omitempty"`
	SourceIP        string   `json:"source_ip,omitempty"`
	SourceZone      string   `json:"source_zone,omitempty"`
	DestinationIP   string   `json:"destination_ip,omitempty"`
	DestinationZone string   `json:"destination_zone,omitempty"`
	WindowSeconds   int      `json:"window_seconds,omitempty"`
	SuccessEventID  string   `json:"success_event_id,omitempty"`
	TerminalEventID string   `json:"terminal_event_id,omitempty"`
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
