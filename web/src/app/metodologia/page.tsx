import type { Metadata } from "next";
import { fetchSources, type PublicSource } from "@/lib/api";
import { REGION_LABELS } from "@/lib/regionTheme";
import { REPO_SLUG, REPO_URL, repoFileURL } from "@/lib/transparency";
import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Metodología — Radar Global",
  description:
    "Cómo se arma cada informe de Radar Global, qué medios se consultan y cómo verificar que ningún informe se edita a mano.",
};

// Ver REVALIDATE_SECONDS en lib/api.ts (acá tiene que ser un literal).
export const revalidate = 300;

const OWNERSHIP_LABELS: Record<string, string> = {
  estatal: "medio estatal",
  privado: "privado",
  partidario: "partidario",
  ong: "sin fines de lucro",
  exilio: "en el exilio",
};

const STANCE_LABELS: Record<string, string> = {
  oficialista: "Oficialista",
  oposicion: "Opositor",
};

type RegionRow = { region: string; countries: Set<string>; byStance: Record<string, number>; total: number };

function summarize(sources: PublicSource[]) {
  const active = sources.filter((s) => s.has_feed);
  const rows = new Map<string, RegionRow>();
  for (const s of active) {
    const row = rows.get(s.region) ?? { region: s.region, countries: new Set(), byStance: {}, total: 0 };
    row.countries.add(s.country);
    row.byStance[s.stance] = (row.byStance[s.stance] ?? 0) + 1;
    row.total++;
    rows.set(s.region, row);
  }
  const order = Object.keys(REGION_LABELS);
  const sorted = [...rows.values()].sort((a, b) => order.indexOf(a.region) - order.indexOf(b.region));
  const stances = [...new Set(active.map((s) => s.stance))].sort();
  return { active, sorted, stances, inactive: sources.length - active.length };
}

