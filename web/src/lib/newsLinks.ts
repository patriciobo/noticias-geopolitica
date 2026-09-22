export type NewsLink = {
  title: string;
  url: string;
  outlet: string;
  country: string;
};

export type NewsLinkGroup = {
  region: string;
  links: NewsLink[];
};

// Debe coincidir con SourcesHeading en core/internal/report/sources.go.
const SOURCES_HEADING = "## Noticias utilizadas";

const LINK_LINE = /^- \[(.+)\]\(<(.+?)>\) — (.+) \(([^()]+)\)$/;

/**
 * Separa la sección "Noticias utilizadas" (generada por el backend, no por el
 * LLM) del resto del reporte para dibujarla como lista de enlaces propia.
 * Si el reporte no la trae (ediciones viejas) o no se puede parsear, devuelve
 * el markdown intacto y sin grupos.
 */
export function splitNewsLinks(markdown: string): {
  body: string;
  groups: NewsLinkGroup[];
} {
  const idx = markdown.indexOf(`\n${SOURCES_HEADING}`);
  if (idx === -1) return { body: markdown, groups: [] };

  const body = markdown.slice(0, idx).trimEnd();
  const groups: NewsLinkGroup[] = [];

  for (const raw of markdown.slice(idx).split("\n")) {
    const line = raw.trim();
    if (line.startsWith("### ")) {
      groups.push({ region: line.slice(4).trim(), links: [] });
      continue;
    }
    const m = LINK_LINE.exec(line);
    if (!m || groups.length === 0) continue;
    groups[groups.length - 1].links.push({
      title: m[1].replace(/\\([[\]])/g, "$1"),
      url: m[2],
      outlet: m[3],
      country: m[4],
    });
  }

  const parsed = groups.filter((g) => g.links.length > 0);
  if (parsed.length === 0) return { body: markdown, groups: [] };
  return { body, groups: parsed };
}

const ABSTRACT_HEADING = "## Resumen ejecutivo";

/**
 * Separa la sección "Resumen ejecutivo" (LLM, primera del reporte) del resto
 * del markdown, para mostrarla como copete siempre visible, incluso cuando
 * el informe completo está colapsado. Ediciones viejas no la traen: en ese
 * caso devuelve abstract null y el markdown intacto.
 */
export function splitAbstract(markdown: string): {
  abstract: string | null;
  body: string;
} {
  const trimmed = markdown.trimStart();
  if (!trimmed.startsWith(ABSTRACT_HEADING)) return { abstract: null, body: markdown };

  const afterHeading = trimmed.slice(ABSTRACT_HEADING.length);
  const nextIdx = afterHeading.indexOf("\n## ");
  const abstract = (nextIdx === -1 ? afterHeading : afterHeading.slice(0, nextIdx)).trim();
  if (!abstract) return { abstract: null, body: markdown };

  const body = nextIdx === -1 ? "" : afterHeading.slice(nextIdx + 1);
  return { abstract, body };
}

export function hostnameOf(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}
