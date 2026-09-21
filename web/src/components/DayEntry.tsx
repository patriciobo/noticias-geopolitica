import type { ReportResponse } from "@/lib/api";
import { ReportBody, formatDate } from "./ReportView";
import styles from "./DayEntry.module.css";

function linkCount(markdown: string): number {
  const idx = markdown.indexOf("\n## Noticias utilizadas");
  if (idx === -1) return 0;
  return (markdown.slice(idx).match(/^- \[/gm) ?? []).length;
}

/** Una edición diaria del blog, con borde propio para separarla de las demás. */
export default function DayEntry({
  report,
  latest = false,
}: {
  report: ReportResponse;
  latest?: boolean;
}) {
  const links = linkCount(report.markdown);
  const meta = [
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
        <ReportBody markdown={report.markdown} sources={report.sources} />
      </article>
    );
  }

  return (
    <details className={styles.entry} id={report.date}>
      <summary className={styles.summary}>
        <span className={styles.date}>{formatDate(report.date)}</span>
        <span className={styles.meta}>{meta}</span>
        <span className={styles.toggle} aria-hidden="true" />
      </summary>
      <div className={styles.content}>
        <ReportBody markdown={report.markdown} sources={report.sources} />
      </div>
    </details>
  );
}
