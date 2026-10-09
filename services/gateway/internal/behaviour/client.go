package behaviour

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/observability"
)

const maxResponseBytes int64 = 1 << 20

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil || c.baseURL == "" {
		return fmt.Errorf("behaviour client is not initialised")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create behaviour health request: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("behaviour health check: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("behaviour service health returned HTTP %d", response.StatusCode)
	}
	return nil
}

func (c *Client) Score(ctx context.Context, request ScoreRequest) (Score, error) {
	if c == nil || c.client == nil || c.baseURL == "" {
		return Score{}, fmt.Errorf("behaviour client is not initialised")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return Score{}, fmt.Errorf("marshal behaviour score request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/score",
		bytes.NewReader(body),
	)
	if err != nil {
		return Score{}, fmt.Errorf("create behaviour score request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if parent := observability.Header(ctx); parent != "" {
		httpRequest.Header.Set("traceparent", parent)
	}

	response, err := c.client.Do(httpRequest)
	if err != nil {
		return Score{}, fmt.Errorf("score behaviour: %w", err)
	}
	defer response.Body.Close()

	limited := io.LimitReader(response.Body, maxResponseBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return Score{}, fmt.Errorf("read behaviour score response: %w", err)
	}
	if int64(len(payload)) > maxResponseBytes {
		return Score{}, fmt.Errorf("behaviour score response exceeded size limit")
	}
	if response.StatusCode != http.StatusOK {
		return Score{}, fmt.Errorf("behaviour service returned HTTP %d", response.StatusCode)
	}

	var score Score
	if err := json.Unmarshal(payload, &score); err != nil {
		return Score{}, fmt.Errorf("decode behaviour score response: %w", err)
	}
	if score.EventID != request.EventID || score.EntityID != request.EntityID {
		return Score{}, fmt.Errorf("behaviour score response identity mismatch")
	}
	if score.AnomalyScore < 0 || score.AnomalyScore > 100 {
		return Score{}, fmt.Errorf("behaviour score outside expected range")
	}

	score.ScoredAt = time.Now().UTC()
	return score, nil
}
