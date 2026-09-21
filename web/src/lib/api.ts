const API_BASE_URL = process.env.NOTICIAS_API_URL ?? "http://localhost:8080";

export type SourceSummary = {
  name: string;
  country: string;
};

export type ReportResponse = {
  date: string;
  markdown: string;
  sources: SourceSummary[];
  source_count: number;
};

export class ReportNotFoundError extends Error {}

async function apiFetch(path: string): Promise<Response> {
  const res = await fetch(`${API_BASE_URL}${path}`, { cache: "no-store" });
  if (res.status === 404) {
    throw new ReportNotFoundError(`No se encontró recurso en ${path}`);
  }
  if (!res.ok) {
    throw new Error(`La API de noticias respondió ${res.status} para ${path}`);
  }
  return res;
}

export async function fetchLatestReport(): Promise<ReportResponse> {
  const res = await apiFetch("/reports/latest");
  return res.json();
}

export async function fetchReportByDate(date: string): Promise<ReportResponse> {
  const res = await apiFetch(`/reports/${date}`);
  return res.json();
}

export async function fetchReportDates(): Promise<string[]> {
  const res = await apiFetch("/reports");
  return res.json();
}
