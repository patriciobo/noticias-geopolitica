import {
  fetchReportByDate,
  fetchReportDates,
  ReportNotFoundError,
  type ReportResponse,
} from "@/lib/api";
import DayEntry from "@/components/DayEntry";
import SubscribeForm from "@/components/SubscribeForm";
import styles from "./page.module.css";

export const metadata = {
  title: "Noticias Internacionales",
  description:
    "Resumen diario de la actualidad internacional y su impacto en el comercio y las empresas multinacionales.",
};

// Cuántas ediciones muestra el feed (más recientes primero).
const MAX_EDITIONS = 30;

export default async function Home() {
  let reports: ReportResponse[] = [];

  try {
    const dates = (await fetchReportDates()).slice(0, MAX_EDITIONS);
    if (dates.length === 0) throw new ReportNotFoundError("sin reportes");
    reports = await Promise.all(dates.map(fetchReportByDate));
  } catch (err) {
    if (err instanceof ReportNotFoundError) {
      return (
        <main className={styles.empty}>
          <h1>Noticias Internacionales</h1>
          <p>
            Todavía no se generó ningún reporte. Corré el pipeline con{" "}
            <code>go run ./cmd/ingest</code> desde <code>core/</code> y volvé a
            cargar esta página.
          </p>
        </main>
      );
    }
    return (
      <main className={styles.empty}>
        <h1>Noticias Internacionales</h1>
        <p>
          No se pudo conectar con la API del backend. Verificá que esté
          corriendo (<code>go run ./cmd/api</code> desde <code>core/</code>) y
          que <code>NOTICIAS_API_URL</code> apunte a la dirección correcta.
        </p>
      </main>
    );
  }

  const [latest, ...older] = reports;

  return (
    <main className={styles.blog}>
      <header className={styles.masthead}>
        <h1>Noticias Internacionales</h1>
        <p>
          Un informe por día: comercio, industria y empresas multinacionales,
          con enlaces a las notas originales de cada medio.
        </p>
        <SubscribeForm />
      </header>
      <div className={styles.feed}>
        <DayEntry report={latest} latest />
        {older.length > 0 && (
          <>
            <h2 className={styles.archiveTitle}>Ediciones anteriores</h2>
            {older.map((r) => (
              <DayEntry key={r.date} report={r} />
            ))}
          </>
        )}
      </div>
    </main>
  );
}
