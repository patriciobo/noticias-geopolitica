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

export type SubscribeResult = { ok: true } | { ok: false; message: string };

// A diferencia de apiFetch (pensado para fetches de página en Server
// Components, que lanzan excepción), este flujo es un formulario
// interactivo que necesita distinguir 400 (email inválido) de 500 (error
// de servidor) para mostrar un mensaje inline sin romper el render.
//
// Pega a la propia route interna de Next.js (/api/subscribe, mismo origen)
// en vez de a NOTICIAS_API_URL directo: esa var es server-only (Vercel no
// deja exponerla con NEXT_PUBLIC_ en este proyecto) y de paso evita tener
// que configurar CORS en core/cmd/api — la route interna reenvía server-side.
export async function subscribeEmail(email: string): Promise<SubscribeResult> {
  try {
    const res = await fetch("/api/subscribe", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });
    if (!res.ok) {
      if (res.status === 400) {
        return { ok: false, message: "Ese email no parece válido." };
      }
      return { ok: false, message: "No se pudo completar la suscripción. Probá de nuevo." };
    }
    return { ok: true };
  } catch {
    return { ok: false, message: "No se pudo conectar con el servidor. Probá de nuevo." };
  }
}
