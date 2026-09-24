import type { Metadata } from "next";
import Link from "next/link";
import { fetchCorrections, type Correction } from "@/lib/api";
import { REPO_URL } from "@/lib/transparency";
import { formatDate } from "@/components/ReportView";
import styles from "../metodologia/page.module.css";

export const metadata: Metadata = {
  title: "Correcciones — Radar Global",
  description: "Ediciones de Radar Global que se volvieron a generar, cuándo y por qué.",
};

// Ver REVALIDATE_SECONDS en lib/api.ts (acá tiene que ser un literal).
export const revalidate = 300;

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("es-AR", {
    timeZone: "America/Argentina/Buenos_Aires",
    dateStyle: "short",
    timeStyle: "short",
  });
}

export default async function CorreccionesPage() {
  let corrections: Correction[] | null = null;
  try {
    corrections = await fetchCorrections();
  } catch {
    corrections = null;
  }

  return (
    <main className={styles.page}>
      <h1>Correcciones</h1>
      <p className={styles.lead}>
        Cuando una edición sale mal (por una falla técnica, un feed roto o un error del modelo), se vuelve a
        generar. Nunca se reemplaza en silencio: la edición muestra su número de versión y el motivo, y esta
        página lista todas las regeneraciones.
      </p>

      <section>
        <h2>Política</h2>
        <ul>
          <li>Las ediciones no se editan a mano: una corrección es siempre una nueva generación automática.</li>
          <li>Cada versión queda firmada y en el historial público del repositorio, junto con las anteriores.</li>
          <li>
            Si encontrás un error, <a href={`${REPO_URL}/issues/new/choose`}>reportalo</a>. Si lleva a una
            regeneración, el motivo se publica acá.
          </li>
        </ul>
      </section>

      <section>
        <h2>Ediciones regeneradas</h2>
        {corrections === null ? (
          <p>La lista no está disponible en este momento.</p>
        ) : corrections.length === 0 ? (
          <p>Todavía no hay ediciones regeneradas con el registro de versiones.</p>
        ) : (
          <ul>
            {corrections.map((c) => (
              <li key={c.date}>
                <Link href={`/reportes/${c.date}`}>{formatDate(c.date)}</Link>
                <ul>
                  {c.revisions.map((r) => (
                    <li key={r.version}>
                      Versión {r.version} — {formatTime(r.generated_at)}: {r.reason ?? "sin motivo declarado"}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2>Antes del registro de versiones</h2>
        <ul>
          <li>
            <Link href="/reportes/2026-09-24">Edición del 24 de septiembre de 2026</Link>: se generó tres veces. La
            primera clasificó solo 9 titulares (un filtro por palabras clave descartó casi todo y Europa y Asia
            Oriental quedaron vacías por una falla técnica); la segunda salió cortada a mitad del texto y así se
            envió por mail; la tercera es la que se publica. Además incluyó dos notas viejas (de 2018 y 2020)
            servidas por feeds congelados. Las tres versiones están en el historial del repositorio.
          </li>
        </ul>
      </section>
    </main>
  );
}
