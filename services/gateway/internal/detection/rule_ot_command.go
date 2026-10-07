package detection

import (
	"context"
	"strings"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type OTUnauthorizedCommandRule struct{}

func NewOTUnauthorizedCommandRule() *OTUnauthorizedCommandRule {
	return &OTUnauthorizedCommandRule{}
}

func (r *OTUnauthorizedCommandRule) ID() string {
	return OTUnauthorizedCommandID
}

func (r *OTUnauthorizedCommandRule) Evaluate(_ context.Context, event telemetry.Event) (*Finding, error) {
	if event.Event.Category != "ot" ||
		event.Event.Action != "command_message" ||
		event.Asset == nil {
		return nil, nil
	}

	if strings.ToLower(strings.TrimSpace(event.Asset.Zone)) != "ot" {
		return nil, nil
	}

	authorization := strings.ToLower(strings.TrimSpace(event.Labels["ot.authorized"]))
	if authorization != "false" {
		return nil, nil
	}

	deviceType := strings.ToLower(strings.TrimSpace(event.Labels["ot.device_type"]))
	protocol := strings.TrimSpace(event.Labels["ot.protocol"])
	operation := strings.TrimSpace(event.Labels["ot.operation"])
	safetyImpact := strings.TrimSpace(event.Labels["ot.safety_impact"])

	destinationIP := ""
	destinationZone := "ot"
	sourceIP := ""
	if event.Network != nil {
		destinationIP = strings.TrimSpace(event.Network.DestinationIP)
		sourceIP = strings.TrimSpace(event.Network.SourceIP)
		if value := strings.ToLower(strings.TrimSpace(event.Network.DestinationZone)); value != "" {
			destinationZone = value
		}
	}

	return &Finding{
		ID:               findingID(OTUnauthorizedCommandID, event.Asset.ID, event.EventID),
		DetectionID:      OTUnauthorizedCommandID,
		DetectionVersion: OTUnauthorizedCommandVersion,
		DedupKey:         dedupKey(OTUnauthorizedCommandID, event.Asset.ID, event.EventID),
		Title:            "Unauthorized OT Command Message",
		Severity:         "critical",
		Confidence:       96,
		Status:           "new",
		FirstObservedAt:  event.Timestamp.UTC(),
		LastObservedAt:   event.Timestamp.UTC(),
		Evidence: Evidence{
			EventIDs:        []string{event.EventID},
			SourceIP:        sourceIP,
			SourceZone:      "ot",
			DestinationIP:   destinationIP,
			DestinationZone: destinationZone,
			TerminalEventID: event.EventID,
			OTProtocol:      protocol,
			OTDeviceType:    deviceType,
			OTOperation:     operation,
			OTAuthorization: authorization,
			SafetyImpact:    safetyImpact,
		},
		EventIDs: []string{event.EventID},
	}, nil
}
