# noticias

Backend independiente que rastrea 90 medios de 17 países, filtra titulares con
potencial internacional/multinacional, y publica un reporte diario en español
formal de Argentina con tres secciones: resumen por región, clima internacional
(comercio/industria/materias primas), y empresas potencialmente afectadas por
región. El blog (Next.js) consume ese reporte vía HTTP.

## Arquitectura

```
config/sources.yaml       90 medios: país, región, orientación editorial, RSS
config/sources_rss.yaml   audit trail de la investigación de feeds (fuente de sources.yaml, con notas)
config/gazetteer.yaml     países/empresas/keywords para el prefiltro barato

core/                     backend Go, independiente de cualquier frontend
  cmd/ingest/             pipeline diario: fetch -> prefiltro -> clasificación LLM -> síntesis -> out/YYYY-MM-DD.md
  cmd/api/                sirve los reportes generados por HTTP (para el blog Next.js y, a futuro, la app mobile)
  cmd/checkfeeds/         smoke test de los 90 feeds sin gastar API de Claude
  internal/model/         tipos compartidos (Source, Article, Classification)
  internal/ingest/        carga de config + parser RSS/Atom/RDF (stdlib, sin dependencias de terceros)
  internal/filter/        prefiltro por gazetteer + clasificador Claude
  internal/report/        síntesis del reporte final vía Claude

web/                      Next.js (App Router) — blog que consume core/cmd/api
  src/app/page.tsx             último reporte + archivo de ediciones anteriores
  src/app/reportes/[fecha]/    reporte de una fecha puntual
  src/lib/api.ts                cliente HTTP hacia core/cmd/api (server-side only)
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

## Estado de los feeds (90 medios)

Investigado y verificado con requests reales (`go run ./cmd/checkfeeds`):

- **64 con RSS funcionando** (RSS 2.0, Atom o RDF/RSS 1.0)
- **22 sin feed público** (`NO_RSS` en `sources.yaml`) — bloqueados por
  Cloudflare, feed discontinuado, o dominio mal configurado. Alternativa
  liviana anotada en `config/sources_rss.yaml` (sitemap, wp-json, cuenta de
  X) para resolver con scraping puntual más adelante, sin bypass de
  anti-bot ni paywall.
- **4 casos límite conocidos**: `daily-telegraph` (402, gate según User-Agent),
  `le-figaro` y `the-sun` (bloqueo anti-bot real, no solo por UA),
  `toronto-star` (429, rate limit — probablemente se resuelve solo corriendo
  una vez por día en vez de en ráfaga).

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
corrida (no vuelve a pagar el viaje de red a algo ya confirmado caído), así
que agregar más candidatos no tiene costo si terminan sin usarse.

```
OPENROUTER_API_KEY=
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_CLASSIFY_MODELS=google/gemma-4-26b-a4b-it:free,otro/modelo:free
OPENROUTER_SYNTHESIZE_MODELS=nvidia/nemotron-3-ultra-550b-a55b:free,otro/modelo:free
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

## Cómo probar la aplicación

Ver sección dedicada más abajo con el detalle paso a paso (tests automáticos,
smoke test de feeds sin costo, corrida real del pipeline, y el blog).

## Estado

- [x] Config de 90 medios con región/orientación editorial
- [x] `config/sources.yaml` — campo `rss` completo (64/90 con feed real, 22 `NO_RSS` documentados)
- [x] Ingesta RSS/Atom/RDF genérica, con soporte de charset no-UTF8
- [x] Prefiltro por gazetteer
- [x] Clasificación LLM por artículo
- [x] Síntesis del reporte de 3 secciones, en español formal de Argentina
- [x] API HTTP mínima sobre archivos
- [x] Tests automáticos (`go test ./...`) y smoke test de feeds (`cmd/checkfeeds`)
- [x] Blog Next.js consumiendo la API del core
- [ ] Programar corrida diaria (cron)
- [ ] Resolver scraping puntual para los 22 `NO_RSS` (sitemap/wp-json donde aplica)
- [ ] Persistencia en Postgres (cuando se necesite historial/búsqueda más allá de archivos por fecha)

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
Último resultado conocido: 64 OK, 22 `NO_RSS` (esperado), 4 casos
límite (ver arriba).

### 3. Corrida real del pipeline (consume créditos de la API de Claude)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
cd core
go run ./cmd/ingest
```

Con 64 fuentes activas y hasta 10 titulares cada una, esto hace más o menos
64 llamados baratos a Claude Haiku (solo para lo que pasa el prefiltro, así
que en la práctica van a ser bastantes menos) más un llamado a Claude Sonnet
para la síntesis final. Si querés probar con menos costo antes de correrlo
completo, editá `config/sources.yaml` temporalmente y dejá activos (sin
`NO_RSS`) solo 5 o 6 medios de un par de regiones.

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
