import type {
  AssetPivot,
  EventListResponse,
  FindingListResponse,
  HuntDefinition,
  HuntListResponse,
  HuntQuery,
  HuntRunResult,
  InvestigationDetail,
  InvestigationListResponse,
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
