const API_BASE_URL = process.env.NOTICIAS_API_URL ?? "http://localhost:8080";

export type SourceSummary = {
  name: string;
  country: string;
};

// Cómo se generó una edición (ver model.Provenance en core). Ausente en
// ediciones anteriores al registro de auditoría.
export type Provenance = {
  generated_at: string;
  commit?: string;
  run_url?: string;
  provider: string;
  classify_models: string[];
  synthesize_models: string[];
  prompt_sha256: Record<string, string>;
  counts: {
    sources_configured?: number; // ausentes en ediciones anteriores al 2026-09-25
    sources_responded?: number;
    fetched: number;
    prefilter_rejected: number;
    classifier_rejected: number;
    classifier_errors: number;
    accepted: number;
    links_removed: number;
  };
  source_problems?: SourceProblem[];
};

// Medio que no aportó titulares en la corrida (ver model.SourceStatus).
export type SourceProblem = {
  name: string;
  country: string;
  region: string;
  status: "error" | "sin_vigentes";
  detail?: string;
};

export type ReportResponse = {
  date: string;
  markdown: string;
  sources: SourceSummary[];
  source_count: number;
  provenance?: Provenance;
};

// Un medio de config/sources.yaml tal como lo publica GET /sources.
export type PublicSource = {
  name: string;
  country: string;
  region: string;
  stance: string;
  ownership: string; // estatal | privado | partidario | ong | exilio
  ownership_note?: string;
  homepage: string;
  has_feed: boolean;
};

export class ReportNotFoundError extends Error {}

// Cada cuántos segundos se revalidan las páginas y los fetches a la API.
// Los reportes salen una vez por día: con 5 minutos de cache, un pico de
// visitas (ej. un posteo que se viraliza) lo absorbe el CDN de Vercel en
// vez de pegarle a Render (instancia gratis que se duerme) por cada
// visitante. Las páginas exportan el mismo valor como `revalidate` —
// tiene que ser un literal ahí, por eso no se importa esta constante.
export const REVALIDATE_SECONDS = 300;

async function apiFetch(path: string): Promise<Response> {
  const res = await fetch(`${API_BASE_URL}${path}`, { next: { revalidate: REVALIDATE_SECONDS } });
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

export async function fetchSources(): Promise<PublicSource[]> {
  const res = await apiFetch("/sources");
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
export type SubscribePayload = {
  email: string;
  turnstileToken?: string; // respuesta del captcha de Cloudflare Turnstile, si está habilitado
  website?: string; // honeypot: campo oculto que un humano deja vacío
};

export async function subscribeEmail(payload: SubscribePayload): Promise<SubscribeResult> {
  try {
    const res = await fetch("/api/subscribe", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      if (res.status === 400) {
        return { ok: false, message: "Ese email no parece válido." };
      }
      if (res.status === 403) {
        return { ok: false, message: "No pudimos verificar que no seas un bot. Recargá la página y probá de nuevo." };
      }
      if (res.status === 429) {
        return { ok: false, message: "Demasiados intentos. Probá de nuevo en un rato." };
      }
      if (res.status === 503) {
        return { ok: false, message: "Las suscripciones están pausadas momentáneamente. Probá más tarde." };
      }
      return { ok: false, message: "No se pudo completar la suscripción. Probá de nuevo." };
    }
    return { ok: true };
  } catch {
    return { ok: false, message: "No se pudo conectar con el servidor. Probá de nuevo." };
  }
}
