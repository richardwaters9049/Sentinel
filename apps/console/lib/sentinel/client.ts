import type {
  AssetPivot,
  EventListResponse,
  FindingDetail,
  FindingEvidenceContext,
  FindingListResponse,
  FindingStatus,
  DetectionListResponse,
  DetectionMetricsResponse,
  DetectionRecord,
  HuntDefinition,
  HuntListResponse,
  HuntQuery,
  HuntRunResult,
  InvestigationDetail,
  InvestigationListResponse,
  IntelligenceIndicatorListResponse,
  IntelligenceMatchListResponse,
  IntelligenceMetrics,
  IntelligenceSource,
  IntelligenceSourceListResponse,
  BehaviourMetrics,
  BehaviourScore,
  BehaviourScoreListResponse,
} from "@/lib/sentinel/types";

type APIErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

async function readJSON<T>(response: Response): Promise<T> {
  const payload = (await response.json()) as T & APIErrorPayload;

  if (!response.ok) {
    throw new Error(
      payload.error?.message ??
        `Sentinel request failed with status ${response.status}`,
    );
  }

  return payload;
}

export async function listInvestigations(
  signal?: AbortSignal,
): Promise<InvestigationListResponse> {
  const response = await fetch("/api/sentinel/api/v1/investigations?limit=50", {
    cache: "no-store",
    signal,
  });
  return readJSON<InvestigationListResponse>(response);
}

export async function getInvestigation(
  id: string,
  signal?: AbortSignal,
): Promise<InvestigationDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/investigations/${encodeURIComponent(id)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<InvestigationDetail>(response);
}

export async function listHunts(signal?: AbortSignal): Promise<HuntListResponse> {
  const response = await fetch("/api/sentinel/api/v1/hunts", {
    cache: "no-store",
    signal,
  });
  return readJSON<HuntListResponse>(response);
}

export async function createHunt(
  input: {
    name: string;
    description: string;
    hypothesis: string;
    query: HuntQuery;
  },
  actorID: string,
): Promise<HuntDefinition> {
  const response = await fetch("/api/sentinel/api/v1/hunts", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Sentinel-Actor": actorID,
    },
    body: JSON.stringify(input),
  });
  return readJSON<HuntDefinition>(response);
}

export async function runHunt(
  id: string,
  actorID: string,
  override: HuntQuery = {},
): Promise<HuntRunResult> {
  const response = await fetch(
    `/api/sentinel/api/v1/hunts/${encodeURIComponent(id)}/run`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ override }),
    },
  );
  return readJSON<HuntRunResult>(response);
}

export async function listEvents(
  limit = 120,
  signal?: AbortSignal,
): Promise<EventListResponse> {
  const response = await fetch(
    `/api/sentinel/api/v1/events?limit=${Math.min(Math.max(limit, 1), 200)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<EventListResponse>(response);
}

export async function listFindings(
  limit = 120,
  signal?: AbortSignal,
): Promise<FindingListResponse> {
  const response = await fetch(
    `/api/sentinel/api/v1/findings?limit=${Math.min(Math.max(limit, 1), 200)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<FindingListResponse>(response);
}

export async function getFinding(
  id: string,
  signal?: AbortSignal,
): Promise<FindingDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/findings/${encodeURIComponent(id)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<FindingDetail>(response);
}

export async function getFindingEvidence(
  id: string,
  contextMinutes = 5,
  signal?: AbortSignal,
): Promise<FindingEvidenceContext> {
  const response = await fetch(
    `/api/sentinel/api/v1/findings/${encodeURIComponent(id)}/evidence?context_minutes=${Math.min(Math.max(contextMinutes, 0), 60)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<FindingEvidenceContext>(response);
}

export async function updateFindingStatus(
  id: string,
  status: FindingStatus,
  actorID: string,
): Promise<FindingDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/findings/${encodeURIComponent(id)}/status`,
    {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ status }),
    },
  );
  return readJSON<FindingDetail>(response);
}

