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

export type Finding = {
  id: string;
  detection_id: string;
  detection_version: number;
  title: string;
  severity: string;
  confidence: number;
  status: string;
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
