# Guía de despliegue

## Arquitectura de producción

```
GitHub Actions (cron diario)          Coolify (VPS)               Vercel (free)
  go run ./cmd/ingest      webhook    cmd/api sirve       fetch     web/ (Next.js)
  escribe reports/*.md    ───────►    reports/ de la     ◄───────   NOTICIAS_API_URL
  y commitea al repo      redeploy    imagen en :8080
```

Por qué así: el pipeline es un job diario (no un servidor) y la API lee
archivos de disco. Los reportes viven **en el repo**: Actions los commitea y
dispara un redeploy del contenedor, que se reconstruye con la edición nueva
adentro (`core/Dockerfile`). Sin base de datos ni volúmenes.

| Pieza | Plataforma | Costo |
|---|---|---|
| Job diario | GitHub Actions | gratis (repos públicos sin límite; privados ~2000 min/mes) |
| API Go | Coolify en VPS propio (contenedor de `core/Dockerfile`) | lo que cueste el VPS |
| Blog Next.js | Vercel (Hobby) | gratis, uso no comercial |
| LLM | Gemini free tier (aistudio.google.com) | gratis, con límites de uso |
| DB de suscriptores | Neon (Postgres free tier) | gratis, sostenido |
| Envío de newsletter | Brevo (transactional) | gratis, 300 emails/día para siempre |

Los límites de cada plan cambian: confirmalos en las páginas de precios antes
de depender de ellos.

## Requisitos

- Repo en GitHub (ver `git push -u origin main`).
- Key de Gemini: aistudio.google.com → *Get API key*.
- Un VPS con Coolify instalado y un dominio (o subdominio) para el API
  apuntando a ese VPS.
- Cuenta en Vercel con login de GitHub.
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

El cron de GitHub intenta cada 20 minutos entre 08:07 y 09:47 UTC;
el primer intento genera el informe y el resto se saltea. Ese cron es "best
effort" (GitHub lo atrasa horas o lo saltea), así que para garantizar la
hora conviene sumar el disparo externo de abajo. Si el free tier de Gemini
devuelve 429 seguido, bajá `GEMINI_CLASSIFY_CONCURRENCY` (ya defaultea a 2).

### Disparo externo puntual (cron-job.org)

cron-job.org (gratis) sí respeta la hora: llama a la API de GitHub para
lanzar el workflow con `solo_si_falta=true`, así que si el informe ya
existe la corrida termina en segundos sin duplicar nada.

1. **Token de GitHub** — *Settings (de tu cuenta) → Developer settings →
   Personal access tokens → Fine-grained tokens → Generate new token*:
   - *Repository access*: **Only select repositories** → `noticias-geopolitica`.
   - *Permissions → Repository permissions → Actions*: **Read and write**.
     Nada más (Metadata: Read queda puesto solo).
   - *Expiration*: la más larga que permita; anotá la fecha para renovarlo.
   Este token solo puede lanzar/cancelar workflows de este repo: no lee ni
   escribe código ni secrets. Igual tratálo como secreto: pegalo solo en
   cron-job.org.
2. **cron-job.org** → *Create cronjob*:
   - *URL*:
     `https://api.github.com/repos/patriciobo/noticias-geopolitica/actions/workflows/daily.yml/dispatches`
   - *Schedule*: custom, zona horaria **UTC**, expresión `0,30 8-9 * * *`
     (cuatro intentos: 08:00, 08:30, 09:00 y 09:30 UTC; la edición sale
     con el primero).
   - *Advanced → Request method*: **POST**.
   - *Advanced → Headers*:
     - `Authorization: Bearer <el token>`
     - `Accept: application/vnd.github+json`
     - `X-GitHub-Api-Version: 2022-11-28`
     - `Content-Type: application/json`
   - *Advanced → Request body*:
     `{"ref":"main","inputs":{"solo_si_falta":"true"}}`
   - *Notifications*: activá "on failure" para enterarte si el token vence.
3. *Test run* en cron-job.org: tiene que responder **204** y aparecer una
   corrida nueva en *Actions → Informe diario* (si el informe de hoy ya
   existe, termina en el job `check`).

Si responde 401 el token venció o está mal copiado; 404, el token no tiene
acceso al repo o falta el permiso *Actions*.

Caveats:
- Algunos medios pueden bloquear IPs de datacenter (GitHub); esos feeds fallan
  y el pipeline sigue con el resto (ver el log del job).
- GitHub desactiva crons de repos sin actividad por 60 días; los commits
  diarios del propio job cuentan como actividad.

## Paso 2 — API (Coolify)

El API corre en el contenedor de `core/Dockerfile`. La imagen lleva
`reports/` y `config/sources.yaml` adentro, así que cada edición nueva
llega con un redeploy; no hace falta volumen.

