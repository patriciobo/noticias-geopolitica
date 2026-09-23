import { NextResponse } from "next/server";

// Corre server-side: recibe el POST del browser (mismo origen, sin CORS) y
// lo reenvía a core/cmd/api usando NOTICIAS_API_URL (var privada, server-only
// — Vercel no permite exponerla con NEXT_PUBLIC_ en este proyecto, y de
// paso evita tener que configurar CORS en el backend Go).
//
// Es la primera barrera contra bots: honeypot, captcha (Turnstile) y chequeo
// de origen acá; rate limit por IP, tope diario y doble opt-in en el backend.
const API_BASE_URL = process.env.NOTICIAS_API_URL ?? "http://localhost:8080";

// Secreto compartido con core/cmd/api (INTERNAL_API_SECRET en los dos
// lados). Con él seteado, Render rechaza altas que no vengan de acá.
const INTERNAL_API_SECRET = process.env.INTERNAL_API_SECRET ?? "";

// Opcional: sin la secret key de Turnstile no se exige captcha (útil en
// desarrollo local). En producción conviene tenerlo siempre.
const TURNSTILE_SECRET_KEY = process.env.TURNSTILE_SECRET_KEY ?? "";

function clientIP(request: Request): string {
  // En Vercel, x-forwarded-for lo pone la plataforma (pisa lo que mande el
  // cliente); la primera entrada es la IP real del navegador.
  return request.headers.get("x-forwarded-for")?.split(",")[0]?.trim() ?? "";
}

function sameOrigin(request: Request): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return true; // navegadores viejos no siempre lo mandan en same-origin
  try {
    return new URL(origin).host === request.headers.get("host");
  } catch {
    return false;
  }
}

async function verifyTurnstile(token: unknown, ip: string): Promise<boolean> {
  if (!TURNSTILE_SECRET_KEY) return true;
  if (typeof token !== "string" || token === "") return false;
  try {
    const res = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
      method: "POST",
      body: new URLSearchParams({ secret: TURNSTILE_SECRET_KEY, response: token, remoteip: ip }),
    });
    const data = (await res.json()) as { success?: boolean };
    return data.success === true;
  } catch {
    return false;
  }
}

export async function POST(request: Request) {
  if (!sameOrigin(request)) {
    return NextResponse.json({ error: "origen no permitido" }, { status: 403 });
  }

  let body: Record<string, unknown>;
  try {
    const parsed: unknown = await request.json();
    if (typeof parsed !== "object" || parsed === null) throw new Error();
    body = parsed as Record<string, unknown>;
  } catch {
    return NextResponse.json({ error: "body inválido" }, { status: 400 });
  }

  const email = body.email;
  if (typeof email !== "string" || email.length > 254) {
    return NextResponse.json({ error: "email inválido" }, { status: 400 });
  }

  // Honeypot: un humano no ve el campo "website" y lo deja vacío. A un bot
  // que lo completa se le contesta éxito, sin hacer nada, para que no
  // aprenda a esquivarlo.
  if (typeof body.website === "string" && body.website !== "") {
    return NextResponse.json({ status: "ok" });
  }

  const ip = clientIP(request);
  if (!(await verifyTurnstile(body.turnstileToken, ip))) {
    return NextResponse.json({ error: "verificación anti-bot fallida" }, { status: 403 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${API_BASE_URL}/subscribers`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Internal-Secret": INTERNAL_API_SECRET,
        "X-Client-IP": ip,
      },
      body: JSON.stringify({ email }),
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "no se pudo conectar con la API" }, { status: 502 });
  }

  if (!upstream.ok) {
    // 400/429/503 tienen un mensaje específico para mostrarle al usuario.
    // Un 403 de Go significa INTERNAL_API_SECRET distinto entre Vercel y
    // Render (error de config nuestro, no del usuario): se reporta como 502.
    const status = [400, 429, 503].includes(upstream.status) ? upstream.status : 502;
    if (upstream.status === 403) {
      console.error("POST /subscribers devolvió 403: INTERNAL_API_SECRET no coincide entre Vercel y Render");
    }
    return NextResponse.json({ error: "no se pudo completar la suscripción" }, { status });
  }
  return NextResponse.json({ status: "ok" });
}
