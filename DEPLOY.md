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
| DB de suscriptores | Neon (Postgres free tier) | gratis, sostenido (no expira como el Postgres de Render) |
| Envío de newsletter | Brevo (transactional) | gratis, 300 emails/día para siempre |

Los límites de cada plan cambian: confirmalos en las páginas de precios antes
de depender de ellos.

## Requisitos

- Repo en GitHub (ver `git push -u origin main`).
- Key de Gemini: aistudio.google.com → *Get API key*.
- Cuentas en Render y Vercel, ambas con login de GitHub.
- Opcional (solo si querés el newsletter): cuenta en neon.tech y en brevo.com.

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

## Paso 4 — Newsletter (Neon + Brevo, opcional)

Sin esto, todo lo de arriba sigue funcionando igual — el newsletter es
enteramente opcional. Si querés habilitarlo:

1. **Neon**: neon.tech → *New Project* → copiá el *Connection string*
   (`postgres://usuario:password@host/dbname?sslmode=require`). No hace
   falta crear tablas a mano: `cmd/api`/`cmd/newsletter` las crean solas al
   arrancar (`CREATE TABLE IF NOT EXISTS`, ver `core/internal/store`).
2. **Brevo**: brevo.com → creá cuenta gratis (sin tarjeta) → verificá un
   email/dominio remitente (*Senders, Domains & Dedicated IPs*) → *SMTP &
   API → API Keys → Generate a new API key*.
3. **Render** (donde corre `cmd/api`) → *Environment* → agregá:
   - `DATABASE_URL` = el connection string de Neon.
   - `WEB_ORIGIN` = la URL de tu blog en Vercel (para el CORS del form de
     suscripción — sin esto, o dejándolo en `*`, el form igual funciona pero
     sin restringir el origen).
4. **GitHub** (Settings → Secrets and variables → Actions) → agregá, además
   de los secrets del paso 1:
   - `DATABASE_URL` = el mismo connection string de Neon (sí, otra vez —
     es un secret distinto, en un lugar distinto; ver más abajo por qué).
   - `BREVO_API_KEY` = la key generada en el paso 2.
   - `BREVO_SENDER_EMAIL` = el email verificado en Brevo.
   - *Settings → Secrets and variables → Actions → Variables* (no
     *Secrets* — no es sensible) → `API_BASE_URL` = la URL de tu servicio en
     Render (para armar el link de baja en el email).
5. En Vercel (paso 3 de arriba) agregá también `NEXT_PUBLIC_NOTICIAS_API_URL`
   = la misma URL de Render que `NOTICIAS_API_URL` — el form de suscripción
   corre en el browser, y Next.js solo expone al cliente las variables con
   prefijo `NEXT_PUBLIC_`.

**`DATABASE_URL` va en dos lugares distintos y hay que cargarla en los dos**:
Render (para que `cmd/api` sirva `POST /subscribers`) y GitHub Actions (para
que `cmd/newsletter` corra en el cron diario). Son procesos separados que no
comparten variables de entorno entre sí — cargar solo uno dejaría la otra
mitad del feature rota en silencio (el form suscribe gente bien, pero nunca
les llega el correo, o viceversa).

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
| El form de suscripción no hace nada / error de CORS en la consola del browser | Falta `NEXT_PUBLIC_NOTICIAS_API_URL` en Vercel, o `WEB_ORIGIN` en Render no incluye el origen del blog. |
| La gente se suscribe bien pero nunca recibe el correo | `DATABASE_URL` está seteada en Render pero no en GitHub Actions (o al revés) — tiene que estar en los dos, ver paso 4. |
| `cmd/newsletter` loguea "no configurados, no se envía" | Falta `DATABASE_URL` o `BREVO_API_KEY` como secret de GitHub Actions. |

## Alternativas

- **Cloudflare Pages / Netlify** en lugar de Vercel para el blog: mismo esquema.
- **Koyeb / Railway** en lugar de Render para la API: los planes gratis varían;
  revisá que el disco no sea un requisito (acá no lo es).
- **Todo en un solo host** (una VM gratuita tipo Oracle Cloud Always Free) con
  cron + `cmd/api` + `next start`: más control, pero más mantenimiento.
