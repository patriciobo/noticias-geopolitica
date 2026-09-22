import type { ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import Abstract from "./Abstract";
import RegionBanner from "./RegionBanner";
import NewsLinks from "./NewsLinks";
import SourcesDisclosure from "./SourcesDisclosure";
import { splitAbstract, splitNewsLinks } from "@/lib/newsLinks";
import { REGION_THEME } from "@/lib/regionTheme";
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

const markdownComponents: Components = {
  h2({ children }) {
    const text = textContent(children).trim();
    const icon = SECTION_ICONS[text];
    return (
      <h2>
        {icon ? `${icon} ` : ""}
        {children}
      </h2>
    );
  },
  h3({ children }) {
    const text = textContent(children).trim();
    if (text in REGION_THEME) {
      return <RegionBanner region={text} />;
    }
    if (text === "Cobertura cruzada") {
      return <h3 className={styles.crossCoverage}>🌐 {children}</h3>;
    }
    return <h3>{children}</h3>;
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
  return (
    <>
      <SourcesDisclosure sources={sources} />
      <div className={styles.markdown}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
          {body}
        </ReactMarkdown>
      </div>
      {groups.length > 0 && <NewsLinks groups={groups} />}
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
