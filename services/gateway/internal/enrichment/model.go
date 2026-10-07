package enrichment

import "time"

type Indicator struct {
	ID               string                 `json:"id"`
	SourceID         string                 `json:"source_id"`
	SourceName       string                 `json:"source_name"`
	SourceType       string                 `json:"source_type"`
	IndicatorType    string                 `json:"indicator_type"`
	Value            string                 `json:"value"`
	NormalizedValue  string                 `json:"normalized_value"`
	SourceConfidence int                    `json:"source_confidence"`
	Confidence       int                    `json:"confidence"`
	ValidFrom        time.Time              `json:"valid_from"`
	ValidUntil       *time.Time             `json:"valid_until,omitempty"`
	Tags             []string               `json:"tags"`
	Context          map[string]interface{} `json:"context"`
	Provenance       map[string]interface{} `json:"provenance"`
}

type Match struct {
	EventID             string                 `json:"event_id"`
	IndicatorID         string                 `json:"indicator_id"`
	SourceID            string                 `json:"source_id"`
	EventField          string                 `json:"event_field"`
	ObservedValue       string                 `json:"observed_value"`
	SourceConfidence    int                    `json:"source_confidence"`
	IndicatorConfidence int                    `json:"indicator_confidence"`
	EffectiveConfidence int                    `json:"effective_confidence"`
	Provenance          map[string]interface{} `json:"provenance"`
	MatchedAt           time.Time              `json:"matched_at"`
}
