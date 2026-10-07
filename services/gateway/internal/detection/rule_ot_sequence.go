package detection

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

const otChangeSequenceWindow = 10 * time.Minute

type OTChangeSequenceRule struct {
	repository Repository
}

func NewOTChangeSequenceRule(repository Repository) *OTChangeSequenceRule {
	return &OTChangeSequenceRule{repository: repository}
}

func (r *OTChangeSequenceRule) ID() string {
	return OTChangeSequenceDetectionID
}

func (r *OTChangeSequenceRule) Evaluate(
	ctx context.Context,
	event telemetry.Event,
) (*Finding, error) {
	if event.Event.Category != "ot" ||
		event.Event.Action != "parameter_change" ||
		event.Asset == nil ||
		r == nil ||
		r.repository == nil {
		return nil, nil
	}

	if strings.ToLower(strings.TrimSpace(event.Asset.Zone)) != "ot" {
		return nil, nil
	}

	recent, err := r.repository.RecentOTActions(
		ctx,
		event.Asset.ID,
		event.Timestamp,
		otChangeSequenceWindow,
		[]string{"controller_mode_change"},
		8,
	)
	if err != nil {
		return nil, err
	}
	if len(recent) == 0 {
		return nil, nil
	}

	events := append([]OTEvent(nil), recent...)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	eventIDs := make([]string, 0, len(events)+1)
	for _, previous := range events {
		eventIDs = append(eventIDs, previous.EventID)
	}
	eventIDs = append(eventIDs, event.EventID)

	firstObserved := events[0].Timestamp.UTC()
	deviceType := strings.ToLower(strings.TrimSpace(event.Labels["ot.device_type"]))
	protocol := strings.TrimSpace(event.Labels["ot.protocol"])
	operation := strings.TrimSpace(event.Labels["ot.operation"])
	safetyImpact := strings.TrimSpace(event.Labels["ot.safety_impact"])

	return &Finding{
		ID: findingID(
			OTChangeSequenceDetectionID,
			event.Asset.ID,
			events[len(events)-1].EventID,
			event.EventID,
		),
		DetectionID:      OTChangeSequenceDetectionID,
		DetectionVersion: OTChangeSequenceVersion,
		DedupKey: dedupKey(
			OTChangeSequenceDetectionID,
			event.Asset.ID,
			events[len(events)-1].EventID,
			event.EventID,
		),
		Title:           "Controller Mode Change Followed by Parameter Change",
		Severity:        "critical",
		Confidence:      94,
		Status:          "new",
		FirstObservedAt: firstObserved,
		LastObservedAt:  event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:        eventIDs,
			SourceZone:      "ot",
			WindowSeconds:   int(otChangeSequenceWindow.Seconds()),
			TerminalEventID: event.EventID,
			OTProtocol:      protocol,
			OTDeviceType:    deviceType,
			OTOperation:     operation,
			SafetyImpact:    safetyImpact,
		},
		EventIDs: eventIDs,
	}, nil
}
