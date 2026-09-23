# noticias

Backend independiente que rastrea 115 medios de 23 países, filtra titulares con
potencial internacional/multinacional, y publica un reporte diario en español
formal de Argentina con tres secciones: resumen por región, clima internacional
(comercio/industria/materias primas), y empresas potencialmente afectadas por
región. El blog (Next.js) consume ese reporte vía HTTP. Quien quiera puede
suscribirse con solo su email y recibir por correo el Resumen ejecutivo +
Resumen por región de cada edición.

## Arquitectura

```
config/sources.yaml       115 medios: país, región, orientación editorial, RSS
config/sources_rss.yaml   audit trail de la investigación de feeds (fuente de sources.yaml, con notas)
config/gazetteer.yaml     países/empresas/keywords para el prefiltro barato

core/                     backend Go, independiente de cualquier frontend
  cmd/ingest/             pipeline diario: fetch -> prefiltro -> clasificación LLM -> síntesis -> out/YYYY-MM-DD.md
  cmd/api/                sirve los reportes generados por HTTP + alta/baja de suscriptores del newsletter
  cmd/newsletter/         manda por email el Resumen ejecutivo + Resumen por región a los suscriptores activos
  cmd/checkfeeds/         smoke test de los 90 feeds sin gastar API de Claude
  internal/config/        helpers de entorno (.env, env vars) compartidos entre los cmd/
  internal/model/         tipos compartidos (Source, Article, Classification)
  internal/ingest/        carga de config + parser RSS/Atom/RDF (stdlib, sin dependencias de terceros)
  internal/filter/        prefiltro por gazetteer + clasificador Claude
  internal/report/        síntesis del reporte final vía Claude + extracción de secciones para el email
  internal/store/         conexión a Postgres (Neon) + schema idempotente
  internal/subscriber/    dominio y repositorio de suscriptores del newsletter
  internal/newsletter/    conversión markdown->HTML del email + envío vía Brevo

web/                      Next.js (App Router) — blog que consume core/cmd/api
  src/app/page.tsx             último reporte + archivo de ediciones anteriores + form de suscripción
  src/app/reportes/[fecha]/    reporte de una fecha puntual
  src/lib/api.ts                cliente HTTP hacia core/cmd/api (server-side + el POST de suscripción, client-side)
  src/components/SubscribeForm.tsx   form de alta al newsletter (solo email)
```

## Por qué Go para el core

Memory-safe, binario estático sin dependency hell, stdlib alcanza para HTTP/XML,
config mínima (una env var con la key del proveedor de LLM que elijas — ver
abajo). Única dependencia externa: `gopkg.in/yaml.v3` para leer la config.

## Pipeline de filtrado (dos etapas)

1. **Prefiltro (`internal/filter/prefilter.go`)**: matching por gazetteer
   (países, empresas multinacionales, keywords de comercio/tratados) sobre
   título+snippet. Barato, favorece recall. Pasa si menciona una empresa
   trackeada, una keyword de comercio/tratado, o dos o más países.
2. **Clasificación (`internal/filter/claude.go`)**: a lo que sobrevive el
   prefiltro, Claude Haiku decide con precisión `is_international`,
   entidades, tipo de relación y confianza.

Lo que pasa ambas etapas se sintetiza (`internal/report/synthesize.go`, Claude
Sonnet) en el reporte final de tres secciones, en español formal de Argentina
(voseo, registro profesional).

## Estado de los feeds (115 medios)

Investigado y verificado con requests reales (`go run ./cmd/checkfeeds`):

- **75 con RSS funcionando** (RSS 2.0, Atom o RDF/RSS 1.0)
- **35 sin feed público** (`NO_RSS` en `sources.yaml`) — bloqueados por
  Cloudflare, feed discontinuado, o dominio mal configurado. Alternativa
  liviana anotada en `config/sources_rss.yaml` (sitemap, wp-json, cuenta de
  X) para resolver con scraping puntual más adelante, sin bypass de
  anti-bot ni paywall.