1. Coolify → *Projects → New resource → Public Repository* → URL del
   repo, rama `main`. (Si el repo fuera privado: *Private Repository* con
   la GitHub App o con deploy key.)
   - **Build Pack:** Dockerfile
   - **Base Directory:** `/` (el contexto es la raíz del repo, no `core/`)
   - **Dockerfile Location:** `/core/Dockerfile`
   - **Ports Exposes:** `8080`
   - **Domains:** `https://api.tu-dominio` (Coolify saca el certificado
     de Let's Encrypt; el DNS tiene que apuntar al VPS antes).
   - **Health Check:** path `/health`, puerto `8080`.
   - Si el repo está conectado con la GitHub App, desactivá *Auto Deploy*:
     el deploy lo dispara el workflow (punto 3), y así un push que solo toca
     `web/` no reconstruye el API.
2. *Environment Variables*: solo hacen falta si usás el newsletter (ver
   paso 4). `API_ADDR`, `OUT_DIR` y `SOURCES_PATH` ya vienen en la imagen.
3. *Webhooks* del recurso → copiá la URL de *Deploy Webhook*. *Keys &
   Tokens → API tokens* → token con permiso **deploy** (nada más). En
   GitHub (mismo lugar que `GEMINI_API_KEY`): `COOLIFY_DEPLOY_WEBHOOK` y
   `COOLIFY_API_TOKEN`. El workflow lo llama al final de "Commitear
   reporte".
4. Deploy y probá `https://api.tu-dominio/reports/latest`.
5. *Servers → (tu servidor) → Configuration*: activá la limpieza
   automática de Docker (umbral de disco y frecuencia). Hay un deploy por
   día y cada uno deja una imagen vieja; sin limpieza, el disco del VPS se
   llena con eso, no con los reportes.

IP del cliente: el proxy de Coolify (Traefik) agrega la IP real al final
de `X-Forwarded-For`, que es la que usa `clientIP` — no cambia nada. Si
ponés Cloudflare con proxy (nube naranja) delante, la última entrada pasa
a ser la IP de Cloudflare y todo el tráfico comparte un solo bucket del
rate limit: dejá el DNS en *DNS only* o hay que adaptar `clientIP`.

Por qué el webhook en vez del auto-deploy por push: un webhook de
GitHub hacia el host puede dejar de avisar sin ningún error visible — nos
pasó con el host anterior, y un reporte quedó pusheado pero invisible en
producción varios días. El mismo job que genera el reporte dispara su
propio deploy, y si falla, falla la corrida y te enterás.

Migración desde Render: los mails de confirmación y baja enviados antes
del cambio enlazan a `*.onrender.com`; esos links dejan de funcionar
cuando se apaga ese servicio. Después de apagarlo, borrá el secret
`RENDER_DEPLOY_HOOK_URL` (el paso "Disparar deploy en Render" de
`daily.yml` se saltea solo sin él).

## Paso 3 — Blog (Vercel)

1. vercel.com → *Add New → Project* → importá el repo.
2. **Root Directory:** `web` (Framework: Next.js, se detecta solo).
3. *Environment Variables:* `NOTICIAS_API_URL` = `https://api.tu-dominio`
   (sin `/` final).
4. Deploy.

El blog cachea las páginas 5 minutos (ISR, `revalidate = 300`): la edición
nueva aparece a más tardar 5 minutos después de que Coolify termina de
redeployar, sin redeployar Vercel. Esa cache es lo que evita que un pico de
visitas le pegue al API una vez por visitante.

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
3. **Coolify** (donde corre `cmd/api`) → *Environment Variables* → agregá:
   - `DATABASE_URL` = el connection string de Neon.
   - `BREVO_API_KEY` y `BREVO_SENDER_EMAIL` = los mismos del paso 4 (la API
     manda el mail de confirmación del doble opt-in; sin Brevo, las altas
     nuevas quedan deshabilitadas).
   - `API_BASE_URL` = la URL pública del API, `https://api.tu-dominio`
     (links de confirmación y baja; obligatoria: si falta, caen a
     localhost) y `SITE_URL` = la URL del blog (obligatoria: sin ella los
     mails de confirmación enlazan a localhost).
   - `INTERNAL_API_SECRET` = un valor aleatorio largo (por ejemplo, la
     salida de `openssl rand -hex 32`). El mismo valor va en Vercel (paso 5).
4. **GitHub** (Settings → Secrets and variables → Actions) → agregá, además
   de los secrets del paso 1:
   - `DATABASE_URL` = el mismo connection string de Neon (sí, otra vez —
     es un secret distinto, en un lugar distinto; ver más abajo por qué).
   - `BREVO_API_KEY` = la key generada en el paso 2.
   - `BREVO_SENDER_EMAIL` = el email verificado en Brevo.
   - *Settings → Secrets and variables → Actions → Variables* (no
     *Secrets* — no son sensibles) → `API_BASE_URL` = la URL del API
     (para armar el link de baja en el email) y `SITE_URL` = la URL de
     tu blog en Vercel (para el header, el botón de suscripción y el link
     "Ver edición completa" del email).

5. **Vercel** → *Settings → Environment Variables* → agregá:
   - `INTERNAL_API_SECRET` = el mismo valor que en Coolify. Si no coinciden,
     el API rechaza las altas (403) y el form muestra un error genérico.
   - `TURNSTILE_SITE_KEY` y `TURNSTILE_SECRET_KEY` = de Cloudflare →
     *Turnstile → Add widget* (gratis; dominio: el de tu blog, modo
     *Managed*). Sin las dos, el form funciona sin captcha — no recomendado
     en producción.
   Redeployá Vercel después de cargarlas.

El form de suscripción pega a una route interna de Next.js
(`web/src/app/api/subscribe/route.ts`, mismo origen, sin CORS) que verifica
el captcha y reenvía server-side usando `NOTICIAS_API_URL`. No hace falta
ninguna variable `NEXT_PUBLIC_*`.

**`DATABASE_URL` va en dos lugares distintos y hay que cargarla en los dos**:
Coolify (para que `cmd/api` sirva `POST /subscribers`) y GitHub Actions (para
que `cmd/newsletter` corra en el cron diario). Son procesos separados que no
comparten variables de entorno entre sí — cargar solo uno dejaría la otra
mitad del feature rota en silencio (el form suscribe gente bien, pero nunca
les llega el correo, o viceversa).

## Paso 5 — Seguridad (configuración manual, una sola vez)

Lo que el código no puede configurar solo:

1. **GitHub → Settings → Code security**: activá *Private vulnerability
   reporting* (lo usa `SECURITY.md`), *Dependabot alerts*, *Secret scanning*
   y *Push protection*.
2. **GitHub → Settings → Rules → Rulesets → New branch ruleset** sobre
   `main`: *Restrict deletions* y *Block force pushes*. No agregues
   *Require a pull request*: el bot pushea el informe directo a `main`.
3. **GitHub → Settings → Actions → General**: *Require approval for all
   outside collaborators* en los workflows de forks.
4. **2FA** en GitHub, Coolify, Vercel, Neon, Brevo, Cloudflare y Google AI
   Studio. Límites de gasto o cuota en las keys de LLM (Gemini, OpenRouter).
5. **Dominio del remitente en Brevo**: autenticá SPF, DKIM y DMARC
   (*Senders, Domains & Dedicated IPs → Domains*). Sin eso, Gmail y Yahoo
   mandan el newsletter a spam o lo rechazan.
6. **Monitoreo**: un monitor gratis (UptimeRobot, cron-job.org) sobre
   `/health` cada 10 minutos (el health check de Coolify reinicia el
   contenedor, pero no te avisa si se cae el VPS entero).
7. **VPS**: SSH solo con key, firewall con 22, 80 y 443 abiertos (el panel
   de Coolify detrás de su dominio con TLS, no en `:8000` público),
   actualizaciones automáticas de seguridad y 2FA en el panel.

Ante abuso de las altas: `SUBSCRIPTIONS_ENABLED=false` en Coolify las pausa
sin tocar el resto de la API.

## Verificación de punta a punta

1. Actions → *Run workflow* → esperá el commit del reporte.
2. Coolify muestra un deploy nuevo; `/reports` lista la fecha de hoy.
3. Abrí el blog: última edición arriba, anteriores plegadas, y "Noticias
   utilizadas" con links.

## Problemas comunes

| Síntoma | Causa probable |
|---|---|
| El job falla con `GEMINI_API_KEY no está configurada` | Falta el secret o está mal escrito. |
| El reporte se generó y commiteó pero no aparece en el blog | Coolify no redeployó. Revisá el paso "Disparar deploy en Coolify" del job (si dice "salteo", faltan `COOLIFY_DEPLOY_WEBHOOK`/`COOLIFY_API_TOKEN`; 401, token vencido o sin permiso deploy) y *Deployments* del recurso en Coolify. Para el reporte pendiente, *Redeploy* a mano. |
| `git push` rechazado en el job | Permisos de workflow en *Read only* (paso 1.2) o rama protegida. |
| El blog dice "No se pudo conectar con la API" | `NOTICIAS_API_URL` mal seteada, o el contenedor caído (revisá *Logs* del recurso en Coolify). |
| El blog dice "Todavía no se generó ningún reporte" | `reports/` vacío en el repo o `OUT_DIR` pisado en las variables de Coolify (la imagen ya trae `/app/reports`). |
| Muchos `fetch <medio>` con error en el log | El medio bloquea IPs de GitHub; ver `config/sources_rss.yaml`. |
| El form de suscripción tira "No se pudo conectar con el servidor" | `NOTICIAS_API_URL` mal seteada en Vercel, o el API caído — revisá los logs de la función `/api/subscribe` en Vercel. |
| La gente se suscribe bien pero nunca recibe el correo | `DATABASE_URL` está seteada en Coolify pero no en GitHub Actions (o al revés) — tiene que estar en los dos, ver paso 4. |
| `cmd/newsletter` loguea "no configurados, no se envía" | Falta `DATABASE_URL` o `BREVO_API_KEY` como secret de GitHub Actions. |

## Alternativas

- **Cloudflare Pages / Netlify** en lugar de Vercel para el blog: mismo esquema.
- **Render / Koyeb / Railway** en lugar de Coolify para la API: cualquier
  host que construya `core/Dockerfile` sirve, con su propio deploy hook en
  lugar del de Coolify.
- **Todo en un solo host** (una VM gratuita tipo Oracle Cloud Always Free) con
  cron + `cmd/api` + `next start`: más control, pero más mantenimiento.
