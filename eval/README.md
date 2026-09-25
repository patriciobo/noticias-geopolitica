# Evaluación del clasificador

`classify.jsonl` tiene titulares reales etiquetados a mano como
internacionales o no, con el mismo criterio que el prompt del clasificador
(`core/internal/filter/claude.go`, `classifyCriteria`). Una línea por
titular:

- `expected`: la etiqueta correcta.
- `note`: por qué, en los casos límite.
- `labeled_by`: quién etiquetó.
- `title_es`: traducción al español del titular.
- `votes` y `verification`: resultado de la verificación con tres jueces
  (ver abajo).

## Cómo se verificaron las etiquetas

Las etiquetas iniciales (115 titulares del 2026-09-24, cinco por país) las
hizo un asistente de IA. Después, `verificar_etiquetas.py` le pidió a tres
modelos de proveedores distintos que clasificaran cada titular por su
cuenta, con el mismo criterio que el clasificador: Claude Sonnet 5
(Anthropic), Gemini 3.5 Flash (Google) y GPT-5.4 mini (OpenAI). Ninguno de
los tres es un modelo que la plataforma use para clasificar, así la
evaluación no compara a los modelos de producción consigo mismos.

Regla: una etiqueta se confirma o se cambia solo si los tres jueces
coinciden. Sin unanimidad queda como estaba, marcada "sin consenso".
Resultado del 2026-09-25: 96 confirmadas, 3 cambiadas y 16 sin consenso
(casos límite: la cotización del dólar, una encuesta electoral, decisiones
migratorias domésticas). Es una verificación automática, no humana: si
encontrás una etiqueta mal, corregila y anotá tu nombre en `labeled_by`.

Los titulares no tienen copete (`snippet` vacío): el registro de auditoría
solo guarda títulos. En producción el clasificador también ve el copete, así
que esta evaluación es un poco más difícil que el caso real.

## Correr

Desde GitHub: *Actions → Evaluar clasificador → Run workflow*, con la lista
de modelos. El resultado queda en el resumen de la corrida.

En local, desde `core/`:

```sh
EVAL_API_KEY=sk-or-... EVAL_MODELS=deepseek/deepseek-v4-flash,qwen/qwen3.8-27b:free \
  go run ./cmd/evalclassify
```

Devuelve precisión (de lo que marcó como internacional, cuánto lo era),
cobertura (de lo internacional, cuánto encontró), F1 y la lista de
desacuerdos por modelo. Antes de cambiar el modelo de producción, correrla
con el candidato.
