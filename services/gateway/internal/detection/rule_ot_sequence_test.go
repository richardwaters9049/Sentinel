package detection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type fakeOTRepository struct {
	recent   []OTEvent
	queryErr error
	assetID  string
	before   time.Time
	window   time.Duration
	actions  []string
	limit    int
}

func (f *fakeOTRepository) IsDetectionEnabled(context.Context, string) (bool, error) {
	return true, nil
}

func (f *fakeOTRepository) RecentAuthenticationFailures(
	context.Context,
	string,
	string,
	time.Time,
	time.Duration,
	int,
) ([]AuthFailure, error) {
	return nil, nil
}

func (f *fakeOTRepository) RecentOTActions(
	_ context.Context,
	assetID string,
	before time.Time,
	window time.Duration,
	actions []string,
	limit int,
) ([]OTEvent, error) {
	f.assetID = assetID
	f.before = before
	f.window = window
	f.actions = append([]string(nil), actions...)
	f.limit = limit
	return f.recent, f.queryErr
}

func (f *fakeOTRepository) CreateFinding(context.Context, Finding) (bool, error) {
	return true, nil
}

func TestOTChangeSequenceRuleCorrelatesModeAndParameterChange(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 7, 18, 0, 0, 0, time.UTC)
	repo := &fakeOTRepository{
		recent: []OTEvent{
			{
				EventID:   "evt_mode_001",
				Action:    "controller_mode_change",
				Timestamp: now.Add(-90 * time.Second),
			},
		},
	}

	event := telemetry.Event{
		EventID:   "evt_parameter_002",
		Timestamp: now,
		Asset: &telemetry.Asset{
			ID:       "asset-plc-sim-02",
			Hostname: "plc-sim-02",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "parameter_change",
			Outcome:  "success",
		},
		Labels: map[string]string{
			"ot.device_type":   "plc",
			"ot.protocol":      "modbus-tcp",
			"ot.operation":     "setpoint_change",
			"ot.safety_impact": "potential_process_impact",
		},
	}

	finding, err := NewOTChangeSequenceRule(repo).Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate correlation: %v", err)
	}
	if finding == nil {
		t.Fatal("expected correlated finding")
	}
	if finding.DetectionID != OTChangeSequenceDetectionID {
		t.Fatalf("expected %q, got %q", OTChangeSequenceDetectionID, finding.DetectionID)
	}
	if finding.Severity != "critical" || finding.Confidence != 94 {
		t.Fatalf("unexpected severity/confidence: %s/%d", finding.Severity, finding.Confidence)
	}
	if len(finding.EventIDs) != 2 ||
		finding.EventIDs[0] != "evt_mode_001" ||
		finding.EventIDs[1] != "evt_parameter_002" {
		t.Fatalf("unexpected evidence order: %#v", finding.EventIDs)
	}
	if repo.assetID != "asset-plc-sim-02" {
		t.Fatalf("unexpected correlation asset %q", repo.assetID)
	}
	if repo.window != otChangeSequenceWindow || repo.limit != 8 {
		t.Fatalf("unexpected query bounds: %s limit=%d", repo.window, repo.limit)
	}
}

func TestOTChangeSequenceRuleIgnoresParameterChangeWithoutPriorModeChange(t *testing.T) {
	t.Parallel()

	repo := &fakeOTRepository{}
	event := telemetry.Event{
		EventID:   "evt_parameter_only",
		Timestamp: time.Now().UTC(),
		Asset: &telemetry.Asset{
			ID:       "asset-plc-sim-02",
			Hostname: "plc-sim-02",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "parameter_change",
		},
	}

	finding, err := NewOTChangeSequenceRule(repo).Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("evaluate correlation: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding, got %#v", finding)
	}
}

func TestOTChangeSequenceRulePropagatesRepositoryFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeOTRepository{queryErr: errors.New("database unavailable")}
	event := telemetry.Event{
		EventID:   "evt_parameter_error",
		Timestamp: time.Now().UTC(),
		Asset: &telemetry.Asset{
			ID:       "asset-plc-sim-02",
			Hostname: "plc-sim-02",
			Zone:     "ot",
		},
		Event: telemetry.EventDetails{
			Category: "ot",
			Action:   "parameter_change",
		},
	}

	if _, err := NewOTChangeSequenceRule(repo).Evaluate(context.Background(), event); err == nil {
		t.Fatal("expected repository error")
	}
}
