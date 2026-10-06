package detection

import (
	"context"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type ServiceAccountLoginRule struct{}

func NewServiceAccountLoginRule() *ServiceAccountLoginRule {
	return &ServiceAccountLoginRule{}
}

func (r *ServiceAccountLoginRule) ID() string {
	return ServiceAccountLoginDetectionID
}

func (r *ServiceAccountLoginRule) Evaluate(_ context.Context, event telemetry.Event) (*Finding, error) {
	if event.Event.Category != "authentication" ||
		event.Event.Action != "login" ||
		event.Event.Outcome != "success" ||
		event.Actor == nil ||
		strings.ToLower(strings.TrimSpace(event.Actor.Type)) != "service_account" {
		return nil, nil
	}

	identityID := strings.TrimSpace(event.Actor.ID)
	if identityID == "" {
		return nil, nil
	}

	sourceIP := ""
	if event.Network != nil {
		sourceIP = strings.TrimSpace(event.Network.SourceIP)
	}

	return &Finding{
		ID:               findingID(ServiceAccountLoginDetectionID, identityID, event.EventID),
		DetectionID:      ServiceAccountLoginDetectionID,
		DetectionVersion: ServiceAccountLoginVersion,
		DedupKey:         dedupKey(ServiceAccountLoginDetectionID, identityID, event.EventID),
		Title:            "Interactive Login Using a Service Account",
		Severity:         "high",
		Confidence:       90,
		Status:           "new",
		FirstObservedAt:  event.Timestamp.UTC(),
		LastObservedAt:   event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:        []string{event.EventID},
			IdentityID:      identityID,
			ActorType:       "service_account",
			SourceIP:        sourceIP,
			TerminalEventID: event.EventID,
		},
		EventIDs: []string{event.EventID},
	}, nil
}
