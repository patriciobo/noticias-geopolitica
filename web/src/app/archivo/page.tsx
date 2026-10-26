import type { Metadata } from "next";
import Link from "next/link";
import { fetchReportDates, rethrowOutsideBuild } from "@/lib/api";
import { formatDate } from "@/components/ReportView";
import styles from "../metodologia/page.module.css";

export const metadata: Metadata = {
  title: "Archivo — Radar Global",
  description: "Todas las ediciones de Radar Global, ordenadas por mes.",
};

// Ver REVALIDATE_SECONDS en lib/api.ts (acá tiene que ser un literal).
export const revalidate = 300;

// Agrupa las fechas (YYYY-MM-DD, más recientes primero) por mes, manteniendo
// el orden.
function groupByMonth(dates: string[]): [string, string[]][] {
  const months = new Map<string, string[]>();
  for (const d of dates) {
    const key = d.slice(0, 7);
    months.set(key, [...(months.get(key) ?? []), d]);
  }
  return [...months];
}

function formatMonth(yearMonth: string): string {
  const label = new Date(`${yearMonth}-01T00:00:00`).toLocaleDateString("es-AR", {
    month: "long",
    year: "numeric",
  });
  return label.charAt(0).toUpperCase() + label.slice(1);
}

export default async function ArchivoPage() {
  let dates: string[] | null = null;
  try {
    dates = await fetchReportDates();
  } catch (err) {
    rethrowOutsideBuild(err);
    dates = null;
  }

  return (
    <main id="contenido" className={styles.page}>
      <h1>Archivo</h1>
      <p className={styles.lead}>
        El inicio muestra las ediciones de la última semana. Acá están todas, desde la primera.
      </p>

      {dates === null ? (
        <p>No se pudo cargar el archivo. Probá de nuevo en unos minutos.</p>
      ) : dates.length === 0 ? (
        <p>Todavía no hay ediciones.</p>
      ) : (
        groupByMonth(dates).map(([month, days]) => (
          <section key={month} aria-labelledby={`mes-${month}`}>
            <h2 id={`mes-${month}`}>{formatMonth(month)}</h2>
            <ul>
              {days.map((d) => (
                <li key={d}>
                  <Link href={`/reportes/${d}`}>{formatDate(d)}</Link>
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
    </main>
  );
}
