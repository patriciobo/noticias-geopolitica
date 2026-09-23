import Image from "next/image";
import { CAFECITO_URL, TECITO_URL } from "@/lib/donations";
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
    <section className={`${styles.support} ${variantClass}`}>
      <p className={styles.message}>
        Radar Global es un esfuerzo independiente. Si el informe te resulta
        útil, invitanos un café: cada aporte nos ayuda a seguir la actualidad
        internacional, día a día.
      </p>
      <div className={styles.buttons}>
        <a href={CAFECITO_URL} target="_blank" rel="noreferrer" className={styles.button}>
          <Image src="/donar-cafecito.png" alt="" width={180} height={180} className={styles.icon} />
          <span className={styles.buttonText}>
            <strong>Cafecito</strong>
            <span>Desde Argentina · desde $500 ARS</span>
          </span>
        </a>
        <a href={TECITO_URL} target="_blank" rel="noreferrer" className={styles.button}>
          <Image src="/donar-tecito.png" alt="" width={256} height={256} className={styles.icon} />
          <span className={styles.buttonText}>
            <strong>Tecito</strong>
            <span>Desde el exterior · desde USD 1</span>
          </span>
        </a>
      </div>
    </section>
  );
}
