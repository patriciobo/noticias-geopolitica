# Plan: que la plataforma resista una crítica periodística dura

Origen: autocrítica del 2026-09-24 (verificación, atribución, "cobertura
cruzada", etiquetas de medios, fallas silenciosas, versiones, gobernanza).
Cada paso es independiente, se commitea por separado y se marca acá al
terminarlo. Para retomar: leer este archivo y seguir por el primer paso sin
marcar. Los pasos marcados **[necesita al usuario]** requieren datos o
decisiones que no se pueden inferir del código.

## Estado

- [x] **1. Decir qué es.** Aviso "resumen automatizado de titulares, no
  verifica hechos: atribuye lo que publica cada medio" en cada edición de la
  web, en el newsletter y en /metodologia.
- [x] **2. Ficha por medio.** Campo `ownership` en `config/sources.yaml`
  (`estatal` | `privado` | `partidario` | `ong` | `exilio`) y publicado en
  `GET /sources`. Clasificación inicial hecha por Claude — revisar.
- [x] **3. Atribuir siempre.** El prompt de síntesis recibe el tipo de medio
  y exige "según X" en todo lo afirmado; los medios estatales se nombran
  como tales ("la agencia estatal iraní IRNA").
- [x] **4. "Sin datos" ≠ "sin novedades".** Cobertura por región (medios
  que respondieron / configurados); con cobertura baja la región dice
  "Cobertura insuficiente hoy (X de Y medios respondieron)".
- [x] **5. Monitor de feeds.** El registro de auditoría guarda los medios
  que fallaron o solo trajeron notas viejas; la web los muestra en "Cómo se
  hizo esta edición".
- [x] **6. Modelo por titular.** Cada entrada del registro guarda qué
  modelo la clasificó; la edición avisa si se usó un modelo de respaldo.
- [x] **7. Versiones visibles.** Regenerar una edición incrementa su
  versión y guarda el motivo (input `motivo` del workflow); la web muestra
  "Versión N — motivo". Página /correcciones.
- [x] **8. Reportar un error.** Plantilla de issue en el repo y link
  "¿Encontraste un error?" en cada edición y en el newsletter.
- [x] **9. Cada afirmación con su cita.** Los titulares van numerados al
  modelo; cada bullet termina en [n]; el código valida que las citas
  existan, las convierte en links a la nota y cuenta bullets sin cita.
- [x] **10. Confirmación independiente.** Detectar cables de agencias
  (Reuters, AFP, AP, EFE, ...) en la cobertura cruzada: contar "fuentes
  independientes" aparte de "medios que lo publicaron".
- [x] **11. Chequeo de fidelidad.** Segunda pasada del LLM que contrasta
  cada bullet con sus notas citadas; se publica cuántos quedaron sin
  respaldo y cuáles.
- [x] **12. Evaluación del clasificador.** Conjunto de titulares
  etiquetados (`eval/classify.jsonl`) y `cmd/evalclassify` que mide
  precisión/cobertura por modelo. Etiquetas iniciales de Claude — revisar.
- [x] **13. Revisión humana por muestreo.** Workflow semanal que abre un
  issue con 20 clasificaciones y 10 bullets al azar para revisar a mano.
- [ ] **14. Gobernanza [necesita al usuario].** Página /quienes-somos:
  responsable, contacto, financiamiento, conflictos de interés, política
  de correcciones. Borrador listo en `docs/quienes-somos-borrador.md`;
  falta que el usuario conteste las 5 preguntas del final para publicarla.

## Pendientes detectados al probar (2026-09-24)

- [x] **15. Agrupamiento de la cobertura cruzada.** Agrupa por países +
  tipo de relación (o empresa + relación), así que mezcla historias
  distintas: en la prueba local el chequeo de fidelidad marcó 3 de las 8
  afirmaciones sin respaldo en "Cobertura cruzada" porque el título del
  grupo no correspondía a todas sus notas. Agrupar también por similitud
  de título (o pedirle al modelo que agrupe) antes de contar fuentes.
- [ ] **16. Revisar las etiquetas de `eval/classify.jsonl`** (hechas por
  Claude) y la clasificación `ownership` de `config/sources.yaml`
  **[necesita al usuario]**.

## Fuera de alcance por ahora

- Titular original junto a la traducción: la lista "Noticias utilizadas" ya
  muestra el titular original; agregar traducción queda para después.
- Aprobación humana antes de mandar el newsletter: cambia el horario de
  envío; decidir después de ver cómo anda el paso 13.

## Registro

(Una línea por paso terminado: fecha, commit, notas.)
- Paso 1 (2026-09-24): AutomationNotice en cada edición, sección "Qué es y qué no es" en /metodologia, aviso en el pie del newsletter.
- Paso 2 (2026-09-24): ownership + ownership_note en los 131 medios (13 estatales, 3 partidarios, 3 ONG, 6 exilio); publicado en GET /sources y /metodologia. Revisar la clasificación.
- Paso 3 (2026-09-24): Reglas de atribución obligatoria en el prompt de síntesis; cada titular llega con 'tipo: estatal (nota)'; 'confirman' → 'publicaron' en cobertura cruzada.
- Paso 4 (2026-09-24): report.Input con cobertura por región; región vacía con menos de la mitad de sus medios respondiendo dice 'Cobertura insuficiente hoy (X de Y medios respondieron).'; EnsureAllRegionsPresent ahora corre para todos los proveedores.
- Paso 5 (2026-09-24): SourceStatus por medio en el audit (sources) y los que fallaron en provenance.source_problems; la web los lista en 'Cómo se hizo esta edición'. Mismo commit que el paso 4.
- Paso 6 (2026-09-24): AuditEntry.model por titular; provenance.classify_models_used (conteo) y synthesize_model_used; la web lista los modelos usados y avisa si entró un respaldo.
- Paso 7 (2026-09-24): provenance.version/revisions con motivo (input 'motivo' del workflow → REGENERATION_REASON); GET /corrections; aviso de versión en la edición; página /correcciones con política y nota histórica del 24/09.
- Paso 8 (2026-09-24): Plantillas .github/ISSUE_TEMPLATE/error-en-edicion.yml y objecion-medio.yml (labels creados); link '¿Encontraste un error?' con la fecha precargada en cada edición y en el newsletter.
- Paso 9 (2026-09-24): Titulares numerados en el prompt; report.ResolveCitations convierte [n] en link a la nota, saca números inventados y cuenta bullets sin cita (counts.citations_*, uncited_bullets); web muestra superíndices; el newsletter renderiza links.
- Paso 10 (2026-09-24): report/wire.go detecta cables (Reuters, AFP, AP, EFE, ...) y agrupa medios estatales del mismo país como una voz; la cobertura cruzada informa fuentes independientes y cables, y el peso usa voces, no medios.
- Paso 11 (2026-09-24): report.CheckFidelity: lotes de 30 afirmaciones con sus notas citadas → respaldada/inferencia/sin_respaldo; counts.claims_* y fidelity_issues en la edición; FIDELITY_CHECK=off lo apaga. Lo hace el mismo modelo que redactó (control de consistencia, no revisión independiente).
- Paso 13 (2026-09-24): core/cmd/muestra + workflow semanal (lunes) que abre un issue 'revision-semanal' con 20 titulares y 10 afirmaciones al azar de los últimos 7 días, con checklist.
- Paso 12 (2026-09-24): eval/classify.jsonl (115 titulares, etiquetas de Claude a revisar), cmd/evalclassify, workflow manual 'Evaluar clasificador'. Primer resultado: Qwen 3.8 27B free F1 0.85 vs DeepSeek V4 Flash 0.71 → la clasificación pasa a Qwen con DeepSeek de respaldo.
- Paso 15 (2026-09-24): report.GroupStories: el modelo agrupa las notas por hecho concreto (multilingüe) con validación; la cobertura cruzada usa el hecho del grupo como título; si falla, vuelve al agrupamiento por entidades.
- Prueba completa local (2026-09-24, config de producción): 499 titulares, 205 aceptados, 18 historias agrupadas por hecho, 344 citas resueltas y 0 inventadas, 4 bullets sin cita; fidelidad 92 chequeadas / 48 respaldadas / 28 inferencias / 16 sin respaldo (6 eran conteos de cobertura, corregido en 69d692b). Qwen free falló 7 lotes ("Provider returned error") y DeepSeek los cubrió: vigilar si conviene volver a DeepSeek como principal.
