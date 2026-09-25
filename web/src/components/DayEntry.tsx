import type { ReportResponse } from "@/lib/api";
import { splitAbstract } from "@/lib/newsLinks";
import Abstract from "./Abstract";
import { readingLabel } from "@/lib/readingTime";
import { ReportBody, formatDate } from "./ReportView";
import styles from "./DayEntry.module.css";

function linkCount(markdown: string): number {
  const idx = markdown.indexOf("\n## Noticias utilizadas");
  if (idx === -1) return 0;
  return (markdown.slice(idx).match(/^- \[/gm) ?? []).length;
}

/**
 * Una edición diaria del blog, con borde propio para separarla de las demás.
 * El resumen ejecutivo va siempre visible (fuera del <details>), así se lee
 * sin abrir el informe completo — también en las ediciones plegadas.
 */
export default function DayEntry({
  report,
  latest = false,
}: {
  report: ReportResponse;
  latest?: boolean;
}) {
  const { abstract, body } = splitAbstract(report.markdown);
  const links = linkCount(report.markdown);
  const meta = [
    readingLabel(report.markdown),
    `${report.source_count} medios consultados`,
    links > 0 ? `${links} noticias con enlace` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  if (latest) {
    return (
      <article className={`${styles.entry} ${styles.latest}`} id={report.date}>
        <header className={styles.header}>
          <span className={styles.badge}>Última edición</span>
          <h2 className={styles.date}>{formatDate(report.date)}</h2>
          <p className={styles.meta}>{meta}</p>
        </header>
        {abstract && <Abstract markdown={abstract} />}
        <ReportBody date={report.date} markdown={body} sources={report.sources} provenance={report.provenance} />
      </article>
    );
  }

  return (
    <article className={styles.entry} id={report.date}>
      <header className={`${styles.header} ${styles.headerRow}`}>
        <span className={styles.date}>{formatDate(report.date)}</span>
        <span className={styles.meta}>{meta}</span>
      </header>
      {abstract && <Abstract markdown={abstract} />}
      <details className={styles.details}>
        <summary className={styles.summary}>
          Ver informe completo
          <span className={styles.toggle} aria-hidden="true" />
        </summary>
        <div className={styles.content}>
          <ReportBody date={report.date} markdown={body} sources={report.sources} provenance={report.provenance} />
        </div>
      </details>
    </article>
  );
}
