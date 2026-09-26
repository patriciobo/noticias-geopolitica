"use client";

import Script from "next/script";
import { useId, useRef, useState, type FormEvent } from "react";
import { subscribeEmail } from "@/lib/api";
import styles from "./SubscribeForm.module.css";

type Status = "idle" | "loading" | "success" | "error";

// API mínima del script de Cloudflare Turnstile que usamos (render explícito).
type Turnstile = {
  render: (
    el: HTMLElement,
    opts: {
      sitekey: string;
      callback: (token: string) => void;
      "expired-callback": () => void;
      "error-callback": () => void;
      language?: string;
    },
  ) => string;
  reset: (widgetId: string) => void;
};

declare global {
  interface Window {
    turnstile?: Turnstile;
  }
}

// turnstileSiteKey llega desde el Server Component (process.env del lado
// del servidor): la site key es pública, pero así no hace falta una
// variable NEXT_PUBLIC_*. Sin site key, el form funciona sin captcha.
export default function SubscribeForm({ turnstileSiteKey }: { turnstileSiteKey?: string }) {
  const [email, setEmail] = useState("");
  const [website, setWebsite] = useState(""); // honeypot
  const [status, setStatus] = useState<Status>("idle");
  const [errorMessage, setErrorMessage] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");
  const widgetRef = useRef<HTMLDivElement>(null);
  const widgetId = useRef<string | null>(null);
  const inputId = useId();
  const errorId = useId();

  function renderTurnstile() {
    if (!turnstileSiteKey || !widgetRef.current || !window.turnstile || widgetId.current) return;
    widgetId.current = window.turnstile.render(widgetRef.current, {
      sitekey: turnstileSiteKey,
      language: "es",
      callback: setTurnstileToken,
      "expired-callback": () => setTurnstileToken(""),
      "error-callback": () => setTurnstileToken(""),
    });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (turnstileSiteKey && !turnstileToken) {
      setStatus("error");
      setErrorMessage("Esperá a que termine la verificación anti-bot y probá de nuevo.");
      return;
    }
    setStatus("loading");
    const result = await subscribeEmail({ email, website, turnstileToken });
    if (result.ok) {
      setStatus("success");
      return;
    }
    setStatus("error");
    setErrorMessage(result.message);
    // Cada token de Turnstile sirve una sola vez: pedir uno nuevo.
    if (widgetId.current && window.turnstile) {
      window.turnstile.reset(widgetId.current);
      setTurnstileToken("");
    }
  }

  // Las regiones vivas (status/alert) existen desde el primer render: si se
  // montaran recién con el mensaje, los lectores de pantalla no lo anuncian.
  return (
    <div>
      <div role="status">
        {status === "success" && (
          <p className={styles.success}>
            Listo. Te mandamos un correo para confirmar la suscripción: hacé click en el enlace y
            empezás a recibir el resumen. Si no lo ves, revisá la carpeta de spam.
          </p>
        )}
      </div>
      {status !== "success" && (
        <form className={styles.form} onSubmit={handleSubmit} aria-busy={status === "loading"}>
          <label htmlFor={inputId} className={styles.label}>
            Recibí el informe diario por email
          </label>
          <input
            id={inputId}
            type="email"
            name="email"
            required
            maxLength={254}
            autoComplete="email"
            inputMode="email"
            placeholder="tu@email.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={status === "loading"}
            className={styles.input}
            aria-invalid={status === "error" ? true : undefined}
            aria-describedby={status === "error" ? errorId : undefined}
          />
          {/* Honeypot: invisible para humanos y lectores de pantalla; los bots
              que completan todos los campos lo llenan y se descartan. */}
          <input
            type="text"
            name="website"
            tabIndex={-1}
            autoComplete="off"
            aria-hidden="true"
            value={website}
            onChange={(e) => setWebsite(e.target.value)}
            className={styles.honeypot}
          />
          <button type="submit" disabled={status === "loading"} className={styles.button}>
            {status === "loading" ? "Enviando..." : "Recibir por email"}
          </button>
          {turnstileSiteKey && (
            <>
              <Script
                src="https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"
                strategy="afterInteractive"
                onReady={renderTurnstile}
              />
              <div ref={widgetRef} className={styles.captcha} />
            </>
          )}
          <div role="alert" id={errorId} className={styles.error}>
            {status === "error" ? errorMessage : ""}
          </div>
        </form>
      )}
    </div>
  );
}
