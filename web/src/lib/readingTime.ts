// Palabras por minuto de lectura en español para texto periodístico.
const WORDS_PER_MINUTE = 200;

/**
 * Minutos estimados para leer la edición completa: cuenta las palabras del
 * informe sin la lista "Noticias utilizadas" (que no se lee de corrido) y
 * sin la sintaxis de markdown, links ni citas. Mismo cálculo que
 * newsletter.ReadingMinutes en core.
 */
export function readingMinutes(markdown: string): number {
  const idx = markdown.indexOf("\n## Noticias utilizadas");
  const body = idx === -1 ? markdown : markdown.slice(0, idx);
  const text = body
    .replace(/\[\[\d+\]\]\(<[^>]*>\)/g, " ") // citas [[n]](<url>)
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1") // links: queda el texto
    .replace(/[#*_>`|-]/g, " ");
  const words = text.split(/\s+/).filter((w) => /[\p{L}\p{N}]/u.test(w)).length;
  return Math.max(1, Math.round(words / WORDS_PER_MINUTE));
}

export function readingLabel(markdown: string): string {
  return `${readingMinutes(markdown)} min de lectura`;
}