- **4 casos límite conocidos**: `daily-telegraph` (402, gate según User-Agent),
  `le-figaro` y `the-sun` (bloqueo anti-bot real, no solo por UA),
  `toronto-star` (429, rate limit — probablemente se resuelve solo corriendo
  una vez por día en vez de en ráfaga).

Regiones cubiertas: `north_america`, `latin_america`, `europe`, `east_asia`,
`eurasia`, `africa` y `oceania` (las últimas dos, agregadas junto con
Sudáfrica/Egipto/Nigeria y Australia/Nueva Zelanda) — ver `regionLabels` en
`core/internal/report/synthesize.go`.

## Proveedor de LLM: Claude, Gemini, cualquier API compatible con OpenAI, u Ollama local

El core no depende de un solo proveedor. `filter.Classifier` y
`report.Synthesizer` son interfaces con cuatro implementaciones:

- **Claude** (`ClaudeClassifier` / `ClaudeSynthesizer`) — paga, mejor calidad, la más cara.
- **Gemini** — vía `OpenAICompatClassifier`/`OpenAICompatSynthesizer` precargado con el
  endpoint compatible de Google (`/v1beta/openai/`). Tiene free tier — a este volumen
  (~100 clasificaciones + 1 síntesis por día) probablemente no pagás nada, pero el
  límite de tasa exacto depende de tu cuenta: confirmalo en
  [aistudio.google.com/rate-limit](https://aistudio.google.com/rate-limit) después de
  crear la key.
- **`openai_compat`** — el mismo cliente genérico sin los defaults de Gemini, para
  pegarle a OpenAI, DeepSeek o cualquier otra API que hable el formato Chat Completions.
- **Ollama** (`OllamaClassifier` / `OllamaSynthesizer`) — gratis, corre
  local, sin key, sin mandar ni un titular a un servidor de terceros.
  Necesita `ollama serve` corriendo y los modelos bajados
  (`ollama pull qwen3:1.7b`, `ollama pull qwen3.5:9b` o los que prefieras).

Se elige con `LLM_PROVIDER=claude|gemini|openai_compat|ollama`. Sin configurar,
`cmd/ingest` se autoselecciona por la primera key que encuentre, en ese orden, y si no
hay ninguna cae a Ollama. Variables por proveedor (todas opcionales, con default):

```
# Gemini
GEMINI_API_KEY=
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai/
GEMINI_CLASSIFY_MODEL=gemini-3.5-flash-lite
GEMINI_SYNTHESIZE_MODEL=gemini-3.5-flash-lite

# openai_compat genérico (ejemplo con DeepSeek)
OPENAI_COMPAT_API_KEY=
OPENAI_COMPAT_BASE_URL=https://api.openai.com/v1
OPENAI_COMPAT_CLASSIFY_MODEL=gpt-5-nano
OPENAI_COMPAT_SYNTHESIZE_MODEL=gpt-5-nano

# Ollama
OLLAMA_HOST=http://localhost:11434
OLLAMA_CLASSIFY_MODEL=qwen3:1.7b
OLLAMA_SYNTHESIZE_MODEL=qwen3.5:9b
```

### Por qué clasificar en batch (y por qué bajamos titulares por fuente)

El 2026-09-23 Gemini pegó un 429 y el único fallback de OpenRouter de ese
día falló también, ambos a mitad de la corrida — con la clasificación de a
un titular por request, eso significa cientos de requests/día, cada una
compitiendo por el mismo límite por minuto del free tier. Lo que de verdad
se agota no es la cuota diaria (esa alcanza de sobra a este volumen), es el
límite por minuto ante una ráfaga de requests concurrentes.

Dos cambios en consecuencia:
- **`classifyAll` agrupa `classifyBatchSize` (12) titulares por request** en
  vez de mandar uno por request — mismo trabajo pedido al modelo (las
  mismas reglas de `classifyCriteria`), 12x menos llamadas. Si el modelo se
  saltea algún índice de la respuesta, ese puntual se completa individual
  (`OpenAICompatClassifier.ClassifyBatch`) en vez de perderse en silencio.
  `ChainClassifier.ClassifyBatch` banca un proveedor para el resto de la
  corrida solo si le falló el LOTE completo — un fallo de uno o dos items
  puntuales no lo tira, evita empujar tráfico de más al fallback (con cupo
  mucho más chico) por un hipo menor.
- **`MAX_HEADLINES_PER_SOURCE` bajó de 10 a 6** (default, configurable):
  menos titulares candidatos, menos requests en total. Las noticias con
  potencial internacional real suelen estar cerca del tope de cada feed, así
  que el recall que se pierde en la cola de cada fuente es acotado.
- Los clientes compatibles con OpenAI (Gemini, OpenRouter, `openai_compat`)
  ya reintentaban con backoff ante 429/502/503 — ahora también reintentan
  cuando el proveedor devuelve un error "blando" reconocido como transitorio
  (ej. `Provider returned error`, `overloaded`, `timeout`) aunque el status
  HTTP no lo dispare por sí solo, que es justo lo que pasó ese día con
  OpenRouter.

### Fallback: OpenRouter

Independiente del proveedor principal de arriba, si `OPENROUTER_API_KEY` está
configurada, `cmd/ingest` envuelve el classifier y el synthesizer con
`filter.ChainClassifier`/`report.ChainSynthesizer`: una cadena de proveedores
probados en orden — el principal primero, después cada modelo free de
OpenRouter de la lista — hasta que uno responda. Sin esa key, el pipeline
sigue andando exactamente igual que antes, con un solo proveedor.

Por qué una cadena y no un solo fallback: un modelo free individual de
OpenRouter puede fallar puntualmente (`Provider returned error`, timeout) sin
que el proveedor esté caído — con un solo candidato eso pierde el artículo, con
varios prueba el siguiente. Cada link que falla se banca para el resto de la
corrida (no vuelve a pagar el viaje de red a algo ya confirmado caído) — el
costo es que si TODOS los links fallan una vez (ej. Gemini con rate-limit +
el único fallback de OpenRouter con un error puntual), toda la corrida se
queda sin clasificar nada más desde ese momento: nos pasó el 2026-09-23,
Europa y Asia Oriental quedaron con "sin novedades" no porque no hubiera
noticias, sino porque los dos proveedores configurados fallaron a la vez a
mitad de la corrida. Por eso el default incluye `openrouter/free` al final:
es el router gratis de OpenRouter, que reparte internamente entre varios
modelos free en cada request — no se agota ni vence como un id de modelo
puntual.

```
OPENROUTER_API_KEY=
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_CLASSIFY_MODELS=google/gemma-4-26b-a4b-it:free,openrouter/free
OPENROUTER_SYNTHESIZE_MODELS=nvidia/nemotron-3-ultra-550b-a55b:free,openrouter/free
```

Lista distinta por paso, separada por comas: clasificar dispara varias
llamadas chicas en paralelo, sintetizar es una sola llamada grande — un
modelo más pesado ahí tolera ser más lento.

Key en [openrouter.ai/keys](https://openrouter.ai/keys). Los ids de modelos
free rotan con el tiempo — confirmá los vigentes en
[openrouter.ai/models](https://openrouter.ai/models) (filtro "Free") antes de
confiar en los defaults; un id vencido en la lista simplemente se banca al
primer fallo y no rompe nada, pero tampoco suma cobertura real.

Importante: el límite gratis de OpenRouter (50/día sin crédito comprado,
1000/día con $10 de crédito) es **por cuenta, compartido entre todos los
modelos `:free`** — agregar más modelos a la lista no multiplica ese
presupuesto diario, solo da resistencia a que uno específico falle.

Costo aproximado corriendo 1 vez/día (~100 clasificaciones + 1 síntesis):
Gemini free tier ≈ $0 (dentro del límite de tasa), gpt-5-nano/DeepSeek ≈
$0.25-0.60/mes, Claude Haiku+Sonnet ≈ $4.30/mes. El volumen de este proyecto
es chico — la diferencia entre proveedores pagos es centavos, no dólares.

## Correr el pipeline

```bash
cd core
go run ./cmd/ingest       # genera core/out/YYYY-MM-DD.md
go run ./cmd/api          # sirve GET /reports/latest, /reports, /reports/{fecha} en :8080
```

Con una key configurada en `core/.env` (o exportada) usa ese proveedor. Sin
ninguna, usa Ollama local gratis (necesita `ollama serve` corriendo).

`OUT_DIR` default es `./out` (gitignored) — separado a propósito de
`reports/` en la raíz del repo, que es lo que escribe el job de GitHub
Actions (`OUT_DIR=../reports`, ver `.github/workflows/daily.yml`) y lo que
está realmente commiteado/en producción. Corriendo local sin tocar
`OUT_DIR`, `cmd/api` sirve tus pruebas en `./out`, no los reportes reales —
para ver esos últimos con la API local:

```bash
OUT_DIR=../reports go run ./cmd/api
```

No pongas `OUT_DIR=../reports` de forma permanente en `core/.env`: una
corrida de prueba de `cmd/ingest` con esa config escribiría directo en la
carpeta versionada por git, mezclando basura de pruebas con el historial
compartido. Pasalo puntual según qué necesites: testear el pipeline (`./out`,
default, descartable) o ver lo que está commiteado (`../reports`).

Para que corra todos los días, agendá `ingest` con cron (ej. 07:00) y dejá
`cmd/api` y el blog levantados; el blog lee siempre el disco vía la API, así que
la edición nueva aparece sin redeploy:

```cron
0 7 * * * cd /ruta/a/noticias/core && /usr/local/go/bin/go run ./cmd/ingest >> ingest.log 2>&1
```

El home muestra la última edición abierta y las anteriores (hasta 30) como
tarjetas plegables. Cada edición cierra con "Noticias utilizadas": lista de
enlaces a las notas originales, por región.

## Newsletter (alta con solo email)

El home tiene un form de suscripción que pide únicamente un email — nada de
nombre ni otros datos. El submit pega a `web/src/app/api/subscribe/route.ts`
(una route interna de Next.js, mismo origen que el browser, sin CORS), que
del lado del servidor reenvía a `cmd/api` usando `NOTICIAS_API_URL` — la
misma variable server-only que ya usa el resto de `web/`, sin necesitar un
`NEXT_PUBLIC_*` nuevo. `cmd/api` lo guarda en Postgres (Neon) y, después de
que `cmd/ingest` genera la edición del día, `cmd/newsletter` manda por
correo (vía Brevo) el `## Resumen ejecutivo` + `## Resumen por región` de
esa edición a todos los suscriptores activos, con un link de baja propio por
suscriptor en el pie del correo (ese link sí lo abre el browser directo
contra `cmd/api`, pero como navegación normal — no un fetch — tampoco pasa
por CORS).

El email sigue el mismo estilo editorial rojo/negro que el blog (header con
nav, botón de suscripción, cita del resumen ejecutivo con borde rojo).
Regiones sin novedades ese día (pasa seguido — ver el fix del bug de
regiones más abajo) no reciben el mismo espacio que una con contenido real:
se agrupan compactas en una fila aparte (`core/internal/newsletter/regions.go`),
para que el layout no se vea roto en un día con dos o tres regiones vacías.
No incluye dirección postal física en el pie — si la necesitás por norma
legal de tu jurisdicción, agregala a mano en `core/internal/newsletter/template.go`.

Es opcional en los tres niveles: sin `DATABASE_URL`, `cmd/api` no registra
las rutas de `/subscribers*` (el resto de la API sigue igual); sin
`DATABASE_URL`/`BREVO_API_KEY`, `cmd/newsletter` no hace nada; sin ninguna de
las dos, el pipeline diario (`.github/workflows/daily.yml`) sigue publicando
el reporte exactamente igual que antes.

**Doble opt-in.** El alta no activa a nadie: `cmd/api` guarda el email como
pendiente y le manda un mail de confirmación (Brevo). Solo quien hace click
en ese mail recibe el newsletter; las altas sin confirmar se borran a los 7
días. Así nadie puede anotar direcciones ajenas. Los suscriptores que ya
existían antes de este cambio quedan confirmados por la migración.
Confirmación y baja son en dos pasos (el link del mail muestra un botón y la
acción es un POST), para que los antivirus de correo que abren todos los
links no confirmen ni den de baja solos. El newsletter lleva los headers
`List-Unsubscribe` / `List-Unsubscribe-Post` (baja en un click, RFC 8058).

Variables:

```
# core/cmd/api y core/cmd/newsletter
DATABASE_URL=postgres://...          # connection string de Neon (o cualquier Postgres)
BREVO_API_KEY=                       # API key de Brevo (transactional) — en cmd/api, para el mail de confirmación
BREVO_SENDER_EMAIL=                  # remitente verificado en Brevo
BREVO_SENDER_NAME=Radar Global
API_BASE_URL=https://tu-api.onrender.com   # host de los links de confirmación y baja
SITE_URL=https://tu-blog.vercel.app        # host del header/CTA y del link a la edición completa

# core/cmd/api — protección de las altas
INTERNAL_API_SECRET=                 # secreto compartido con web/ (el mismo valor en Vercel); sin él, POST /subscribers acepta cualquier origen
SUBSCRIPTIONS_ENABLED=true           # "false" pausa las altas nuevas (kill switch ante abuso)
CONFIRM_EMAILS_PER_DAY=100           # tope global de mails de confirmación por 24h (protege la cuota de Brevo)
CONFIRM_RESEND_AFTER_MINUTES=60      # mínimo entre dos mails de confirmación al mismo email
SUBSCRIBE_PER_HOUR=5                 # altas por IP por hora
RATE_LIMIT_PER_MINUTE=120            # requests por IP por minuto, toda la API

# core/cmd/newsletter
REPORT_DATE=2026-09-23                     # opcional; sin setear usa "hoy" (UTC)

# web/ (Vercel)
INTERNAL_API_SECRET=                 # el mismo valor que en cmd/api
TURNSTILE_SITE_KEY=                  # Cloudflare Turnstile (captcha); sin las dos keys, el form anda sin captcha
TURNSTILE_SECRET_KEY=
```

`DATABASE_URL` tiene que estar seteada en **dos lugares** por separado: en el
servicio de Render (para que `cmd/api` sirva `/subscribers`) y como secret de
GitHub Actions (para que `cmd/newsletter` corra en el cron diario) — cargar
solo uno de los dos deja la otra mitad rota en silencio.

Correrlo local de punta a punta:

```bash
cd core
DATABASE_URL=... BREVO_API_KEY=... go run ./cmd/api        # habilita /subscribers*
curl -X POST localhost:8080/subscribers -H 'Content-Type: application/json' -d '{"email":"vos@ejemplo.com"}'

# con un reports/{fecha}.md ya generado:
DATABASE_URL=... BREVO_API_KEY=... REPORT_DATE=2026-09-23 go run ./cmd/newsletter
```

## Integridad de los informes

El objetivo es que nadie tenga que creer en nuestra palabra de que las
noticias no se manipulan: todo se puede verificar desde afuera. La versión
para lectores está en `/metodologia` del blog.

- **Registro de auditoría por edición** (`reports/FECHA.audit.json`): cada
  titular descargado y su destino (descartado por el prefiltro, descartado o
  aceptado por el clasificador con su motivo, o error), los modelos usados,
  el hash SHA-256 de cada prompt de sistema, el commit del código y el link
  al log de la corrida en Actions. `cmd/api` expone el resumen como
  `provenance` en `/reports/*` y el blog lo muestra en "Cómo se hizo esta
  edición".
- **Firma de cada edición**: `daily.yml` genera una atestación Sigstore
  (`actions/attest-build-provenance`) del `.md`, `.sources.json` y
  `.audit.json`. Cualquiera puede comprobar que el archivo publicado es el
  que generó el workflow, sin cambios posteriores:
  `gh attestation verify reports/AAAA-MM-DD.md -R patriciobo/noticias-geopolitica`.
- **Guardia de integridad** (`.github/workflows/reports-guard.yml`): falla
  en público si un commit que no es del bot modifica o borra un informe ya
  publicado.
- **Lista de medios pública**: `GET /sources` y `config/sources.yaml`, con
  país, región y orientación editorial.

### Defensas contra prompt injection

Los titulares vienen de terceros y llegan al LLM, así que se tratan como
dato no confiable:

- `internal/ingest/clean.go`: saca saltos de línea y caracteres de control
  (un titular no puede hacerse pasar por otro item de la lista), limita el
  largo y descarta links que no sean http(s).
- Los prompts de clasificación y síntesis aclaran que el contenido de los
  medios es material a procesar, no instrucciones.
- `report.StripUnknownLinks`: del texto del LLM se sacan los enlaces que no
  correspondan a una nota procesada. La sección "Noticias utilizadas" la
  arma el código, no el modelo.

## Seguridad

Ver `SECURITY.md` para reportar vulnerabilidades. Resumen de lo que hay:

- **API** (`core/cmd/api`): timeouts de servidor, rate limit por IP,
  headers de seguridad (CSP estricta, `Referrer-Policy: no-referrer` porque
  las URLs de confirmación y baja llevan tokens), `Cache-Control` en
  `/reports*`.
- **Altas**: honeypot + Cloudflare Turnstile + chequeo de origen en
  `web/src/app/api/subscribe/route.ts`; secreto compartido, rate limit por
  IP, tope diario global y doble opt-in en `cmd/api`.
- **Blog**: CSP, HSTS, `X-Frame-Options`, `Permissions-Policy`; páginas
  cacheadas con ISR (5 minutos), así un pico de tráfico lo absorbe el CDN de
  Vercel y no la instancia gratis de Render.
- **Repo**: actions fijadas por SHA, permisos mínimos por job, CodeQL y
  Dependabot.

## Cómo probar la aplicación

Ver sección dedicada más abajo con el detalle paso a paso (tests automáticos,
smoke test de feeds sin costo, corrida real del pipeline, y el blog).

## Estado

- [x] Config de 115 medios con región/orientación editorial
- [x] `config/sources.yaml` — campo `rss` completo (64/90 con feed real, 22 `NO_RSS` documentados)
- [x] Ingesta RSS/Atom/RDF genérica, con soporte de charset no-UTF8
- [x] Prefiltro por gazetteer
- [x] Clasificación LLM por artículo
- [x] Síntesis del reporte de 3 secciones, en español formal de Argentina
- [x] API HTTP mínima sobre archivos
- [x] Tests automáticos (`go test ./...`) y smoke test de feeds (`cmd/checkfeeds`)
- [x] Blog Next.js consumiendo la API del core
- [x] Persistencia en Postgres (Neon) — hoy solo para suscriptores del newsletter
- [x] Newsletter por email (alta con solo email, baja con link propio, envío vía Brevo)
- [ ] Programar corrida diaria (cron)
- [ ] Resolver scraping puntual para los 22 `NO_RSS` (sitemap/wp-json donde aplica)

## Cómo probar la aplicación

### 1. Tests automáticos del core (sin costo, sin red)

```bash
cd core
go test ./...
```

Cubre el prefiltro por gazetteer y los tres parsers de feed (RSS 2.0, Atom,
RDF), incluyendo decodificación de charsets no-UTF8. No necesita
`ANTHROPIC_API_KEY` ni conexión a internet.

### 2. Smoke test de los 90 feeds (sin costo — no llama a Claude)

```bash
cd core
go run ./cmd/checkfeeds
```

Hace un request real a cada feed configurado y muestra un titular de
ejemplo por medio que responde, más la lista de los que fallaron al final.
Sirve para revalidar el estado de los feeds sin gastar créditos de API.
Último resultado conocido: 75 OK, 35 `NO_RSS` (esperado), el resto son los
casos límite (ver arriba).

### 3. Corrida real del pipeline (consume créditos de la API de Claude)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
cd core
go run ./cmd/ingest
```

Con las fuentes activas (hasta `MAX_HEADLINES_PER_SOURCE`, default 6,
titulares cada una) esto clasifica lo que pasa el prefiltro barato en lotes
de `classifyBatchSize` (12 titulares por request, no uno por request — ver
"Por qué batch" más abajo) más un llamado de síntesis final. Si querés
probar con menos costo antes de correrlo completo, editá
`config/sources.yaml` temporalmente y dejá activos (sin `NO_RSS`) solo 5 o 6
medios de un par de regiones.

Al terminar, el reporte queda en `core/out/YYYY-MM-DD.md`. Abrilo directo
para chequear que las tres secciones (resumen por región, clima
internacional, empresas afectadas) tengan sentido y estén en español formal
de Argentina (voseo).

### 4. API HTTP del core

```bash
cd core
go run ./cmd/api
```

En otra terminal:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/reports
curl http://localhost:8080/reports/latest
curl http://localhost:8080/reports/2026-09-14   # con la fecha que corresponda
```

`/reports/latest` y `/reports/{fecha}` devuelven 404 si todavía no corriste
el pipeline (paso 3) para esa fecha.

### 5. Blog Next.js

```bash
cd web
cp .env.example .env.local   # NOTICIAS_API_URL=http://localhost:8080 por defecto
npm run dev
```

Con `core/cmd/api` corriendo (paso 4) y al menos un reporte generado (paso
3), abrí `http://localhost:3000`. Tiene que mostrar el reporte del día con
las tres secciones y, si hay más de una fecha, un archivo de "Ediciones
anteriores" abajo con links a `/reportes/{fecha}`.

Si abrís el blog sin haber corrido el pipeline todavía, o sin la API
corriendo, la página muestra un mensaje explicando qué falta en vez de
romper — es el comportamiento esperado, no un bug.

`npm run build` también sirve como chequeo rápido de tipos/compilación sin
necesitar el backend levantado.

### 6. Newsletter (opcional — necesita Postgres y Brevo)

```bash
cd core
DATABASE_URL=postgres://... BREVO_API_KEY=... go run ./cmd/api
```

Suscribite desde el form del blog o directo con curl:

```bash
curl -X POST localhost:8080/subscribers -H 'Content-Type: application/json' -d '{"email":"vos@ejemplo.com"}'
```

Con un reporte ya generado (paso 3), mandá el correo de esa fecha:

```bash
DATABASE_URL=postgres://... BREVO_API_KEY=... REPORT_DATE=2026-09-23 go run ./cmd/newsletter
```

Revisá la bandeja: el correo debe traer solo Resumen ejecutivo + Resumen por
región, con un link de baja al pie. Clickearlo confirma la baja
(`unsubscribed_at` en la tabla `subscribers`) y una corrida posterior de
`cmd/newsletter` ya no le manda nada a ese email.
