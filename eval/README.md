# Evaluación del clasificador

`classify.jsonl` tiene titulares reales etiquetados a mano como
internacionales o no, con el mismo criterio que el prompt del clasificador
(`core/internal/filter/claude.go`, `classifyCriteria`). Una línea por
titular:

- `expected`: la etiqueta correcta.
- `note`: por qué, en los casos límite.
- `labeled_by`: quién etiquetó. Las etiquetas iniciales (115 titulares del
  2026-09-24, cinco por país) las hizo un asistente de IA y **necesitan
  revisión humana**; al revisar una, cambiá `labeled_by` por tu nombre.

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
