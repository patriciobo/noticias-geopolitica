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
- [ ] **2. Ficha por medio.** Campo `ownership` en `config/sources.yaml`
  (`estatal` | `privado` | `partidario` | `ong` | `exilio`) y publicado en
  `GET /sources`. Clasificación inicial hecha por Claude — revisar.
- [ ] **3. Atribuir siempre.** El prompt de síntesis recibe el tipo de medio
  y exige "según X" en todo lo afirmado; los medios estatales se nombran
  como tales ("la agencia estatal iraní IRNA").
- [ ] **4. "Sin datos" ≠ "sin novedades".** Cobertura por región (medios
  que respondieron / configurados); con cobertura baja la región dice
  "Cobertura insuficiente hoy (X de Y medios respondieron)".
- [ ] **5. Monitor de feeds.** El registro de auditoría guarda los medios
  que fallaron o solo trajeron notas viejas; la web los muestra en "Cómo se
  hizo esta edición".
- [ ] **6. Modelo por titular.** Cada entrada del registro guarda qué
  modelo la clasificó; la edición avisa si se usó un modelo de respaldo.
- [ ] **7. Versiones visibles.** Regenerar una edición incrementa su
  versión y guarda el motivo (input `motivo` del workflow); la web muestra
  "Versión N — motivo". Página /correcciones.
- [ ] **8. Reportar un error.** Plantilla de issue en el repo y link
  "¿Encontraste un error?" en cada edición y en el newsletter.
- [ ] **9. Cada afirmación con su cita.** Los titulares van numerados al
  modelo; cada bullet termina en [n]; el código valida que las citas
  existan, las convierte en links a la nota y cuenta bullets sin cita.
- [ ] **10. Confirmación independiente.** Detectar cables de agencias
  (Reuters, AFP, AP, EFE, ...) en la cobertura cruzada: contar "fuentes
  independientes" aparte de "medios que lo publicaron".
- [ ] **11. Chequeo de fidelidad.** Segunda pasada del LLM que contrasta
  cada bullet con sus notas citadas; se publica cuántos quedaron sin
  respaldo y cuáles.
- [ ] **12. Evaluación del clasificador.** Conjunto de titulares
  etiquetados (`eval/classify.jsonl`) y `cmd/evalclassify` que mide
  precisión/cobertura por modelo. Etiquetas iniciales de Claude — revisar.
- [ ] **13. Revisión humana por muestreo.** Workflow semanal que abre un
  issue con 20 clasificaciones y 10 bullets al azar para revisar a mano.
- [ ] **14. Gobernanza [necesita al usuario].** Página /quienes-somos:
  responsable, contacto, financiamiento, conflictos de interés, política
  de correcciones. Se deja el borrador con los datos faltantes marcados.

## Fuera de alcance por ahora

- Titular original junto a la traducción: la lista "Noticias utilizadas" ya
  muestra el titular original; agregar traducción queda para después.
- Aprobación humana antes de mandar el newsletter: cambia el horario de
  envío; decidir después de ver cómo anda el paso 13.

## Registro

(Una línea por paso terminado: fecha, commit, notas.)
- Paso 1 (2026-09-24): AutomationNotice en cada edición, sección "Qué es y qué no es" en /metodologia, aviso en el pie del newsletter.
