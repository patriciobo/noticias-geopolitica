# Guía de despliegue en plataformas gratuitas

## Arquitectura de producción

```
GitHub Actions (cron diario)          Render (free)               Vercel (free)
  go run ./cmd/ingest        push     cmd/api sirve       fetch     web/ (Next.js)
  escribe reports/*.md    ───────►    reports/ del repo  ◄───────   NOTICIAS_API_URL
  y commitea al repo      redeploy    en :$PORT
```

Por qué así: el pipeline es un job diario (no un servidor) y la API lee
archivos de disco. Los discos de los planes gratis son efímeros, así que los
reportes viven **en el repo**: Actions los commitea y cada push redeploya la API
con la edición nueva incluida. Sin base de datos, sin cambios de código.

| Pieza | Plataforma | Costo |
|---|---|---|
| Job diario | GitHub Actions | gratis (repos públicos sin límite; privados ~2000 min/mes) |
| API Go | Render (Web Service) | gratis, se duerme tras ~15 min sin tráfico |
| Blog Next.js | Vercel (Hobby) | gratis, uso no comercial |
| LLM | Gemini free tier (aistudio.google.com) | gratis, con límites de uso |

Los límites de cada plan cambian: confirmalos en las páginas de precios antes
de depender de ellos.

## Requisitos

- Repo en GitHub (ver `git push -u origin main`).
- Key de Gemini: aistudio.google.com → *Get API key*.
- Cuentas en Render y Vercel, ambas con login de GitHub.

> Ollama no sirve en producción gratis (necesita una máquina propia) y la API
> de Claude es paga. El workflow usa `LLM_PROVIDER=gemini`; si querés Claude,
> cambiá el provider y el secret en `.github/workflows/daily.yml`.

## Paso 1 — Job diario (GitHub Actions)

El workflow ya está en `.github/workflows/daily.yml`.

1. Repo → *Settings → Secrets and variables → Actions → New repository secret*:
   `GEMINI_API_KEY` = tu key.
   Opcional: `OPENROUTER_API_KEY` = key de openrouter.ai/keys — fallback
   automático si Gemini falla (cuota agotada, caída), usa un modelo free.
   Sin este secret el pipeline sigue andando solo con Gemini, como antes.
2. Repo → *Settings → Actions → General → Workflow permissions* →
   **Read and write permissions** (para que pueda commitear).
3. Repo → *Actions → Informe diario → Run workflow* para probarlo a mano.
4. Verificá que aparezca un commit `chore: informe del AAAA-MM-DD` con
   `reports/AAAA-MM-DD.md` y `.sources.json`.

Corre todos los días a las 10:00 UTC (07:00 Argentina). Editá el `cron` para
cambiar la hora. Si el free tier de Gemini devuelve 429 seguido, bajá
`GEMINI_CLASSIFY_CONCURRENCY` (ya defaultea a 2).

Caveats:
- Algunos medios pueden bloquear IPs de datacenter (GitHub); esos feeds fallan
  y el pipeline sigue con el resto (ver el log del job).
- GitHub desactiva crons de repos sin actividad por 60 días; los commits
  diarios del propio job cuentan como actividad.

## Paso 2 — API (Render)

1. render.com → *New → Web Service* → conectá el repo.
2. Configuración:
   - **Runtime:** Go
   - **Build Command:** `cd core && go build -o api ./cmd/api`
   - **Start Command:** `API_ADDR=":$PORT" OUT_DIR=./reports ./core/api`
   - **Instance type:** Free
   - **Health Check Path:** `/health`
3. Deploy. Probá `https://<tu-servicio>.onrender.com/reports/latest`.
4. Copiá el **Deploy Hook** de esta pantalla (Settings → Deploy, ícono de
   copiar al lado del campo, es secreto) y agregalo en GitHub como el
   secret `RENDER_DEPLOY_HOOK_URL` (mismo lugar que `GEMINI_API_KEY`, paso 1).

El auto-deploy "On Commit" de Render (webhook de GitHub) puede dejar de
avisarle a Render sin ningún error visible — nos pasó, un reporte quedó
pusheado pero invisible en producción varios días. Por eso el workflow
dispara el deploy directo con el Deploy Hook al final de "Commitear reporte"
(paso 4 de arriba) en vez de depender solo de esa integración. Es opcional:
sin el secret, el pipeline sigue andando y queda a cargo del auto-deploy de
Render — pero si eso se rompe de nuevo, no hay forma de enterarse salvo
notando que el blog no actualiza.

`OUT_DIR=./reports` es relativo al directorio desde el que arranca el proceso
(raíz del repo en Render); si tu servicio usa otro *Root Directory*, ajustalo.

Cold start: tras 15 min sin tráfico el servicio se duerme y el primer request
tarda ~1 min. El home del blog hace un request por edición (hasta 31), así que
la primera visita en frío es lenta. Opcional: un ping cada 10 min a `/health`
desde un monitor gratis (UptimeRobot, cron-job.org) lo mantiene despierto.

## Paso 3 — Blog (Vercel)

1. vercel.com → *Add New → Project* → importá el repo.
2. **Root Directory:** `web` (Framework: Next.js, se detecta solo).
3. *Environment Variables:* `NOTICIAS_API_URL` = `https://<tu-servicio>.onrender.com`
   (sin `/` final).
4. Deploy.

El blog pide los datos en cada visita (`cache: "no-store"`), así que la edición
nueva aparece apenas Render termina de redeployar; no hace falta redeployar
Vercel.

## Verificación de punta a punta

1. Actions → *Run workflow* → esperá el commit del reporte.
2. Render muestra un deploy nuevo; `/reports` lista la fecha de hoy.
3. Abrí el blog: última edición arriba, anteriores plegadas, y "Noticias
   utilizadas" con links.

## Problemas comunes

| Síntoma | Causa probable |
|---|---|
| El job falla con `GEMINI_API_KEY no está configurada` | Falta el secret o está mal escrito. |
| El reporte se generó y commiteó pero no aparece en el blog | Render no redeployó. Revisá *Events* en Render — si el último deploy live es viejo, el webhook de GitHub dejó de avisarle; usá *Manual Deploy* para el reporte pendiente y configurá `RENDER_DEPLOY_HOOK_URL` (paso 2.4) para que no dependa más de ese webhook. |
| `git push` rechazado en el job | Permisos de workflow en *Read only* (paso 1.2) o rama protegida. |
| El blog dice "No se pudo conectar con la API" | `NOTICIAS_API_URL` mal seteada, o Render dormido (reintentá en 1 min). |
| El blog dice "Todavía no se generó ningún reporte" | `reports/` vacío en el repo o `OUT_DIR` mal configurado en Render. |
| Muchos `fetch <medio>` con error en el log | El medio bloquea IPs de GitHub; ver `config/sources_rss.yaml`. |

## Alternativas

- **Cloudflare Pages / Netlify** en lugar de Vercel para el blog: mismo esquema.
- **Koyeb / Railway** en lugar de Render para la API: los planes gratis varían;
  revisá que el disco no sea un requisito (acá no lo es).
- **Todo en un solo host** (una VM gratuita tipo Oracle Cloud Always Free) con
  cron + `cmd/api` + `next start`: más control, pero más mantenimiento.
