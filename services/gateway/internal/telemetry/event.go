package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const SchemaVersion = "1.0.0"

var ErrInvalidEvent = errors.New("invalid telemetry event")

type Source struct {
	Type      string `json:"type"`
	Vendor    string `json:"vendor,omitempty"`
	Collector string `json:"collector"`
}

type Asset struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Zone     string `json:"zone"`
}

type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type EventDetails struct {
	Category string `json:"category"`
	Action   string `json:"action"`
	Outcome  string `json:"outcome,omitempty"`
}

type Network struct {
	SourceIP        string `json:"source_ip,omitempty"`
	DestinationIP   string `json:"destination_ip,omitempty"`
	DestinationPort int    `json:"destination_port,omitempty"`
	DestinationZone string `json:"destination_zone,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
}

type IngestRequest struct {
	EventID   string            `json:"event_id,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Source    Source            `json:"source"`
	Asset     *Asset            `json:"asset,omitempty"`
	Actor     *Actor            `json:"actor,omitempty"`
	Event     EventDetails      `json:"event"`
	Network   *Network          `json:"network,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type Event struct {
	EventID       string            `json:"event_id"`
	SchemaVersion string            `json:"schema_version"`
	Timestamp     time.Time         `json:"timestamp"`
	ReceivedAt    time.Time         `json:"received_at"`
	Source        Source            `json:"source"`
	Asset         *Asset            `json:"asset,omitempty"`
	Actor         *Actor            `json:"actor,omitempty"`
	Event         EventDetails      `json:"event"`
	Network       *Network          `json:"network,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

func Normalize(req IngestRequest, now time.Time) (Event, error) {
	if err := validate(req, now); err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}

	eventID := strings.TrimSpace(req.EventID)
	if eventID == "" {
		generated, err := newEventID()
		if err != nil {
			return Event{}, fmt.Errorf("generate event id: %w", err)
		}
		eventID = generated
	}

	labels := make(map[string]string, len(req.Labels))
	for key, value := range req.Labels {
		labels[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	event := Event{
		EventID:       eventID,
		SchemaVersion: SchemaVersion,
		Timestamp:     req.Timestamp.UTC(),
		ReceivedAt:    now.UTC(),
		Source: Source{
			Type:      strings.ToLower(strings.TrimSpace(req.Source.Type)),
			Vendor:    strings.TrimSpace(req.Source.Vendor),
			Collector: strings.TrimSpace(req.Source.Collector),
		},
		Event: EventDetails{
			Category: strings.ToLower(strings.TrimSpace(req.Event.Category)),
			Action:   strings.ToLower(strings.TrimSpace(req.Event.Action)),
			Outcome:  strings.ToLower(strings.TrimSpace(req.Event.Outcome)),
		},
		Labels: labels,
	}

	if req.Asset != nil {
		event.Asset = &Asset{
			ID:       strings.TrimSpace(req.Asset.ID),
			Hostname: strings.TrimSpace(req.Asset.Hostname),
			Zone:     strings.ToLower(strings.TrimSpace(req.Asset.Zone)),
		}
	}

	if req.Actor != nil {
		event.Actor = &Actor{
			ID:   strings.TrimSpace(req.Actor.ID),
			Type: strings.ToLower(strings.TrimSpace(req.Actor.Type)),
			Name: strings.TrimSpace(req.Actor.Name),
		}
	}

	if req.Network != nil {
		event.Network = &Network{
			SourceIP:        strings.TrimSpace(req.Network.SourceIP),
			DestinationIP:   strings.TrimSpace(req.Network.DestinationIP),
			DestinationPort: req.Network.DestinationPort,
			DestinationZone: strings.ToLower(strings.TrimSpace(req.Network.DestinationZone)),
			Protocol:        strings.ToLower(strings.TrimSpace(req.Network.Protocol)),
		}
	}

	return event, nil
}

func validate(req IngestRequest, now time.Time) error {
	var errs []error

	if req.Timestamp.IsZero() {
		errs = append(errs, errors.New("timestamp is required"))
	} else if req.Timestamp.After(now.Add(5 * time.Minute)) {
		errs = append(errs, errors.New("timestamp cannot be more than 5 minutes in the future"))
	}

	if value := strings.TrimSpace(req.EventID); value != "" && !validIdentifier(value) {
		errs = append(errs, errors.New("event_id contains unsupported characters or is too long"))
	}

	if value := strings.TrimSpace(req.Source.Type); value == "" {
		errs = append(errs, errors.New("source.type is required"))
	} else if len(value) > 64 {
		errs = append(errs, errors.New("source.type must be 64 characters or fewer"))
	}

	if value := strings.TrimSpace(req.Source.Collector); value == "" {
		errs = append(errs, errors.New("source.collector is required"))
	} else if len(value) > 128 {
		errs = append(errs, errors.New("source.collector must be 128 characters or fewer"))
	}

	if len(strings.TrimSpace(req.Source.Vendor)) > 128 {
		errs = append(errs, errors.New("source.vendor must be 128 characters or fewer"))
	}

	if value := strings.TrimSpace(req.Event.Category); value == "" {
		errs = append(errs, errors.New("event.category is required"))
	} else if len(value) > 64 {
		errs = append(errs, errors.New("event.category must be 64 characters or fewer"))
	}

	if value := strings.TrimSpace(req.Event.Action); value == "" {
		errs = append(errs, errors.New("event.action is required"))
	} else if len(value) > 128 {
		errs = append(errs, errors.New("event.action must be 128 characters or fewer"))
	}

	if len(strings.TrimSpace(req.Event.Outcome)) > 64 {
		errs = append(errs, errors.New("event.outcome must be 64 characters or fewer"))
	}

	if req.Asset != nil {
		if !validIdentifier(strings.TrimSpace(req.Asset.ID)) {
			errs = append(errs, errors.New("asset.id is required and must use supported characters"))
		}
		if strings.TrimSpace(req.Asset.Hostname) == "" {
			errs = append(errs, errors.New("asset.hostname is required"))
		}
		if strings.TrimSpace(req.Asset.Zone) == "" {
			errs = append(errs, errors.New("asset.zone is required"))
		}
	}

	if req.Actor != nil {
		if !validIdentifier(strings.TrimSpace(req.Actor.ID)) {
			errs = append(errs, errors.New("actor.id is required and must use supported characters"))
		}
		if strings.TrimSpace(req.Actor.Type) == "" {
			errs = append(errs, errors.New("actor.type is required"))
		}
		if strings.TrimSpace(req.Actor.Name) == "" {
			errs = append(errs, errors.New("actor.name is required"))
		}
	}

	if req.Network != nil {
		if ip := strings.TrimSpace(req.Network.SourceIP); ip != "" && net.ParseIP(ip) == nil {
			errs = append(errs, errors.New("network.source_ip must be a valid IP address"))
		}
		if ip := strings.TrimSpace(req.Network.DestinationIP); ip != "" && net.ParseIP(ip) == nil {
			errs = append(errs, errors.New("network.destination_ip must be a valid IP address"))
		}
		if req.Network.DestinationPort < 0 || req.Network.DestinationPort > 65535 {
			errs = append(errs, errors.New("network.destination_port must be between 0 and 65535"))
		}
		if len(strings.TrimSpace(req.Network.DestinationZone)) > 64 {
			errs = append(errs, errors.New("network.destination_zone must be 64 characters or fewer"))
		}
		if len(strings.TrimSpace(req.Network.Protocol)) > 32 {
			errs = append(errs, errors.New("network.protocol must be 32 characters or fewer"))
		}
	}

	if len(req.Labels) > 50 {
		errs = append(errs, errors.New("labels must contain 50 entries or fewer"))
	}
	for key, value := range req.Labels {
		if strings.TrimSpace(key) == "" || len(key) > 64 {
			errs = append(errs, errors.New("label keys must be between 1 and 64 characters"))
			break
		}
		if len(value) > 256 {
			errs = append(errs, errors.New("label values must be 256 characters or fewer"))
			break
		}
	}

	return errors.Join(errs...)
}

func validIdentifier(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}

	for i, r := range value {
		isLetter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit || (i > 0 && strings.ContainsRune("._:-", r)) {
			continue
		}
		return false
	}

	return true
}

func newEventID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}

	return "evt_" + hex.EncodeToString(random[:]), nil
}
