export type InvestigationSummary = {
  id: string;
  title: string;
  description: string;
  status: "open" | "investigating" | "contained" | "closed";
  priority: "low" | "medium" | "high" | "critical";
  owner_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type FindingStatus =
  | "new"
  | "triaged"
  | "investigating"
  | "false_positive"
  | "benign_expected"
  | "duplicate"
  | "confirmed"
  | "contained"
  | "closed";

export type Finding = {
  id: string;
  detection_id: string;
  detection_version: number;
  title: string;
  severity: string;
  confidence: number;
  status: FindingStatus;
  first_observed_at: string;
  last_observed_at: string;
  evidence: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type SentinelEventPayload = {
  actor?: {
    id?: string;
    name?: string;
    type?: string;
  };
  asset?: {
    id?: string;
    hostname?: string;
    zone?: string;
  };
  network?: {
    source_ip?: string;
    destination_ip?: string;
    destination_port?: number;
    protocol?: string;
  };
  labels?: Record<string, string>;
  source?: {
    collector?: string;
    type?: string;
    vendor?: string;
  };
};

export type EvidenceEvent = {
  id: string;
  source_timestamp: string;
  category: string;
  action: string;
  outcome: string;
  asset_id?: string;
  identity_id?: string;
  payload: SentinelEventPayload & Record<string, unknown>;
};

export type InvestigationNote = {
  id: number;
  investigation_id: string;
  actor_id: string;
  body: string;
  created_at: string;
};

export type AuditRecord = {
  id: number;
  occurred_at: string;
  actor_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  request_id?: string;
  details: Record<string, unknown>;
};

export type TimelineEntry = {
  timestamp: string;
  type: "finding" | "event" | "note" | "audit";
  id: string;
  summary: string;
  data?: Record<string, unknown>;
};

export type InvestigationDetail = InvestigationSummary & {
  findings: Finding[];
  events: EvidenceEvent[];
  enrichments: EventEnrichment[];
  behaviour_scores: BehaviourScore[];
  notes: InvestigationNote[];
  audit: AuditRecord[];
  timeline: TimelineEntry[];
};

export type InvestigationListResponse = {
  count: number;
  investigations: InvestigationSummary[];
};

export type HuntQuery = {
  category?: string;
  categories?: string[];
  action?: string;
  actions?: string[];
  outcome?: string;
  outcomes?: string[];
  asset_id?: string;
  identity_id?: string;
  source_ip?: string;
  destination_ip?: string;
  source_zones?: string[];
  destination_zone?: string;
  destination_zones?: string[];
  destination_ports?: number[];
  labels?: Record<string, string>;
  from?: string;
  to?: string;
  last_minutes?: number;
  limit?: number;
};

export type HuntDefinition = {
  id: string;
  version: number;
  name: string;
  description: string;
  hypothesis: string;
  query: HuntQuery;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type HuntListResponse = {
  count: number;
  hunts: HuntDefinition[];
};

export type HuntRunResult = {
  run_id: number;
  hunt_id: string;
  result_count: number;
  events: EvidenceEvent[];
  executed_at: string;
};

export type TelemetryEvent = {
  event_id: string;
  schema_version: string;
  timestamp: string;
  received_at: string;
  source: {
    type: string;
    vendor?: string;
    collector: string;
  };
  asset?: {
    id: string;
    hostname: string;
    zone: string;
  };
  actor?: {
    id: string;
    type: string;
    name: string;
  };
  event: {
    category: string;
    action: string;
    outcome?: string;
  };
  network?: {
    source_ip?: string;
    destination_ip?: string;
    destination_port?: number;
    destination_zone?: string;
    protocol?: string;
  };
  labels?: Record<string, string>;
};

export type EventListResponse = {
  count: number;
  events: TelemetryEvent[];
};

export type FindingListResponse = {
  count: number;
  findings: Finding[];
};

export type FindingDetail = Finding & {
  events: EvidenceEvent[];
  enrichments: EventEnrichment[];
  behaviour_scores: BehaviourScore[];
  audit: AuditRecord[];
};

export type FindingEvidenceContext = {
  finding_id: string;
  linked_events: EvidenceEvent[];
  context_events: EvidenceEvent[];
  enrichments: EventEnrichment[];
  behaviour_scores: BehaviourScore[];
  context_minutes: number;
};

export type AssetPivot = {
  id: string;
  hostname: string;
  zone: string;
  criticality: string;
  first_seen_at?: string;
  last_seen_at?: string;
  recent_events: EvidenceEvent[];
  findings: Finding[];
};

export type DetectionRecord = {
  id: string;
  version: number;
  title: string;
  description: string;
  severity: string;
  enabled: boolean;
  definition: Record<string, unknown>;
  mitre: Array<Record<string, unknown>>;
  created_at: string;
  updated_at: string;
};

export type DetectionListResponse = {
  count: number;
  detections: DetectionRecord[];
};

export type DetectionMetric = {
  detection_id: string;
  hit_count: number;
  open_count: number;
  confirmed_count: number;
  false_positive_count: number;
  closed_count: number;
  false_positive_rate: number;
  last_triggered_at?: string;
};

export type DetectionMetricsResponse = {
  count: number;
  metrics: DetectionMetric[];
};


export type IntelligenceIndicator = {
  id: string;
  source_id: string;
  source_name: string;
  source_type: string;
  indicator_type: "ip" | "domain" | "sha256";
  value: string;
  normalized_value: string;
  source_confidence: number;
  confidence: number;
  valid_from: string;
  valid_until?: string;
  tags: string[];
  context: Record<string, unknown>;
  provenance: Record<string, unknown>;
};

export type IntelligenceIndicatorListResponse = {
  count: number;
  indicators: IntelligenceIndicator[];
};

export type EventEnrichment = {
  id: number;
  event_id: string;
  indicator_id: string;
  source_id: string;
  source_name: string;
  source_type: string;
  indicator_type: string;
  indicator_value: string;
  event_field: string;
  observed_value: string;
  source_confidence: number;
  indicator_confidence: number;
  effective_confidence: number;
  tags: string[];
  context: Record<string, unknown>;
  provenance: Record<string, unknown>;
  matched_at: string;
};

export type IntelligenceMatchListResponse = {
  count: number;
  matches: EventEnrichment[];
};

export type IntelligenceMetrics = {
  active_sources: number;
  active_indicators: number;
  enriched_events: number;
  total_matches: number;
  high_confidence_hits: number;
};


export type IntelligenceSource = {
  id: string;
  name: string;
  source_type: string;
  description: string;
  default_confidence: number;
  provenance: Record<string, unknown>;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type IntelligenceSourceListResponse = {
  count: number;
  sources: IntelligenceSource[];
};


export type BehaviourExplanation = {
  feature: string;
  observed: number;
  baseline: number;
  deviation: number;
  message: string;
};

export type BehaviourBaselineContext = {
  prior_events_60m: number;
  prior_events_24h: number;
  unique_destination_ips_24h: number;
  unique_destination_ports_24h: number;
  auth_failures_60m: number;
  ot_events_24h: number;
  event_rate_60m: number;
  destination_diversity_24h: number;
  auth_failure_rate_60m: number;
  ot_activity_rate_24h: number;
};

export type BehaviourScore = {
  event_id: string;
  entity_id: string;
  entity_type: "identity" | "asset" | "collector";
  baseline: BehaviourBaselineContext;
  model_version: string;
  model_kind: string;
  anomaly_score: number;
  severity: "low" | "medium" | "high";
  anomalous: boolean;
  threshold: number;
  explanations: BehaviourExplanation[];
  scored_at: string;
};

export type BehaviourScoreListResponse = {
  count: number;
  scores: BehaviourScore[];
};

export type BehaviourMetrics = {
  total_scores: number;
  anomalous_scores: number;
  high_severity_scores: number;
  average_score: number;
  last_scored_at?: string;
};

export type BehaviourSettings = {
  anomaly_threshold: number;
  updated_by?: string;
  updated_at: string;
};

export type BehaviourModelRecord = {
  model_version: string;
  model_kind: string;
  status: "active" | "retired";
  feature_schema: string[];
  training_source: string;
  random_seed?: number;
  registered_at: string;
};

export type BehaviourModelListResponse = {
  count: number;
  models: BehaviourModelRecord[];
};

export type BehaviourEvaluationRun = {
  id: number;
  model_version: string;
  dataset_name: string;
  threshold: number;
  normal_count: number;
  anomaly_count: number;
  true_positive: number;
  false_positive: number;
  true_negative: number;
  false_negative: number;
  precision: number;
  recall: number;
  false_positive_rate: number;
  actor_id?: string;
  created_at: string;
};

export type BehaviourEvaluationListResponse = {
  count: number;
  evaluations: BehaviourEvaluationRun[];
};
