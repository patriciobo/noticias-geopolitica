import { flagFor } from "@/lib/countryFlags";
import { hostnameOf, type NewsLinkGroup } from "@/lib/newsLinks";
import { REGION_THEME } from "@/lib/regionTheme";
import styles from "./NewsLinks.module.css";

export default function NewsLinks({ groups }: { groups: NewsLinkGroup[] }) {
  const total = groups.reduce((n, g) => n + g.links.length, 0);

  return (
    <section className={styles.section} aria-labelledby="noticias-utilizadas">
      <h2 id="noticias-utilizadas" className={styles.heading}>
        🔗 Noticias utilizadas
      </h2>
      <p className={styles.intro}>
        {total} notas originales en que se basa este informe. Cada título lleva
        al artículo en el sitio del medio (se abre en una pestaña nueva).
      </p>
      {groups.map((group) => (
        <div key={group.region} className={styles.group}>
          <h3 className={styles.region}>
            {REGION_THEME[group.region]?.emoji} {group.region}
            <span className={styles.count}>{group.links.length}</span>
          </h3>
          <ul className={styles.list}>
            {group.links.map((link) => (
              <li key={link.url} className={styles.item}>
                <a
                  className={styles.title}
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {link.title}
                  <span aria-hidden="true" className={styles.arrow}>
                    {" "}
                    ↗
                  </span>
                </a>
                <span className={styles.meta}>
                  {flagFor(link.country)} {link.outlet} · {hostnameOf(link.url)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}
