import type {
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