export default async function MetodologiaPage() {
  let sources: PublicSource[] | null = null;
  try {
    sources = await fetchSources();
  } catch {
    sources = null;
  }
  const summary = sources ? summarize(sources) : null;

  return (
    <main className={styles.page}>
      <h1>Metodología</h1>
      <p className={styles.lead}>
        Radar Global es un resumen diario generado de forma automática. Ninguna persona elige, edita ni
        reescribe las noticias de un informe. Esta página explica cómo se arma cada edición y cómo podés
        comprobarlo vos mismo, sin tener que confiar en nuestra palabra.
      </p>

      <section>
        <h2>Qué es y qué no es</h2>
        <ul>
          <li>
            <strong>Es</strong> un resumen de lo que publicaron ese día los medios de la lista, seleccionado y
            redactado por un modelo de lenguaje, con cada dato atribuido al medio que lo publicó.
          </li>
          <li>
            <strong>No es</strong> una verificación de los hechos. El modelo lee el titular y el copete de
            cada nota (no la nota completa) y no consulta otras fuentes: si un medio publica algo falso, el
            resumen lo va a repetir, atribuido a ese medio.
          </li>
          <li>
            <strong>Que varios medios publiquen lo mismo no lo confirma:</strong> muchas veces repiten el
            mismo cable de una agencia.
          </li>
          <li>
            <strong>Los medios estatales</strong> (por ejemplo, agencias oficiales de gobiernos) se incluyen
            para mostrar la posición oficial de su país, no como fuentes independientes.
          </li>
        </ul>
      </section>

      <section>
        <h2>Cómo se arma cada informe</h2>
        <ol>
          <li>
            <strong>Descarga.</strong> Todos los días, temprano a la mañana (antes de las 08:00, hora argentina), un proceso automático en
            GitHub Actions descarga los titulares más recientes de cada medio de la lista (hasta 6 por medio),
            desde sus feeds RSS públicos.
          </li>
          <li>
            <strong>Clasificación.</strong> Un modelo de lenguaje decide si cada titular, en su idioma original, tiene alcance
            internacional real, y deja por escrito el motivo.
          </li>
          <li>
            <strong>Redacción.</strong> Otro modelo escribe el informe solo a partir de los titulares aceptados,
            con instrucciones de no inventar datos y de señalar sin tomar partido cuando medios oficialistas y
            opositores de un mismo país encuadran distinto un hecho. Los prompts (las instrucciones exactas que
            recibe el modelo) están en el código público.
          </li>
          <li>
            <strong>Enlaces.</strong> La lista de &quot;Noticias utilizadas&quot; la arma el código, no el
            modelo. Si el modelo escribe un enlace que no corresponde a una nota procesada, se elimina
            automáticamente.
          </li>
          <li>
            <strong>Publicación.</strong> El informe, la lista de medios consultados y un registro de auditoría
            se guardan en el repositorio público con una firma digital, y el sitio los muestra tal cual.
          </li>
        </ol>
      </section>

      <section>
        <h2>Cómo verificarlo</h2>
        <ul>
          <li>
            <strong>Registro de cada edición.</strong> Cada informe tiene un archivo{" "}
            <code>reports/AAAA-MM-DD.audit.json</code> con todos los titulares descargados ese día y qué pasó
            con cada uno: descartado por el prefiltro, descartado por el clasificador (con su motivo) o usado.
            Así se puede ver no solo lo que se publicó, sino también lo que quedó afuera y por qué. Lo encontrás
            en el bloque &quot;Cómo se hizo esta edición&quot; al pie de cada informe.
          </li>
          <li>
            <strong>Firma digital.</strong> Cada archivo de la edición se firma (Sigstore) en el momento en que
            lo genera el proceso automático. Si alguien lo modificara después, aunque sea una coma, la firma
            dejaría de coincidir. Para verificarlo:{" "}
            <code>gh attestation verify reports/AAAA-MM-DD.md -R {REPO_SLUG}</code>
          </li>
          <li>
            <strong>Historial público.</strong> Todo cambio al repositorio queda registrado con fecha y autor.
            Un control automático marca en rojo, a la vista de todos, cualquier modificación de un informe ya
            publicado que no venga del proceso automático. Si un día se regenera la edición (por ejemplo, porque
            falló un proveedor), la versión anterior también queda en el historial.
          </li>
          <li>
            <strong>Código abierto.</strong> Todo el sistema está en{" "}
            <a href={REPO_URL}>{REPO_URL.replace("https://", "")}</a>, y el registro de cada edición indica el
            commit exacto del código que la generó y los modelos usados.
          </li>
        </ul>
      </section>

      <section>
        <h2>Medios consultados</h2>
        {summary ? (
          <>
            <p>
              Hoy se consultan <strong>{summary.active.length}</strong> medios de{" "}
              {new Set(summary.active.map((s) => s.country)).size} países
              {summary.inactive > 0 &&
                ` (otros ${summary.inactive} están en la lista, pero no tienen hoy un feed público utilizable y no se consultan)`}
              . En cada país se buscó incluir medios de línea oficialista y opositora, para contrastar encuadres.
              Además, cada medio está clasificado según quién lo controla (estatal, privado, partidario, sin fines
              de lucro o en el exilio): los medios estatales se incluyen para mostrar la posición oficial de su
              país y el informe los nombra como tales. Esta clasificación es revisable: si creés que un medio
              está mal clasificado, podés objetarla abriendo un issue en el repositorio.
              La etiqueta de cada medio es una clasificación editorial nuestra, discutible, y está publicada en{" "}
              <a href={repoFileURL("config/sources.yaml")}>config/sources.yaml</a>.
            </p>
            <div className={styles.tableWrap}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Región</th>
                    {summary.stances.map((s) => (
                      <th key={s}>{STANCE_LABELS[s] ?? s}</th>
                    ))}
                    <th>Total</th>
                    <th>Países</th>
                  </tr>
                </thead>
                <tbody>
                  {summary.sorted.map((row) => (
                    <tr key={row.region}>
                      <td>{REGION_LABELS[row.region] ?? row.region}</td>
                      {summary.stances.map((s) => (
                        <td key={s}>{row.byStance[s] ?? 0}</td>
                      ))}
                      <td>{row.total}</td>
                      <td className={styles.countries}>{[...row.countries].sort().join(", ")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <details className={styles.fullList}>
              <summary>Ver la lista completa</summary>
              <ul>
                {[...summary.active]
                  .sort((a, b) => a.country.localeCompare(b.country, "es") || a.name.localeCompare(b.name, "es"))
                  .map((s) => (
                    <li key={s.name}>
                      <a href={s.homepage} target="_blank" rel="noopener noreferrer">{s.name}</a> — {s.country},{" "}
                      {(STANCE_LABELS[s.stance] ?? s.stance).toLowerCase()}
                      {s.ownership && `, ${OWNERSHIP_LABELS[s.ownership] ?? s.ownership}`}
                      {s.ownership_note && ` (${s.ownership_note})`}
                    </li>
                  ))}
              </ul>
            </details>
          </>
        ) : (
          <p>
            La lista de medios no está disponible en este momento. Podés consultarla directamente en{" "}
            <a href={repoFileURL("config/sources.yaml")}>config/sources.yaml</a>.
          </p>
        )}
      </section>

      <section>
        <h2>Límites y sesgos conocidos</h2>
        <p>
          Que nadie edite los informes a mano no significa que no tengan sesgos. Preferimos decirlo de frente:
        </p>
        <ul>
          <li>
            <strong>La selección de medios es una decisión humana.</strong> Qué medios, de qué países y con qué
            etiqueta editorial se definió a mano, y eso determina qué noticias pueden aparecer.
          </li>
          <li>
            <strong>Los modelos de lenguaje se equivocan.</strong> Pueden clasificar mal un titular, resumir con
            imprecisiones o dar más peso a un tema que a otro. Por eso cada informe enlaza las notas originales:
            ante la duda, la fuente es la nota del medio.
          </li>
          <li>
            <strong>Solo titulares y bajadas.</strong> El sistema no lee las notas completas, solo lo que publica
            cada medio en su feed.
          </li>
          <li>
            <strong>No todos los medios tienen feed.</strong> Algunos bloquean el acceso automático; esos quedan
            afuera aunque estén en la lista.
          </li>
          <li>
            <strong>Manipulación por los propios titulares.</strong> Un medio podría publicar un titular armado
            para darle instrucciones al modelo. El sistema lo trata como texto a clasificar, no como orden, y no
            deja pasar enlaces ajenos, pero ninguna defensa es perfecta.
          </li>
        </ul>
      </section>

      <section>
        <h2>Financiamiento</h2>
        <p>
          Radar Global no tiene publicidad ni patrocinadores. Funciona sobre planes gratuitos de infraestructura y,
          opcionalmente, con donaciones voluntarias de lectores, que no dan ningún tipo de influencia sobre el
          contenido.
        </p>
      </section>

      <section>
        <h2>Errores y correcciones</h2>
        <p>
          Si encontrás un error en un informe, avisanos abriendo un issue en{" "}
          <a href={`${REPO_URL}/issues`}>el repositorio</a>. Los informes publicados no se corrigen en silencio:
          cualquier cambio queda en el historial público. Para reportar un problema de seguridad, seguí{" "}
          <a href={repoFileURL("SECURITY.md")}>la política de seguridad</a>.
        </p>
      </section>
    </main>
  );
}
