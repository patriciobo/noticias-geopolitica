import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { splitSubsections } from "@/lib/reportSections";
import styles from "./CompaniesTable.module.css";

const components: Components = {
  a({ href, children }) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    );
  },
};

/**
 * "Empresas potencialmente afectadas por región" como filas: una por región
 * ("### <región>"), con la región como tag y el resto del contenido (bullets
 * empresa: motivo) al lado. Mismo dato que el markdown genérico, solo
 * reorganizado visualmente — no se separan sector/empresa/motivo en columnas
 * propias porque esa distinción no viene estructurada en el texto.
 */
export default function CompaniesTable({ body }: { body: string }) {
  const rows = splitSubsections(body);
  if (rows.length === 0) {
    return (
      <div className={styles.markdown}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {body}
        </ReactMarkdown>
      </div>
    );
  }

  return (
    <div className={styles.table}>
      {rows.map((row) => (
        <div key={row.heading} className={styles.row}>
          <span className={styles.tag}>{row.heading}</span>
          <div className={styles.markdown}>
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
              {row.body}
            </ReactMarkdown>
          </div>
        </div>
      ))}
    </div>
  );
}
