package behaviour

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
)

type Profile struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Features           []string `json:"features"`
	RequiredTelemetry  []string `json:"required_telemetry"`
	ContextWindow      string   `json:"context_window"`
	MinimumPriorEvents int      `json:"minimum_prior_events"`
	Limitations        []string `json:"limitations"`
	AnalystActions     []string `json:"analyst_actions"`
}

type ProfileValidation struct {
	ProfileID                  string   `json:"profile_id"`
	ReferenceCount             int      `json:"reference_count"`
	ChangedCount               int      `json:"changed_count"`
	ReferenceFlagged           int      `json:"reference_flagged"`
	ChangedFlagged             int      `json:"changed_flagged"`
	ReferenceFlagRate          float64  `json:"reference_flag_rate"`
	ChangedFlagRate            float64  `json:"changed_flag_rate"`
	ReferenceMeanScore         float64  `json:"reference_mean_score"`
	ChangedMeanScore           float64  `json:"changed_mean_score"`
	ChangedExplanationFeatures []string `json:"changed_explanation_features"`
	Interpretation             string   `json:"interpretation"`
}

type CatalogueValidation struct {
	CatalogueVersion string              `json:"catalogue_version"`
	ModelVersion     string              `json:"model_version"`
	DatasetName      string              `json:"dataset_name"`
	Threshold        int                 `json:"threshold"`
	Profiles         []ProfileValidation `json:"profiles"`
}

type Catalogue struct {
	CatalogueVersion string              `json:"catalogue_version"`
	ModelVersion     string              `json:"model_version"`
	DatasetName      string              `json:"dataset_name"`
	Profiles         []Profile           `json:"profiles"`
	Validation       CatalogueValidation `json:"validation"`
}

func (c Catalogue) Validate() error {
	bounded := func(value string, max int) bool { return len(value) > 0 && len(value) <= max }
	if !bounded(c.CatalogueVersion, 128) || !bounded(c.ModelVersion, 128) || !bounded(c.DatasetName, 128) || len(c.Profiles) == 0 || len(c.Profiles) > 32 {
		return fmt.Errorf("invalid behavioural catalogue metadata")
	}
	seen := make(map[string]bool, len(c.Profiles))
	featureName := regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	for _, p := range c.Profiles {
		if !bounded(p.ID, 64) || seen[p.ID] || !bounded(p.Title, 128) || !bounded(p.Description, 1024) || !bounded(p.ContextWindow, 128) || p.MinimumPriorEvents < 0 || p.MinimumPriorEvents > 100000 {
			return fmt.Errorf("invalid behavioural profile metadata")
		}
		seen[p.ID] = true
		for _, values := range [][]string{p.Features, p.RequiredTelemetry, p.Limitations, p.AnalystActions} {
			if len(values) == 0 || len(values) > 16 {
				return fmt.Errorf("invalid behavioural profile fields")
			}
			for _, value := range values {
				if !bounded(value, 1024) {
					return fmt.Errorf("invalid behavioural profile field length")
				}
			}
		}
		for _, feature := range p.Features {
			if !featureName.MatchString(feature) {
				return fmt.Errorf("invalid behavioural feature name")
			}
		}
	}

	v := c.Validation
	if v.ModelVersion != c.ModelVersion || v.CatalogueVersion != c.CatalogueVersion || v.DatasetName != c.DatasetName || v.Threshold < 1 || v.Threshold > 99 || len(v.Profiles) != len(c.Profiles) {
		return fmt.Errorf("invalid catalogue validation provenance")
	}
	validated := make(map[string]bool, len(v.Profiles))
	for _, p := range v.Profiles {
		if !seen[p.ProfileID] || validated[p.ProfileID] || p.ReferenceCount < 1 || p.ReferenceCount > 10000 || p.ChangedCount < 1 || p.ChangedCount > 10000 || p.ReferenceFlagged < 0 || p.ReferenceFlagged > p.ReferenceCount || p.ChangedFlagged < 0 || p.ChangedFlagged > p.ChangedCount {
			return fmt.Errorf("invalid profile validation counts")
		}
		validated[p.ProfileID] = true
		for _, value := range []float64{p.ReferenceFlagRate, p.ChangedFlagRate, p.ReferenceMeanScore, p.ChangedMeanScore} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("non-finite catalogue validation")
			}
		}
		if math.Abs(p.ReferenceFlagRate-float64(p.ReferenceFlagged)/float64(p.ReferenceCount)) > 1e-6 || math.Abs(p.ChangedFlagRate-float64(p.ChangedFlagged)/float64(p.ChangedCount)) > 1e-6 || p.ReferenceMeanScore < 0 || p.ReferenceMeanScore > 100 || p.ChangedMeanScore < 0 || p.ChangedMeanScore > 100 || !bounded(p.Interpretation, 1024) {
			return fmt.Errorf("invalid profile validation metrics")
		}
		if len(p.ChangedExplanationFeatures) == 0 || len(p.ChangedExplanationFeatures) > 16 {
			return fmt.Errorf("invalid validation explanation features")
		}
		for _, feature := range p.ChangedExplanationFeatures {
			if !featureName.MatchString(feature) {
				return fmt.Errorf("invalid validation feature name")
			}
		}
	}
	return nil
}

func (c *Client) Catalogue(ctx context.Context) (Catalogue, error) {
	if c == nil || c.client == nil || c.baseURL == "" {
		return Catalogue{}, fmt.Errorf("behaviour client is not initialised")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/catalogue", nil)
	if err != nil {
		return Catalogue{}, fmt.Errorf("create catalogue request: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return Catalogue{}, fmt.Errorf("query behaviour catalogue: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Catalogue{}, fmt.Errorf("catalogue returned HTTP %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return Catalogue{}, fmt.Errorf("read behaviour catalogue: %w", err)
	}
	if int64(len(payload)) > maxResponseBytes {
		return Catalogue{}, fmt.Errorf("behaviour catalogue exceeded size limit")
	}
	var result Catalogue
	if err := json.Unmarshal(payload, &result); err != nil {
		return Catalogue{}, fmt.Errorf("decode behaviour catalogue: %w", err)
	}
	if err := result.Validate(); err != nil {
		return Catalogue{}, err
	}
	return result, nil
}
