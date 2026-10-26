# Imagen del blog para Coolify (ver DEPLOY.md, "Paso 3 — Blog").
# El contexto de build es web/, no la raíz del repo: el blog no necesita
# nada de afuera (los reportes los trae del API por HTTP).
#
# Usa la salida standalone de Next (output: "standalone" en
# next.config.ts): server.js + solo los node_modules que trazó el build.

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# El build prerenderiza las páginas con ISR (home, /correcciones,
# /metodologia) pegándole al API. Sin NOTICIAS_API_URL en el build igual
# compila, pero quedan horneadas con el mensaje de "no se pudo conectar"
# hasta la primera revalidación (5 minutos después de la primera visita).
# En Coolify: marcar las dos como disponibles en build.
ARG NOTICIAS_API_URL
ARG TURNSTILE_SITE_KEY
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
RUN adduser -D -H -u 10001 app
COPY --from=build /app/public ./public
COPY --from=build --chown=app:app /app/.next/standalone ./
COPY --from=build --chown=app:app /app/.next/static ./.next/static
USER app
EXPOSE 3000
# La cache de ISR vive en .next/ dentro del contenedor: se pierde con cada
# redeploy y se rearma sola con las primeras visitas.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:3000/api/health || exit 1
CMD ["node", "server.js"]
