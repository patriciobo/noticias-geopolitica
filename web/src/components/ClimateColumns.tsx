import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import MarkdownLink from "./MarkdownLink";
import { splitSubsections } from "@/lib/reportSections";
import styles from "./ClimateColumns.module.css";

const components: Components = {
  a({ href, children }) {
    return <MarkdownLink href={href}>{children}</MarkdownLink>;
  },
};

/**
 * "Clima internacional: comercio, industria y materias primas" en columnas,
 * una por subsección ("### Comercio", "### Industria", ...). La cantidad de
 * columnas depende de cuántas subsecciones haya ese día — auto-fit, no un
 * grid fijo de 3, para no romper si el LLM escribe más o menos.
 */
export default function ClimateColumns({ body }: { body: string }) {
  const columns = splitSubsections(body);
  if (columns.length === 0) {
    return (
      <div className={styles.markdown}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {body}
        </ReactMarkdown>
      </div>
    );
  }

  return (
    <div className={styles.grid}>
      {columns.map((col) => (
        <div key={col.heading} className={styles.column}>
          <h3 className={styles.heading}>{col.heading}</h3>
          <div className={styles.markdown}>
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
              {col.body}
            </ReactMarkdown>
          </div>
        </div>
      ))}
    </div>
  );
}
