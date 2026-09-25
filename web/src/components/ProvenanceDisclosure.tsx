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
              {provenance.counts.sources_configured !== undefined && (
                <li>
                  <strong>{provenance.counts.sources_responded}</strong> de {provenance.counts.sources_configured} medios
                  respondieron con titulares recientes
                </li>
              )}
              <li><strong>{provenance.counts.fetched}</strong> titulares descargados de los feeds</li>
              <li><strong>{provenance.counts.prefilter_rejected}</strong> descartados por el prefiltro de palabras clave</li>
              <li><strong>{provenance.counts.classifier_rejected}</strong> descartados por el clasificador por no tener alcance internacional</li>
              {provenance.counts.classifier_errors > 0 && (
                <li><strong>{provenance.counts.classifier_errors}</strong> no se pudieron clasificar (error técnico) y quedaron afuera</li>
              )}
              <li><strong>{provenance.counts.accepted}</strong> usados para escribir el informe</li>
              {provenance.counts.citations_resolved !== undefined && (
                <li>
                  <strong>{provenance.counts.citations_resolved}</strong> citas enlazadas a su nota
                  {(provenance.counts.uncited_bullets ?? 0) > 0 &&
                    `; ${provenance.counts.uncited_bullets} afirmaciones quedaron sin cita`}
                  {(provenance.counts.citations_invalid ?? 0) > 0 &&
                    `; ${provenance.counts.citations_invalid} citas del modelo apuntaban a notas inexistentes y se sacaron`}
                </li>
              )}
              {(provenance.counts.claims_checked ?? 0) > 0 && (
                <li>
                  Chequeo de fidelidad: de <strong>{provenance.counts.claims_checked}</strong> afirmaciones con cita,{" "}
                  {provenance.counts.claims_supported} están respaldadas por sus notas, {provenance.counts.claims_inference} son
                  análisis a partir de ellas y <strong>{provenance.counts.claims_unsupported}</strong> no encontraron respaldo
                </li>
              )}
              {provenance.counts.links_removed > 0 && (
                <li><strong>{provenance.counts.links_removed}</strong> enlaces escritos por el modelo se eliminaron por no corresponder a una nota procesada</li>
              )}
            </ul>
            {provenance.source_problems && provenance.source_problems.length > 0 && (
              <details className={styles.problems}>
                <summary>Medios que no aportaron titulares hoy ({provenance.source_problems.length})</summary>
                <ul>
                  {provenance.source_problems.map((p) => (
                    <li key={p.name}>
                      {p.name} ({p.country}) —{" "}
                      {p.status === "error" ? "no se pudo descargar su feed" : "su feed no trajo notas recientes"}
                    </li>
                  ))}
                </ul>
              </details>
            )}
            {provenance.fidelity_issues && provenance.fidelity_issues.length > 0 && (
              <details className={styles.problems}>
                <summary>Afirmaciones sin respaldo en sus notas ({provenance.fidelity_issues.length})</summary>
                <ul>
                  {provenance.fidelity_issues.map((f, i) => (
                    <li key={i}>
                      <em>{f.text}</em> (notas {f.citations.join(", ")}) — {f.problem}
                    </li>
                  ))}
                </ul>
                <p>El chequeo lo hace un modelo de lenguaje y también puede equivocarse.</p>
              </details>
            )}
            <ModelsUsed provenance={provenance} />
            {provenance.cost_usd !== undefined && provenance.cost_usd > 0 && (
              <p>
                Costo de esta edición en modelos de lenguaje: USD{" "}
                {provenance.cost_usd.toLocaleString("es-AR", { minimumFractionDigits: 3, maximumFractionDigits: 3 })}.
              </p>
            )}
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
          <Link href="/metodologia">Metodología completa</Link> · <Link href="/correcciones">Correcciones</Link>
        </p>
      </div>
    </details>
  );
}

// Qué modelos se usaron de verdad. Con una cadena de respaldo, el principal
// configurado puede no ser el que clasificó o redactó: decirlo evita que
// una edición escrita por un modelo de respaldo pase por la del principal.
function ModelsUsed({ provenance }: { provenance: Provenance }) {
  const primaryClassify = provenance.classify_models[0];
  const primarySynth = provenance.synthesize_models[0];
  const used = provenance.classify_models_used;
  const synthUsed = provenance.synthesize_model_used;

  if (!used && !synthUsed) {
    return (
      <p>
        Modelos: clasificación con <code>{primaryClassify}</code>, redacción con <code>{primarySynth}</code>
        {(provenance.classify_models.length > 1 || provenance.synthesize_models.length > 1) &&
          " (con modelos de respaldo si el principal falla, listados en el registro)"}
        .
      </p>
    );
  }

  const usedEntries = Object.entries(used ?? {}).sort((a, b) => b[1] - a[1]);
  const fallbackUsed =
    usedEntries.some(([m]) => m !== primaryClassify) || (synthUsed !== undefined && synthUsed !== primarySynth);

  return (
    <>
      <p>
        Clasificación:{" "}
        {usedEntries.map(([m, n], i) => (
          <span key={m}>
            {i > 0 && ", "}
            <code>{m}</code> ({n} titulares)
          </span>
        ))}
        . Redacción: <code>{synthUsed ?? primarySynth}</code>.
      </p>
      {fallbackUsed && (
        <p>
          En esta edición se usó al menos un modelo de respaldo porque el principal (<code>{primaryClassify}</code>
          {primarySynth !== primaryClassify && (
            <>
              {" "}/ <code>{primarySynth}</code>
            </>
          )}
          ) falló en parte de la corrida.
        </p>
      )}
    </>
  );
}
