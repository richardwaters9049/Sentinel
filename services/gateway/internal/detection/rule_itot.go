package detection

import (
	"context"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type CorporateToOTRule struct{}

func NewCorporateToOTRule() *CorporateToOTRule {
	return &CorporateToOTRule{}
}

func (r *CorporateToOTRule) ID() string {
	return CorporateToOTDetectionID
}

func (r *CorporateToOTRule) Evaluate(_ context.Context, event telemetry.Event) (*Finding, error) {
	if event.Event.Category != "network" ||
		event.Event.Action != "connection" ||
		event.Asset == nil ||
		event.Network == nil {
		return nil, nil
	}

	sourceZone := strings.ToLower(strings.TrimSpace(event.Asset.Zone))
	destinationZone := strings.ToLower(strings.TrimSpace(event.Network.DestinationZone))
	if sourceZone != "corporate" || destinationZone != "ot" {
		return nil, nil
	}

	return &Finding{
		ID:               findingID(CorporateToOTDetectionID, event.Asset.ID, event.EventID),
		DetectionID:      CorporateToOTDetectionID,
		DetectionVersion: CorporateToOTDetectionVersion,
		DedupKey:         dedupKey(CorporateToOTDetectionID, event.Asset.ID, event.EventID),
		Title:            "Unexpected Corporate-to-OT Network Connection",
		Severity:         "high",
		Confidence:       95,
		Status:           "new",
		FirstObservedAt:  event.Timestamp.UTC(),
		LastObservedAt:   event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:        []string{event.EventID},
			SourceIP:        strings.TrimSpace(event.Network.SourceIP),
			SourceZone:      sourceZone,
			DestinationIP:   strings.TrimSpace(event.Network.DestinationIP),
			DestinationZone: destinationZone,
			TerminalEventID: event.EventID,
		},
		EventIDs: []string{event.EventID},
	}, nil
}
