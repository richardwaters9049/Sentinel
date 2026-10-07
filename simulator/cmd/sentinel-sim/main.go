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
	case "ot-hmi-read-baseline":
		return []map[string]interface{}{
			otHMIReadEvent(start),
		}, nil
	case "ot-plc-parameter-change":
		return []map[string]interface{}{
			otPLCParameterChangeEvent(start),
		}, nil
	case "ot-unauthorized-command":
		return []map[string]interface{}{
			otUnauthorizedCommandEvent(start),
		}, nil
	case "ot-controller-mode-change":
		return []map[string]interface{}{
			otControllerModeChangeEvent(start),
		}, nil
	case "ot-change-sequence":
		return []map[string]interface{}{
			otControllerModeChangeEvent(start),
			otPLC02ParameterChangeEvent(start.Add(90 * time.Second)),
		}, nil
	case "ot-historian-read-baseline":
		return []map[string]interface{}{
			otHistorianReadEvent(start),
		}, nil
	case "ot-sensor-telemetry-baseline":
		return []map[string]interface{}{
			otSensorTelemetryEvent(start),
		}, nil
	case "intel-ioc-match":
		return []map[string]interface{}{
			intelligenceIOCMatchEvent(start),
		}, nil
	case "intel-domain-match":
		return []map[string]interface{}{
			intelligenceDomainMatchEvent(start),
		}, nil
	case "intel-sha256-match":
		return []map[string]interface{}{
			intelligenceSHA256MatchEvent(start),
		}, nil
	case "intel-finding-match":
		return []map[string]interface{}{
			intelligenceFindingMatchEvent(start),
		}, nil
	case "behaviour-normal":
		return []map[string]interface{}{
			behaviourNormalEvent(start),
		}, nil
	case "behaviour-anomaly":
		return []map[string]interface{}{
			behaviourAnomalyEvent(start),
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

func otHMIReadEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-hmi-01",
			"hostname": "hmi-01",
			"zone":     "ot",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "telemetry_read",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.30.0.10",
			"destination_ip":   "10.30.0.40",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-hmi-read-baseline",
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "read_register",
			"ot.authorized":    "true",
			"ot.safety_impact": "none",
			"ot.simulated":     "true",
		},
	}
}

func otPLCParameterChangeEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-plc-sim-01",
			"hostname": "plc-sim-01",
			"zone":     "ot",
		},
		"actor": map[string]interface{}{
			"id":   "user-demo-engineer",
			"type": "human",
			"name": "Demo Engineer",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "parameter_change",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.30.0.20",
			"destination_ip":   "10.30.0.40",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-plc-parameter-change",
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "setpoint_change",
			"ot.authorized":    "true",
			"ot.safety_impact": "potential_process_impact",
			"ot.simulated":     "true",
		},
	}
}

func otUnauthorizedCommandEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-engineering-ws-01",
			"hostname": "engineering-workstation-01",
			"zone":     "ot",
		},
		"actor": map[string]interface{}{
			"id":   "user-demo-engineer",
			"type": "human",
			"name": "Demo Engineer",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "command_message",
			"outcome":  "observed",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.30.0.20",
			"destination_ip":   "10.30.0.40",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-unauthorized-command",
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "write_request",
			"ot.authorized":    "false",
			"ot.safety_impact": "potential_process_impact",
			"ot.simulated":     "true",
		},
	}
}

func otControllerModeChangeEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-plc-sim-02",
			"hostname": "plc-sim-02",
			"zone":     "ot",
		},
		"actor": map[string]interface{}{
			"id":   "user-demo-engineer",
			"type": "human",
			"name": "Demo Engineer",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "controller_mode_change",
			"outcome":  "success",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-controller-mode-change",
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "mode_change",
			"ot.authorized":    "true",
			"ot.safety_impact": "potential_process_impact",
			"ot.simulated":     "true",
		},
	}
}

func otPLC02ParameterChangeEvent(timestamp time.Time) map[string]interface{} {
	event := otPLCParameterChangeEvent(timestamp)
	event["asset"] = map[string]interface{}{
		"id":       "asset-plc-sim-02",
		"hostname": "plc-sim-02",
		"zone":     "ot",
	}
	labels := event["labels"].(map[string]string)
	labels["scenario"] = "ot-change-sequence"
	return event
}

func otHistorianReadEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-historian-01",
			"hostname": "historian-01",
			"zone":     "dmz",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "historian_read",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.20.0.40",
			"destination_ip":   "10.30.0.60",
			"destination_port": 443,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-historian-read-baseline",
			"ot.device_type":   "historian",
			"ot.protocol":      "https",
			"ot.operation":     "timeseries_read",
			"ot.authorized":    "true",
			"ot.safety_impact": "none",
			"ot.simulated":     "true",
		},
	}
}

func otSensorTelemetryEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "ot",
			"vendor":    "sentinel-sim",
			"collector": "simulator-ot-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-sensor-sim-01",
			"hostname": "sensor-sim-01",
			"zone":     "ot",
		},
		"event": map[string]interface{}{
			"category": "ot",
			"action":   "sensor_telemetry",
			"outcome":  "success",
		},
		"labels": map[string]string{
			"environment":      "lab",
			"scenario":         "ot-sensor-telemetry-baseline",
			"ot.device_type":   "sensor",
			"ot.protocol":      "telemetry",
			"ot.operation":     "sample",
			"ot.authorized":    "true",
			"ot.safety_impact": "none",
			"ot.simulated":     "true",
		},
	}
}

func intelligenceIOCMatchEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "network",
			"vendor":    "sentinel-sim",
			"collector": "simulator-network-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-intel-01",
			"hostname": "workstation-intel-01",
			"zone":     "corporate",
		},
		"event": map[string]interface{}{
			"category": "network",
			"action":   "connection",
			"outcome":  "observed",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.0.25",
			"destination_ip":   "198.51.100.66",
			"destination_port": 443,
			"destination_zone": "external",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "intel-ioc-match",
			"simulated":   "true",
		},
	}
}

func intelligenceDomainMatchEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "dns",
			"vendor":    "sentinel-sim",
			"collector": "simulator-dns-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-intel-02",
			"hostname": "workstation-intel-02",
			"zone":     "corporate",
		},
		"event": map[string]interface{}{
			"category": "dns",
			"action":   "query",
			"outcome":  "observed",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "intel-domain-match",
			"dns.query":   "telemetry-sync.example",
			"simulated":   "true",
		},
	}
}

func intelligenceSHA256MatchEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "endpoint",
			"vendor":    "sentinel-sim",
			"collector": "simulator-endpoint-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-intel-03",
			"hostname": "workstation-intel-03",
			"zone":     "corporate",
		},
		"event": map[string]interface{}{
			"category": "file",
			"action":   "observed",
			"outcome":  "observed",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "intel-sha256-match",
			"file.sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"simulated":   "true",
		},
	}
}

func intelligenceFindingMatchEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "network",
			"vendor":    "sentinel-sim",
			"collector": "simulator-network-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-intel-04",
			"hostname": "workstation-intel-04",
			"zone":     "corporate",
		},
		"event": map[string]interface{}{
			"category": "network",
			"action":   "connection",
			"outcome":  "observed",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.0.25",
			"destination_ip":   "10.30.0.40",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "intel-finding-match",
			"simulated":   "true",
		},
	}
}

func behaviourNormalEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "endpoint",
			"vendor":    "sentinel-sim",
			"collector": "simulator-behaviour-01",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-behaviour-01",
			"hostname": "workstation-behaviour-01",
			"zone":     "corporate",
		},
		"actor": map[string]interface{}{
			"id":   "user-behaviour-normal",
			"type": "user",
			"name": "Behaviour Baseline User",
		},
		"event": map[string]interface{}{
			"category": "network",
			"action":   "connection",
			"outcome":  "success",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.5.20",
			"destination_ip":   "10.10.5.30",
			"destination_port": 443,
			"destination_zone": "corporate",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "behaviour-normal",
			"simulated":   "true",
		},
	}
}

func behaviourAnomalyEvent(timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": timestamp.Format(time.RFC3339Nano),
		"source": map[string]interface{}{
			"type":      "identity",
			"vendor":    "sentinel-sim",
			"collector": "simulator-behaviour-02",
		},
		"asset": map[string]interface{}{
			"id":       "asset-corporate-behaviour-02",
			"hostname": "jump-behaviour-02",
			"zone":     "corporate",
		},
		"actor": map[string]interface{}{
			"id":   "svc-behaviour-anomaly",
			"type": "service_account",
			"name": "Behaviour Anomaly Service",
		},
		"event": map[string]interface{}{
			"category": "authentication",
			"action":   "login",
			"outcome":  "failure",
		},
		"network": map[string]interface{}{
			"source_ip":        "10.10.8.40",
			"destination_ip":   "10.30.0.40",
			"destination_port": 502,
			"destination_zone": "ot",
			"protocol":         "tcp",
		},
		"labels": map[string]string{
			"environment": "lab",
			"scenario":    "behaviour-anomaly",
			"simulated":   "true",
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
