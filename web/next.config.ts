import type { NextConfig } from "next";
import path from "path";

const isDev = process.env.NODE_ENV === "development";

// CSP sin nonces (ver node_modules/next/dist/docs/01-app/02-guides/content-security-policy.md):
// los nonces obligan a renderizar cada página por request, y el sitio se
// sirve cacheado (ISR) justamente para aguantar picos de tráfico. Con
// 'unsafe-inline' en script-src lo que protege es el resto: sin HTML crudo
// en el markdown (react-markdown lo escapa), sin orígenes externos salvo
// Cloudflare Turnstile, y sin posibilidad de embeber el sitio en un iframe.
const csp = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline' https://challenges.cloudflare.com${isDev ? " 'unsafe-eval'" : ""}`,
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' blob: data:",
  "font-src 'self'",
  "connect-src 'self' https://challenges.cloudflare.com",
  "frame-src https://challenges.cloudflare.com",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
  // En dev el sitio corre en http://localhost: forzar https rompería los assets.
  ...(isDev ? [] : ["upgrade-insecure-requests"]),
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: csp },
  { key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), browsing-topics=()" },
];

const nextConfig: NextConfig = {
  turbopack: {
    root: path.join(__dirname),
  },
  poweredByHeader: false,
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default nextConfig;
