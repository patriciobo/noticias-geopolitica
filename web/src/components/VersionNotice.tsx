import Link from "next/link";
import type { Provenance } from "@/lib/api";
import styles from "./AutomationNotice.module.css";

// Una edición regenerada lo dice arriba de todo, con el motivo: nunca se
// reemplaza una edición en silencio.
export default function VersionNotice({ provenance }: { provenance?: Provenance }) {
  const revisions = provenance?.revisions ?? [];
  if (!provenance?.version || provenance.version < 2 || revisions.length === 0) {
    return null;
  }
  const last = revisions[revisions.length - 1];
  const when = new Date(last.generated_at).toLocaleString("es-AR", {
    timeZone: "America/Argentina/Buenos_Aires",
    dateStyle: "short",
    timeStyle: "short",
  });
  return (
    <aside className={styles.notice} aria-label="Versión de la edición">
      <strong>Versión {provenance.version} de esta edición</strong> (actualizada el {when}, hora argentina).
      Motivo: {last.reason ?? "sin motivo declarado"}. <Link href="/correcciones">Ver todas las correcciones</Link>
    </aside>
  );
}
