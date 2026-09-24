import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import MarkdownLink from "./MarkdownLink";
import styles from "./Abstract.module.css";

/** Copete del día, siempre visible aunque el informe completo esté colapsado. */
export default function Abstract({ markdown }: { markdown: string }) {
  return (
    <div className={styles.box}>
      <p className={styles.label}>📝 Resumen</p>
      <div className={styles.content}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={{ a: ({ href, children }) => <MarkdownLink href={href}>{children}</MarkdownLink> }}>
          {markdown}
        </ReactMarkdown>
      </div>
    </div>
  );
}
