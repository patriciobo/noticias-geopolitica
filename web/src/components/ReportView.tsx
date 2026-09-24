import type { ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import MarkdownLink from "./MarkdownLink";
import Abstract from "./Abstract";
import AutomationNotice from "./AutomationNotice";
import VersionNotice from "./VersionNotice";
import ClimateColumns from "./ClimateColumns";
import CompaniesTable from "./CompaniesTable";
import NewsLinks from "./NewsLinks";
import ProvenanceDisclosure from "./ProvenanceDisclosure";
import SourcesDisclosure from "./SourcesDisclosure";
import { splitAbstract, splitNewsLinks } from "@/lib/newsLinks";
import { splitTopSections } from "@/lib/reportSections";
import type { Provenance, SourceSummary } from "@/lib/api";
import styles from "./ReportView.module.css";

function textContent(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(textContent).join("");
  return "";
}

const SECTION_ICONS: Record<string, string> = {
  "Resumen por región": "📰",
  "Clima internacional: comercio, industria y materias primas": "🌐",
  "Empresas potencialmente afectadas por región": "🏢",
};

// id para que el nav del header ("Regiones") pueda anclar directo a esta
// sección — mismo contenido de siempre, solo un atributo de más en el h2.
const SECTION_IDS: Record<string, string> = {
  "Resumen por región": "resumen-por-region",
};

const NO_NEWS_TEXT = "Sin novedades relevantes hoy.";

const genericMarkdownComponents: Components = {
  h3({ children }) {
    const text = textContent(children).trim();
    if (text === "Cobertura cruzada") {
      return <h3 className={styles.crossCoverage}>🌐 {children}</h3>;
    }
    return <h3>{children}</h3>;
  },
  p({ children }) {
    const text = textContent(children).trim();
    if (text === NO_NEWS_TEXT || text.startsWith("Cobertura insuficiente hoy")) {
      return <p className={styles.noNews}>{children}</p>;
    }
    return <p>{children}</p>;
  },
  a({ href, children }) {
    return <MarkdownLink href={href}>{children}</MarkdownLink>;
  },
};

export function ReportBody({
  date,
  markdown,
  sources,
  provenance,
}: {
  date: string;
  markdown: string;
  sources: SourceSummary[];
  provenance?: Provenance;
}) {
  const { body, groups } = splitNewsLinks(markdown);
  const sections = splitTopSections(body);

  return (
    <>
      <VersionNotice provenance={provenance} />
      <AutomationNotice date={date} />
      <div className={styles.markdown}>
        {sections.map((section) => {
          const icon = SECTION_ICONS[section.heading];
          const id = SECTION_IDS[section.heading];
          return (
            <section key={section.heading}>
              <h2 id={id}>
                {icon ? `${icon} ` : ""}
                {section.heading}
              </h2>
              {section.heading === "Clima internacional: comercio, industria y materias primas" ? (
                <ClimateColumns body={section.body} />
              ) : section.heading === "Empresas potencialmente afectadas por región" ? (
                <CompaniesTable body={section.body} />
              ) : (
                <ReactMarkdown remarkPlugins={[remarkGfm]} components={genericMarkdownComponents}>
                  {section.body}
                </ReactMarkdown>
              )}
            </section>
          );
        })}
      </div>
      {groups.length > 0 && <NewsLinks groups={groups} />}
      <div id="fuentes-consultadas">
        <SourcesDisclosure sources={sources} />
      </div>
      <ProvenanceDisclosure date={date} provenance={provenance} />
    </>
  );
}

export default function ReportView({
  date,
  markdown,
  sources,
  provenance,
}: {
  date: string;
  markdown: string;
  sources: SourceSummary[];
  provenance?: Provenance;
}) {
  const { abstract, body } = splitAbstract(markdown);
  return (
    <article className={styles.article}>
      <p className={styles.date}>Edición del {formatDate(date)}</p>
      {abstract && <Abstract markdown={abstract} />}
      <ReportBody date={date} markdown={body} sources={sources} provenance={provenance} />
    </article>
  );
}

export function formatDate(isoDate: string): string {
  const parsed = new Date(`${isoDate}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return isoDate;
  }
  return parsed.toLocaleDateString("es-AR", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}
