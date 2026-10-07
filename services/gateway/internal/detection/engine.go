package detection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type Repository interface {
	IsDetectionEnabled(context.Context, string) (bool, error)
	RecentAuthenticationFailures(
		context.Context,
		string,
		string,
		time.Time,
		time.Duration,
		int,
	) ([]AuthFailure, error)
	RecentOTActions(
		context.Context,
		string,
		time.Time,
		time.Duration,
		[]string,
		int,
	) ([]OTEvent, error)
	CreateFinding(context.Context, Finding) (bool, error)
}

type Rule interface {
	ID() string
	Evaluate(context.Context, telemetry.Event) (*Finding, error)
}

type Engine struct {
	repository Repository
	rules      []Rule
}

func New(repository Repository) *Engine {
	return &Engine{
		repository: repository,
		rules: []Rule{
			NewAuthBurstRule(repository),
			NewServiceAccountLoginRule(),
			NewCorporateToOTRule(),
			NewOTParameterChangeRule(),
			NewOTUnauthorizedCommandRule(),
			NewOTChangeSequenceRule(repository),
		},
	}
}

func NewWithRules(repository Repository, rules ...Rule) *Engine {
	return &Engine{
		repository: repository,
		rules:      append([]Rule(nil), rules...),
	}
}

func (e *Engine) Process(ctx context.Context, event telemetry.Event) error {
	if e == nil || e.repository == nil {
		return fmt.Errorf("detection engine is not initialised")
	}

	for _, rule := range e.rules {
		if rule == nil {
			continue
		}

		enabled, err := e.repository.IsDetectionEnabled(ctx, rule.ID())
		if err != nil {
			return fmt.Errorf("check detection state for %s: %w", rule.ID(), err)
		}
		if !enabled {
			continue
		}

		finding, err := rule.Evaluate(ctx, event)
		if err != nil {
			return fmt.Errorf("evaluate detection %s: %w", rule.ID(), err)
		}
		if finding == nil {
			continue
		}

		if _, err := e.repository.CreateFinding(ctx, *finding); err != nil {
			return fmt.Errorf("create finding for %s: %w", rule.ID(), err)
		}
	}

	return nil
}

func findingID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "fnd_" + hex.EncodeToString(sum[:16])
}

func dedupKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
