import styles from "./SiteHeader.module.css";

// Header global (layout.tsx, aparece en todas las páginas). Los links de
// nav son anclas simples a secciones que ya existen en el home — sin
// funcionalidad nueva, sin JS: navegación nativa del browser.
export default function SiteHeader() {
  return (
    <header className={styles.header}>
      <div className={styles.bar}>
        <a href="/" className={styles.brand}>
          <span aria-hidden="true">📰</span> Noticias Internacionales
        </a>
        <nav className={styles.nav}>
          <a href="/#ediciones-anteriores">Ediciones</a>
          <a href="/#resumen-por-region">Regiones</a>
          <a href="/#fuentes-consultadas">Fuentes</a>
        </nav>
      </div>
      <div className={styles.accentBar} aria-hidden="true" />
    </header>
  );
}
