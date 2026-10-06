package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type acceptedEvent struct {
	Accepted bool   `json:"accepted"`
	EventID  string `json:"event_id"`
}

func main() {
	var (
		baseURL  = flag.String("url", "http://127.0.0.1:8080", "Sentinel gateway base URL")
		scenario = flag.String("scenario", "auth-burst", "synthetic scenario to emit")
		delay    = flag.Duration("delay", 250*time.Millisecond, "delay between events")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := scenarioEvents(*scenario, time.Now().UTC())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for i, event := range events {
		eventID, err := postEvent(ctx, client, *baseURL, event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "event %d failed: %v\n", i+1, err)
			os.Exit(1)
		}

		fmt.Printf("accepted %d/%d: %s\n", i+1, len(events), eventID)

		if i < len(events)-1 {
			select {
			case <-ctx.Done():
				fmt.Fprintln(os.Stderr, "simulation timed out")
				os.Exit(1)
			case <-time.After(*delay):
			}
		}
	}
}

func scenarioEvents(name string, start time.Time) ([]map[string]interface{}, error) {
	switch name {
	case "auth-burst":
		return authBurst(start), nil
	case "normal-login":
		return []map[string]interface{}{
			authEvent(start, "success", "10.10.10.20"),
		}, nil
	case "service-account-login":
		return []map[string]interface{}{
			serviceAccountLoginEvent(start),
		}, nil
	case "it-to-ot-connection":
		return []map[string]interface{}{
			corporateToOTEvent(start),
		}, nil
	default:
		return nil, fmt.Errorf("unknown scenario %q", name)
	}
}

func authBurst(start time.Time) []map[string]interface{} {
	events := make([]map[string]interface{}, 0, 5)
	for i := 0; i < 4; i++ {
		events = append(events, authEvent(
			start.Add(time.Duration(i)*10*time.Second),
			"failure",
			"10.10.99.44",
		))
	}

	events = append(events, authEvent(
		start.Add(45*time.Second),
		"success",
		"10.10.99.44",
	))

	return events
}

func authEvent(timestamp time.Time, outcome, sourceIP string) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "identity",
			"vendor":    "sentinel-sim",
			"collector": "simulator-identity-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-engineering-ws-01",
			"hostname": "engineering-workstation-01",
			"zone":     "ot",
		},
		"actor": map[string]interface{}{
			"id":   "user-demo-operator",
			"type": "human",
			"name": "Demo Operator",
		},
		"event": map[string]interface{}{
			"category": "authentication",
			"action":   "login",
			"outcome":  outcome,
		},
		"network": map[string]interface{}{
			"source_ip":        sourceIP,
			"destination_ip":   "10.20.0.15",
			"destination_port": 443,
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "auth-burst",
		},
	}
}

func serviceAccountLoginEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "identity",
			"vendor":    "sentinel-sim",
			"collector": "simulator-identity-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-app-server-01",
			"hostname": "application-server-01",
			"zone":     "corporate",
		},
		"actor": map[string]interface{}{
			"id":   "svc-backup",
			"type": "service_account",
			"name": "Backup Service",
		},
		"event": map[string]interface{}{
			"category": "authentication",
			"action":   "login",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.20.15",
			"destination_ip":   "10.10.20.30",
			"destination_port": 22,
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "service-account-login",
		},
	}
}

func corporateToOTEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "network",
			"vendor":    "sentinel-sim",
			"collector": "simulator-network-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-employee-ws-01",
			"hostname": "employee-workstation-01",
			"zone":     "corporate",
		},
		"event": map[string]interface{}{
			"category": "network",
			"action":   "connection",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.10.25",
			"destination_ip":   "10.30.0.10",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "it-to-ot-connection",
		},
	}
}

func postEvent(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	event map[string]interface{},
) (string, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode event: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/api/v1/telemetry",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send event: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if response.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf(
			"gateway returned %s: %s",
			response.Status,
			string(body),
		)
	}

	var accepted acceptedEvent
	if err := json.Unmarshal(body, &accepted); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if !accepted.Accepted || accepted.EventID == "" {
		return "", fmt.Errorf("gateway did not return an accepted event id")
	}

	return accepted.EventID, nil
}
