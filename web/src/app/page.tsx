import Image from "next/image";
import Link from "next/link";
import {
  fetchReportByDate,
  fetchReportDates,
  ReportNotFoundError,
  rethrowOutsideBuild,
  type ReportResponse,
} from "@/lib/api";
import DayEntry from "@/components/DayEntry";
import SubscribeForm from "@/components/SubscribeForm";
import DonationSupport from "@/components/DonationSupport";
import styles from "./page.module.css";

export const metadata = {
  title: "Radar Global",
  description:
    "Resumen diario de la actualidad internacional y su impacto en el comercio y las empresas multinacionales.",
};

// Ver REVALIDATE_SECONDS en lib/api.ts (acá tiene que ser un literal).
export const revalidate = 300;

// El feed muestra las ediciones de los últimos 7 días, contados desde la
// edición más reciente (no desde hoy: si el pipeline se frena unos días, el
// home no queda vacío). Las anteriores están en /archivo.
const FEED_DAYS = 7;

function withinFeed(dates: string[]): string[] {
  if (dates.length === 0) return dates;
  const cutoff = new Date(`${dates[0]}T00:00:00Z`);
  cutoff.setUTCDate(cutoff.getUTCDate() - (FEED_DAYS - 1));
  const min = cutoff.toISOString().slice(0, 10);
  return dates.filter((d) => d >= min);
}

export default async function Home() {
  let reports: ReportResponse[] = [];

  try {
    const dates = withinFeed(await fetchReportDates());
    if (dates.length === 0) throw new ReportNotFoundError("sin reportes");
    reports = await Promise.all(dates.map(fetchReportByDate));
  } catch (err) {
    if (err instanceof ReportNotFoundError) {
      return (
        <main id="contenido" className={styles.empty}>
          <h1>Noticias Internacionales</h1>
          <p>
            Todavía no se generó ningún reporte. Corré el pipeline con{" "}
            <code>go run ./cmd/ingest</code> desde <code>core/</code> y volvé a
            cargar esta página.
          </p>
        </main>
      );
    }
    rethrowOutsideBuild(err);
    return (
      <main id="contenido" className={styles.empty}>
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
    <main id="contenido" className={styles.blog}>
      <header className={styles.masthead}>
        <div className={styles.mastheadBrand}>
          <Image src="/logo.png" alt="" width={1024} height={1024} className={styles.mastheadLogo} />
          <Image src="/nombre-sf.png" alt="" width={1536} height={1152} className={styles.mastheadName} />
        </div>
        <h1>Noticias Internacionales</h1>
        <p>
          Un informe por día: comercio, industria y empresas multinacionales,
          con enlaces a las notas originales de cada medio.
        </p>
        <SubscribeForm turnstileSiteKey={process.env.TURNSTILE_SITE_KEY} />
        <DonationSupport />
      </header>
      <div className={styles.feed}>
        <DayEntry report={latest} latest />
        {older.length > 0 && (
          <section className={styles.archive} id="ediciones-anteriores" aria-labelledby="ediciones-anteriores-titulo">
            <h2 className={styles.archiveTitle} id="ediciones-anteriores-titulo">Ediciones anteriores</h2>
            <div className={styles.archiveList}>
              {older.map((r) => (
                <DayEntry key={r.date} report={r} />
              ))}
            </div>
            <Link href="/archivo" className={styles.archiveMore}>
              Ver todas las ediciones en el archivo →
            </Link>
          </section>
        )}
        {older.length === 0 && (
          <Link href="/archivo" className={styles.archiveMore}>
            Ver todas las ediciones en el archivo →
          </Link>
        )}
      </div>
    </main>
  );
}
