import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import styles from "./Abstract.module.css";

/** Copete del día, siempre visible aunque el informe completo esté colapsado. */
export default function Abstract({ markdown }: { markdown: string }) {
  return (
    <div className={styles.box}>
      <p className={styles.label}>📝 Resumen</p>
      <div className={styles.content}>
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{markdown}</ReactMarkdown>
      </div>
    </div>
  );
}
