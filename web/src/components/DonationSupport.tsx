import Image from "next/image";
import { CAFECITO_URL, TECITO_URL } from "@/lib/donations";
import ExternalLink from "./ExternalLink";
import styles from "./DonationSupport.module.css";

// Sección de apoyo con links de donación (Cafecito para Argentina, Tecito
// para el exterior). Es server component: no tiene estado ni interacción
// propia, solo links externos.
export default function DonationSupport({
  variant = "masthead",
}: {
  variant?: "masthead" | "article";
}) {
  const variantClass = variant === "article" ? styles.article : styles.masthead;

  return (
    <section className={`${styles.support} ${variantClass}`} aria-label="Apoyá a Radar Global">
      <p className={styles.message}>
        Radar Global es un esfuerzo independiente. Si el informe te resulta
        útil, invitanos un café: cada aporte nos ayuda a seguir la actualidad
        internacional, día a día.
      </p>
      <div className={styles.buttons}>
        <ExternalLink href={CAFECITO_URL} className={styles.button}>
          <Image src="/donar-cafecito.png" alt="" width={180} height={180} className={styles.icon} />
          <span className={styles.buttonText}>
            <strong>Cafecito</strong>
            <span>Desde Argentina · desde $500 ARS</span>
          </span>
        </ExternalLink>
        <ExternalLink href={TECITO_URL} className={styles.button}>
          <Image src="/donar-tecito.png" alt="" width={256} height={256} className={styles.icon} />
          <span className={styles.buttonText}>
            <strong>Tecito</strong>
            <span>Desde el exterior · desde USD 1</span>
          </span>
        </ExternalLink>
      </div>
    </section>
  );
}
