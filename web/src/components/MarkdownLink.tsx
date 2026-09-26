import type { ReactNode } from "react";
import ExternalLink, { NEW_TAB_TEXT } from "./ExternalLink";
import styles from "./MarkdownLink.module.css";

function text(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(text).join("");
  return "";
}

const CITATION_RE = /^\[(\d+)\]$/;

/**
 * Link de un informe renderizado desde markdown. Las citas ("[3]", que el
 * backend convierte en link a la nota 3) se muestran como un número chico
 * con el título de la nota como destino, para que cada afirmación se pueda
 * rastrear sin ensuciar la lectura.
 */
export default function MarkdownLink({ href, children }: { href?: string; children?: ReactNode }) {
  const m = CITATION_RE.exec(text(children).trim());
  if (m) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className={styles.citation} aria-label={`Fuente ${m[1]} (${NEW_TAB_TEXT})`}>
        {m[1]}
      </a>
    );
  }
  return <ExternalLink href={href}>{children}</ExternalLink>;
}
