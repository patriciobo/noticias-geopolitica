export type Section = { heading: string; body: string };

/**
 * Divide el cuerpo del reporte (ya sin "Resumen ejecutivo" ni "Noticias
 * utilizadas", ver splitAbstract/splitNewsLinks) en sus secciones de nivel 2
 * ("## ..."), para poder darle un layout distinto a "Clima internacional" y
 * "Empresas potencialmente afectadas por región" sin tocar el contenido.
 */
export function splitTopSections(markdown: string): Section[] {
  const matches = [...markdown.matchAll(/^## .+$/gm)];
  return matches.map((m, i) => {
    const start = m.index!;
    const end = i + 1 < matches.length ? matches[i + 1].index! : markdown.length;
    return {
      heading: m[0].slice(3).trim(),
      body: markdown.slice(start + m[0].length, end).trim(),
    };
  });
}

/**
 * Divide el cuerpo de una sección en sus subsecciones de nivel 3 ("### ...").
 * Cantidad variable a propósito: el LLM decide cuántas hay (ej. "Comercio",
 * "Industria", "Materias primas" u otras) y cuántas regiones tienen contenido
 * — el layout que consuma esto no debe asumir un número fijo.
 */
export function splitSubsections(markdown: string): Section[] {
  const matches = [...markdown.matchAll(/^### .+$/gm)];
  if (matches.length === 0) return [];
  return matches.map((m, i) => {
    const start = m.index!;
    const end = i + 1 < matches.length ? matches[i + 1].index! : markdown.length;
    return {
      heading: m[0].slice(4).trim(),
      body: markdown.slice(start + m[0].length, end).trim(),
    };
  });
}