export async function listDetections(
  signal?: AbortSignal,
): Promise<DetectionListResponse> {
  const response = await fetch("/api/sentinel/api/v1/detections", {
    cache: "no-store",
    signal,
  });
  return readJSON<DetectionListResponse>(response);
}

export async function getDetectionMetrics(
  signal?: AbortSignal,
): Promise<DetectionMetricsResponse> {
  const response = await fetch("/api/sentinel/api/v1/detections/metrics", {
    cache: "no-store",
    signal,
  });
  return readJSON<DetectionMetricsResponse>(response);
}

export async function setDetectionEnabled(
  id: string,
  enabled: boolean,
  actorID: string,
): Promise<DetectionRecord> {
  const response = await fetch(
    `/api/sentinel/api/v1/detections/${encodeURIComponent(id)}`,
    {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ enabled }),
    },
  );
  return readJSON<DetectionRecord>(response);
}


export async function listIntelligenceIndicators(
  signal?: AbortSignal,
): Promise<IntelligenceIndicatorListResponse> {
  const response = await fetch(
    "/api/sentinel/api/v1/intelligence/indicators?limit=100",
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<IntelligenceIndicatorListResponse>(response);
}


export async function listIntelligenceSources(
  signal?: AbortSignal,
): Promise<IntelligenceSourceListResponse> {
  const response = await fetch("/api/sentinel/api/v1/intelligence/sources", {
    cache: "no-store",
    signal,
  });
  return readJSON<IntelligenceSourceListResponse>(response);
}

export async function setIntelligenceSourceEnabled(
  id: string,
  enabled: boolean,
  actorID: string,
): Promise<IntelligenceSource> {
  const response = await fetch(
    `/api/sentinel/api/v1/intelligence/sources/${encodeURIComponent(id)}`,
    {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ enabled }),
    },
  );
  return readJSON<IntelligenceSource>(response);
}

export async function listIntelligenceMatches(
  limit = 50,
  signal?: AbortSignal,
): Promise<IntelligenceMatchListResponse> {
  const response = await fetch(
    `/api/sentinel/api/v1/intelligence/matches?limit=${Math.min(Math.max(limit, 1), 200)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<IntelligenceMatchListResponse>(response);
}

export async function getIntelligenceMetrics(
  signal?: AbortSignal,
): Promise<IntelligenceMetrics> {
  const response = await fetch("/api/sentinel/api/v1/intelligence/metrics", {
    cache: "no-store",
    signal,
  });
  return readJSON<IntelligenceMetrics>(response);
}

export async function getAssetPivot(
  id: string,
  limit = 50,
  signal?: AbortSignal,
): Promise<AssetPivot> {
  const response = await fetch(
    `/api/sentinel/api/v1/assets/${encodeURIComponent(id)}/pivot?limit=${Math.min(Math.max(limit, 1), 200)}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<AssetPivot>(response);
}

export async function createInvestigation(
  input: {
    title: string;
    description: string;
    priority: "low" | "medium" | "high" | "critical";
    owner_id?: string;
    finding_ids?: string[];
    event_ids?: string[];
  },
  actorID: string,
): Promise<InvestigationDetail> {
  const response = await fetch("/api/sentinel/api/v1/investigations", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Sentinel-Actor": actorID,
    },
    body: JSON.stringify(input),
  });
  return readJSON<InvestigationDetail>(response);
}

export async function attachHuntRunToInvestigation(
  investigationID: string,
  runID: number,
  actorID: string,
): Promise<InvestigationDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/investigations/${encodeURIComponent(investigationID)}/hunt-runs/${runID}`,
    {
      method: "POST",
      headers: {
        "X-Sentinel-Actor": actorID,
      },
    },
  );
  return readJSON<InvestigationDetail>(response);
}

export async function updateInvestigationMetadata(
  id: string,
  input: {
    owner_id?: string;
    priority?: "low" | "medium" | "high" | "critical";
  },
  actorID: string,
): Promise<InvestigationDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/investigations/${encodeURIComponent(id)}`,
    {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify(input),
    },
  );
  return readJSON<InvestigationDetail>(response);
}

