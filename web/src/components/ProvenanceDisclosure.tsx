import Link from "next/link";
import type { Provenance } from "@/lib/api";
import { REPO_SLUG, repoFileURL, repoHistoryURL, repoTreeURL } from "@/lib/transparency";
import styles from "./ProvenanceDisclosure.module.css";

/**
 * "Cómo se hizo esta edición": los datos para que cualquiera pueda
 * verificar que el informe salió del pipeline automático tal cual, qué
 * titulares quedaron afuera y por qué. Mismo patrón plegable que
 * SourcesDisclosure.
 */
export default function ProvenanceDisclosure({
  date,
  provenance,
}: {
  date: string;
  provenance?: Provenance;
}) {
  const reportPath = `reports/${date}.md`;
  const auditPath = `reports/${date}.audit.json`;

  return (
    <details className={styles.disclosure}>
      <summary className={styles.summary}>Cómo se hizo esta edición</summary>
      <div className={styles.body}>
        {provenance ? (
          <>
            <p>
              Generada automáticamente el{" "}
              {new Date(provenance.generated_at).toLocaleString("es-AR", { timeZone: "UTC", dateStyle: "long", timeStyle: "short" })} (UTC),
              sin edición humana.
            </p>
            <ul className={styles.counts}>
              <li><strong>{provenance.counts.fetched}</strong> titulares descargados de los feeds</li>
              <li><strong>{provenance.counts.prefilter_rejected}</strong> descartados por el prefiltro de palabras clave</li>
              <li><strong>{provenance.counts.classifier_rejected}</strong> descartados por el clasificador por no tener alcance internacional</li>
              {provenance.counts.classifier_errors > 0 && (
                <li><strong>{provenance.counts.classifier_errors}</strong> no se pudieron clasificar (error técnico) y quedaron afuera</li>
              )}
              <li><strong>{provenance.counts.accepted}</strong> usados para escribir el informe</li>
              {provenance.counts.links_removed > 0 && (
                <li><strong>{provenance.counts.links_removed}</strong> enlaces escritos por el modelo se eliminaron por no corresponder a una nota procesada</li>
              )}
            </ul>
            <p>
              Modelos: clasificación con <code>{provenance.classify_models[0]}</code>, redacción con{" "}
              <code>{provenance.synthesize_models[0]}</code>
              {(provenance.classify_models.length > 1 || provenance.synthesize_models.length > 1) &&
                " (con modelos de respaldo si el principal falla, listados en el registro)"}
              .
            </p>
            <ul className={styles.links}>
              <li>
                <a href={repoFileURL(auditPath)} target="_blank" rel="noopener noreferrer">Registro completo</a>: cada titular descargado y qué pasó con él
              </li>
              {provenance.run_url && (
                <li>
                  <a href={provenance.run_url} target="_blank" rel="noopener noreferrer">Log de la corrida</a> en GitHub Actions
                </li>
              )}
              {provenance.commit && (
                <li>
                  <a href={repoTreeURL(provenance.commit)} target="_blank" rel="noopener noreferrer">Código exacto</a> que la generó (incluye los prompts)
                </li>
              )}
              <li>
                <a href={repoHistoryURL(reportPath)} target="_blank" rel="noopener noreferrer">Historial del archivo</a>: cualquier cambio posterior quedaría a la vista
              </li>
            </ul>
            <p className={styles.verify}>
              Verificá la firma: <code>gh attestation verify {reportPath} -R {REPO_SLUG}</code>
            </p>
          </>
        ) : (
          <p>
            Esta edición es anterior al registro de auditoría por edición. Igual podés ver el{" "}
            <a href={repoHistoryURL(reportPath)} target="_blank" rel="noopener noreferrer">historial del archivo</a> en el repositorio público.
          </p>
        )}
        <p>
          <Link href="/metodologia">Metodología completa</Link>
        </p>
      </div>
    </details>
  );
}
