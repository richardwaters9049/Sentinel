package detection

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

const (
	authFailureThreshold = 4
	authWindow           = 5 * time.Minute
)

type AuthBurstRule struct {
	repository Repository
}

func NewAuthBurstRule(repository Repository) *AuthBurstRule {
	return &AuthBurstRule{repository: repository}
}

func (r *AuthBurstRule) ID() string {
	return AuthBurstDetectionID
}

func (r *AuthBurstRule) Evaluate(ctx context.Context, event telemetry.Event) (*Finding, error) {
	if !isSuccessfulLogin(event) {
		return nil, nil
	}

	identityID, sourceIP, ok := correlationKeys(event)
	if !ok {
		return nil, nil
	}

	failures, err := r.repository.RecentAuthenticationFailures(
		ctx,
		identityID,
		sourceIP,
		event.Timestamp,
		authWindow,
		authFailureThreshold,
	)
	if err != nil {
		return nil, fmt.Errorf("query authentication failures: %w", err)
	}
	if len(failures) < authFailureThreshold {
		return nil, nil
	}

	sort.Slice(failures, func(i, j int) bool {
		return failures[i].Timestamp.Before(failures[j].Timestamp)
	})

	eventIDs := make([]string, 0, len(failures)+1)
	for _, failure := range failures {
		eventIDs = append(eventIDs, failure.EventID)
	}
	eventIDs = append(eventIDs, event.EventID)

	return &Finding{
		ID:               findingID(AuthBurstDetectionID, identityID, sourceIP, event.EventID),
		DetectionID:      AuthBurstDetectionID,
		DetectionVersion: AuthBurstDetectionVersion,
		DedupKey:         dedupKey(AuthBurstDetectionID, identityID, sourceIP, event.EventID),
		Title:            "Repeated Authentication Failures Followed by Success",
		Severity:         "high",
		Confidence:       85,
		Status:           "new",
		FirstObservedAt:  failures[0].Timestamp.UTC(),
		LastObservedAt:   event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:       eventIDs,
			FailureCount:   len(failures),
			IdentityID:     identityID,
			SourceIP:       sourceIP,
			WindowSeconds:  int(authWindow.Seconds()),
			SuccessEventID: event.EventID,
		},
		EventIDs: eventIDs,
	}, nil
}

func isSuccessfulLogin(event telemetry.Event) bool {
	return event.Event.Category == "authentication" &&
		event.Event.Action == "login" &&
		event.Event.Outcome == "success"
}

func correlationKeys(event telemetry.Event) (string, string, bool) {
	if event.Actor == nil || event.Network == nil {
		return "", "", false
	}

	identityID := strings.TrimSpace(event.Actor.ID)
	sourceIP := strings.TrimSpace(event.Network.SourceIP)
	if identityID == "" || sourceIP == "" {
		return "", "", false
	}

	return identityID, sourceIP, true
}
