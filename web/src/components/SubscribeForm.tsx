"use client";

import { useState, type FormEvent } from "react";
import { subscribeEmail } from "@/lib/api";
import styles from "./SubscribeForm.module.css";

type Status = "idle" | "loading" | "success" | "error";

export default function SubscribeForm() {
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<Status>("idle");
  const [errorMessage, setErrorMessage] = useState("");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setStatus("loading");
    const result = await subscribeEmail(email);
    if (result.ok) {
      setStatus("success");
    } else {
      setStatus("error");
      setErrorMessage(result.message);
    }
  }

  if (status === "success") {
    return <p className={styles.success}>Listo, vas a recibir el resumen por email.</p>;
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <input
        type="email"
        required
        placeholder="tu@email.com"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        disabled={status === "loading"}
        className={styles.input}
        aria-label="Email para suscribirte al newsletter"
      />
      <button type="submit" disabled={status === "loading"} className={styles.button}>
        {status === "loading" ? "Enviando..." : "Recibir por email"}
      </button>
      {status === "error" && <p className={styles.error}>{errorMessage}</p>}
    </form>
  );
}