export async function updateInvestigationStatus(
  id: string,
  status: "open" | "investigating" | "contained" | "closed",
  actorID: string,
): Promise<InvestigationDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/investigations/${encodeURIComponent(id)}/status`,
    {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ status }),
    },
  );
  return readJSON<InvestigationDetail>(response);
}

export async function addInvestigationNote(
  id: string,
  body: string,
  actorID: string,
): Promise<InvestigationDetail> {
  const response = await fetch(
    `/api/sentinel/api/v1/investigations/${encodeURIComponent(id)}/notes`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Sentinel-Actor": actorID,
      },
      body: JSON.stringify({ body }),
    },
  );
  return readJSON<InvestigationDetail>(response);
}


export type BehaviourEntityFilter = {
  entity_id: string;
  entity_type: BehaviourScore["entity_type"];
};

export async function getBehaviourMetrics(
  signal?: AbortSignal,
  entity?: BehaviourEntityFilter,
): Promise<BehaviourMetrics> {
  const query = new URLSearchParams(entity ?? {});
  const response = await fetch(`/api/sentinel/api/v1/behaviour/metrics?${query}`, {
    cache: "no-store",
    signal,
  });
  return readJSON<BehaviourMetrics>(response);
}

export async function listBehaviourScores(
  limit = 50,
  signal?: AbortSignal,
  entity?: BehaviourEntityFilter,
): Promise<BehaviourScoreListResponse> {
  const query = new URLSearchParams({
    ...entity,
    limit: String(Math.min(Math.max(limit, 1), 200)),
  });
  const response = await fetch(
    `/api/sentinel/api/v1/behaviour/scores?${query}`,
    {
      cache: "no-store",
      signal,
    },
  );
  return readJSON<BehaviourScoreListResponse>(response);
}

export async function getBehaviourSettings(
  signal?: AbortSignal,
): Promise<import("./types").BehaviourSettings> {
  const response = await fetch("/api/sentinel/api/v1/behaviour/settings", {
    cache: "no-store",
    signal,
  });
  return readJSON<import("./types").BehaviourSettings>(response);
}

export async function updateBehaviourSettings(
  anomalyThreshold: number,
): Promise<import("./types").BehaviourSettings> {
  const response = await fetch("/api/sentinel/api/v1/behaviour/settings", {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      "X-Sentinel-Actor": "sentinel-console",
    },
    body: JSON.stringify({ anomaly_threshold: anomalyThreshold }),
  });
  return readJSON<import("./types").BehaviourSettings>(response);
}

export async function listBehaviourModels(
  signal?: AbortSignal,
): Promise<import("./types").BehaviourModelListResponse> {
  const response = await fetch("/api/sentinel/api/v1/behaviour/models", {
    cache: "no-store",
    signal,
  });
  return readJSON<import("./types").BehaviourModelListResponse>(response);
}

export async function listBehaviourEvaluations(
  limit = 20,
  signal?: AbortSignal,
): Promise<import("./types").BehaviourEvaluationListResponse> {
  const bounded = Math.min(Math.max(limit, 1), 100);
  const response = await fetch(
    "/api/sentinel/api/v1/behaviour/evaluations?limit=" + String(bounded),
    { cache: "no-store", signal },
  );
  return readJSON<import("./types").BehaviourEvaluationListResponse>(response);
}

export async function getBehaviourMonitor(
  signal?: AbortSignal, entity?: BehaviourEntityFilter,
): Promise<import("./types").BehaviourMonitor> {
  const query = new URLSearchParams(entity ?? {});
  const response = await fetch(`/api/sentinel/api/v1/behaviour/monitor?${query}`, {
    cache: "no-store", signal,
  });
  return readJSON<import("./types").BehaviourMonitor>(response);
}

export async function getBehaviourCatalogue(
  signal?: AbortSignal,
): Promise<import("./types").BehaviourCatalogue> {
  const response = await fetch("/api/sentinel/api/v1/behaviour/catalogue", {
    cache: "no-store",
    signal,
  });
  return readJSON<import("./types").BehaviourCatalogue>(response);
}
