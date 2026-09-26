import Link from "next/link";
import { reportErrorURL } from "@/lib/transparency";
import ExternalLink from "./ExternalLink";
import styles from "./AutomationNotice.module.css";

// Aviso fijo en cada edición: qué es Radar Global y qué no es. Un resumen
// automatizado de titulares no verifica hechos — decirlo en cada edición,
// no solo en /metodologia, evita que el formato de "informe" sugiera un
// trabajo editorial que no existe.
export default function AutomationNotice({ date }: { date: string }) {
  return (
    <div className={styles.notice} role="note" aria-label="Cómo leer esta edición">
      <strong>Resumen automatizado de titulares.</strong> Un modelo de lenguaje selecciona y resume lo que
      publicaron los medios listados abajo. No verifica los hechos: atribuye cada dato al medio que lo
      publicó. <Link href="/metodologia">Cómo se hace</Link> ·{" "}
      <ExternalLink href={reportErrorURL(date)}>¿Encontraste un error?</ExternalLink>
    </div>
  );
}
