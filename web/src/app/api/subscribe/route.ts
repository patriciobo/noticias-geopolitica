import { NextResponse } from "next/server";

// Corre server-side: recibe el POST del browser (mismo origen, sin CORS) y
// lo reenvía a core/cmd/api usando NOTICIAS_API_URL (var privada, server-only
// — Vercel no permite exponerla con NEXT_PUBLIC_ en este proyecto, y de
// paso evita tener que configurar CORS en el backend Go).
const API_BASE_URL = process.env.NOTICIAS_API_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "body inválido" }, { status: 400 });
  }

  const email = typeof body === "object" && body !== null && "email" in body ? (body as { email: unknown }).email : undefined;
  if (typeof email !== "string") {
    return NextResponse.json({ error: "email inválido" }, { status: 400 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${API_BASE_URL}/subscribers`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });
  } catch {
    return NextResponse.json({ error: "no se pudo conectar con la API" }, { status: 502 });
  }

  if (!upstream.ok) {
    return NextResponse.json({ error: "no se pudo completar la suscripción" }, { status: upstream.status });
  }
  return NextResponse.json({ status: "ok" });
}
