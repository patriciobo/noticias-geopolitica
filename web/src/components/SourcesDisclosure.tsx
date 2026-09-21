import type { SourceSummary } from "@/lib/api";
import { flagFor } from "@/lib/countryFlags";
import styles from "./SourcesDisclosure.module.css";

export default function SourcesDisclosure({ sources }: { sources: SourceSummary[] }) {
  if (sources.length === 0) return null;

  const byCountry = new Map<string, string[]>();
  for (const s of sources) {
    const list = byCountry.get(s.country) ?? [];
    list.push(s.name);
    byCountry.set(s.country, list);
  }
  const countries = [...byCountry.keys()].sort();

  return (
    <details className={styles.disclosure}>
      <summary className={styles.summary}>
        {sources.length} medios consultados
      </summary>
      <div className={styles.grid}>
        {countries.map((country) => (
          <div key={country} className={styles.countryBlock}>
            <p className={styles.countryName}>
              {flagFor(country)} {country}
            </p>
            <ul className={styles.mediaList}>
              {byCountry.get(country)!.map((name) => (
                <li key={name}>{name}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </details>
  );
}
