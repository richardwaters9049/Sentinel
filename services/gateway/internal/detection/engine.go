package detection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type Repository interface {
	RecentAuthenticationFailures(
		context.Context,
		string,
		string,
		time.Time,
		time.Duration,
		int,
	) ([]AuthFailure, error)
	CreateFinding(context.Context, Finding) (bool, error)
}

type Engine struct {
	repository Repository
}

func New(repository Repository) *Engine {
	return &Engine{repository: repository}
}

func (e *Engine) Process(ctx context.Context, event telemetry.Event) error {
	if e == nil || e.repository == nil {
		return fmt.Errorf("detection engine is not initialised")
	}

	if !isSuccessfulLogin(event) {
		return nil
	}

	identityID, sourceIP, ok := correlationKeys(event)
	if !ok {
		return nil
	}

	failures, err := e.repository.RecentAuthenticationFailures(
		ctx,
		identityID,
		sourceIP,
		event.Timestamp,
		authWindow,
		authFailureThreshold,
	)
	if err != nil {
		return fmt.Errorf("query authentication failures: %w", err)
	}

	if len(failures) < authFailureThreshold {
		return nil
	}

	sort.Slice(failures, func(i, j int) bool {
		return failures[i].Timestamp.Before(failures[j].Timestamp)
	})

	eventIDs := make([]string, 0, len(failures)+1)
	for _, failure := range failures {
		eventIDs = append(eventIDs, failure.EventID)
	}
	eventIDs = append(eventIDs, event.EventID)

	firstObserved := failures[0].Timestamp.UTC()
	lastObserved := event.Timestamp.UTC()

	finding := Finding{
		ID:               findingID(AuthBurstDetectionID, identityID, sourceIP, event.EventID),
		DetectionID:      AuthBurstDetectionID,
		DetectionVersion: AuthBurstDetectionVersion,
		DedupKey:         dedupKey(AuthBurstDetectionID, identityID, sourceIP, event.EventID),
		Title:            "Repeated Authentication Failures Followed by Success",
		Severity:         "high",
		Confidence:       85,
		Status:           "new",
		FirstObservedAt:  firstObserved,
		LastObservedAt:   lastObserved,
		Evidence: Evidence{
			EventIDs:       eventIDs,
			FailureCount:   len(failures),
			IdentityID:     identityID,
			SourceIP:       sourceIP,
			WindowSeconds:  int(authWindow.Seconds()),
			SuccessEventID: event.EventID,
		},
		EventIDs: eventIDs,
	}

	if _, err := e.repository.CreateFinding(ctx, finding); err != nil {
		return fmt.Errorf("create finding: %w", err)
	}

	return nil
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

func findingID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "fnd_" + hex.EncodeToString(sum[:16])
}

func dedupKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
