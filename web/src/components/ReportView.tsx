import type { ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import Abstract from "./Abstract";
import ClimateColumns from "./ClimateColumns";
import CompaniesTable from "./CompaniesTable";
import NewsLinks from "./NewsLinks";
import SourcesDisclosure from "./SourcesDisclosure";
import { splitAbstract, splitNewsLinks } from "@/lib/newsLinks";
import { splitTopSections } from "@/lib/reportSections";
import type { SourceSummary } from "@/lib/api";
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
    if (textContent(children).trim() === NO_NEWS_TEXT) {
      return <p className={styles.noNews}>{children}</p>;
    }
    return <p>{children}</p>;
  },
  a({ href, children }) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    );
  },
};

export function ReportBody({
  markdown,
  sources,
}: {
  markdown: string;
  sources: SourceSummary[];
}) {
  const { body, groups } = splitNewsLinks(markdown);
  const sections = splitTopSections(body);

  return (
    <>
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
    </>
  );
}

export default function ReportView({
  date,
  markdown,
  sources,
}: {
  date: string;
  markdown: string;
  sources: SourceSummary[];
}) {
  const { abstract, body } = splitAbstract(markdown);
  return (
    <article className={styles.article}>
      <p className={styles.date}>Edición del {formatDate(date)}</p>
      {abstract && <Abstract markdown={abstract} />}
      <ReportBody markdown={body} sources={sources} />
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
