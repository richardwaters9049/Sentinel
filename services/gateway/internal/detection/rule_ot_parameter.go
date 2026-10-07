package detection

import (
	"context"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type OTParameterChangeRule struct{}

func NewOTParameterChangeRule() *OTParameterChangeRule {
	return &OTParameterChangeRule{}
}

func (r *OTParameterChangeRule) ID() string {
	return OTParameterChangeDetectionID
}

func (r *OTParameterChangeRule) Evaluate(_ context.Context, event telemetry.Event) (*Finding, error) {
	if event.Event.Category != "ot" ||
		event.Event.Action != "parameter_change" ||
		event.Asset == nil {
		return nil, nil
	}

	if strings.ToLower(strings.TrimSpace(event.Asset.Zone)) != "ot" {
		return nil, nil
	}

	deviceType := strings.ToLower(strings.TrimSpace(event.Labels["ot.device_type"]))
	if deviceType != "plc" && deviceType != "rtu" && deviceType != "pac" {
		return nil, nil
	}

	operation := strings.TrimSpace(event.Labels["ot.operation"])
	protocol := strings.TrimSpace(event.Labels["ot.protocol"])
	safetyImpact := strings.TrimSpace(event.Labels["ot.safety_impact"])

	return &Finding{
		ID:               findingID(OTParameterChangeDetectionID, event.Asset.ID, event.EventID),
		DetectionID:      OTParameterChangeDetectionID,
		DetectionVersion: OTParameterChangeVersion,
		DedupKey:         dedupKey(OTParameterChangeDetectionID, event.Asset.ID, event.EventID),
		Title:            "PLC or Controller Parameter Change",
		Severity:         "high",
		Confidence:       92,
		Status:           "new",
		FirstObservedAt:  event.Timestamp.UTC(),
		LastObservedAt:   event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:        []string{event.EventID},
			SourceZone:      "ot",
			TerminalEventID: event.EventID,
			OTProtocol:      protocol,
			OTDeviceType:    deviceType,
			OTOperation:     operation,
			SafetyImpact:    safetyImpact,
		},
		EventIDs: []string{event.EventID},
	}, nil
}
