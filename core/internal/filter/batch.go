package filter

import (
	"context"
	"strconv"
	"strings"

	"noticias/core/internal/model"
)

// BatchItem es un artículo (ya pasado el prefiltro) listo para clasificar
// en lote.
type BatchItem struct {
	Article model.Article
	Pre     PrefilterResult
}

// BatchResult siempre viene en la misma posición que el BatchItem de
// entrada — el caller nunca tiene que hacer matching por índice, cuenta o
// texto: ClassifyBatch garantiza len(results) == len(items) y cada slot
// resuelto (con Classification o con Err, nunca los dos vacíos).
type BatchResult struct {
	Classification model.Classification
	Err            error
	// Model es el modelo que produjo la clasificación (vacío si no se
	// sabe). Va al registro de auditoría: con una cadena de respaldo, cada
	// titular puede haberlo decidido un modelo distinto.
	Model string
}

// ModelNamer lo cumplen los clasificadores que saben qué modelo usan.
type ModelNamer interface {
	ModelName() string
}

func modelName(c any) string {
	if n, ok := c.(ModelNamer); ok {
		return n.ModelName()
	}
	return ""
}

// batchClassifier es la interfaz opcional que un Classifier puede cumplir
// para clasificar varios artículos en una sola llamada — el objetivo es
// bajar la CANTIDAD de requests (no el trabajo pedido al modelo) para no
// pisar límites por minuto de los free tiers. No forma parte de la
// interfaz Classifier: quien no la implementa cae solo a Classify()
// artículo por artículo vía classifySequential.
type batchClassifier interface {
	ClassifyBatch(ctx context.Context, items []BatchItem) []BatchResult
}

// ClassifyBatch es el punto de entrada que usa cmd/ingest: si c sabe
// clasificar en lote (hoy, OpenAICompatClassifier — cubre Gemini y
// OpenRouter) lo usa; si no (Claude, Ollama), cae a una llamada por
// artículo, que sigue siendo correcta aunque no ahorre requests.
func ClassifyBatch(ctx context.Context, c Classifier, items []BatchItem) []BatchResult {
	var results []BatchResult
	if bc, ok := c.(batchClassifier); ok {
		results = bc.ClassifyBatch(ctx, items)
	} else {
		results = classifySequential(ctx, c, items)
	}
	if name := modelName(c); name != "" {
		for i := range results {
			if results[i].Err == nil && results[i].Model == "" {
				results[i].Model = name
			}
		}
	}
	return results
}

func classifySequential(ctx context.Context, c Classifier, items []BatchItem) []BatchResult {
	results := make([]BatchResult, len(items))
	for i, it := range items {
		cls, err := c.Classify(ctx, it.Article, it.Pre)
		results[i] = BatchResult{Classification: cls, Err: err}
	}
	return results
}

// batchClassifySystemPrompt pide lo mismo que classifySystemPrompt (mismo
// classifyCriteria) pero para una LISTA de items en una sola llamada, cada
// uno identificado por índice — así el caller puede reconciliar la
// respuesta sin depender del orden.
const batchClassifySystemPrompt = `Sos un clasificador de noticias. Te paso una LISTA de titulares+snippets, cada
uno precedido por su índice numérico ("0.", "1.", etc.).
Respondé EXCLUSIVAMENTE con un array JSON (sin texto adicional, sin markdown, sin
envolverlo en un objeto) con UN objeto por CADA item de la lista — mismo índice, no
te saltees ninguno — con esta forma exacta:
[
  {
    "index": <índice numérico del item, tal cual te lo pasé>,
    "is_international": boolean,
    "countries": ["nombre de país", ...],
    "companies": ["nombre de empresa", ...],
    "relation_type": "treaty" | "trade" | "sanction" | "supply_chain" | "regulatory" | "none",
    "confidence": number entre 0 y 1,
    "reason": "una oración breve en español formal de Argentina (voseo: 'vos', 'tenés', etc. — nunca 'tú'; registro profesional, sin modismos coloquiales)"
  }
]
` + classifyCriteria

// buildBatchUserPrompt numera cada item para que la respuesta pueda
// reconciliarse por índice en vez de por orden/cantidad.
func buildBatchUserPrompt(items []BatchItem) string {
	var b strings.Builder
	for i, it := range items {
		b.WriteString(strconv.Itoa(i))
		b.WriteString(". Título: ")
		b.WriteString(it.Article.Title)
		b.WriteString("\nSnippet: ")
		b.WriteString(it.Article.Snippet)
		b.WriteString("\n\n")
	}
	return b.String()
}

// extractJSONArray es el equivalente de extractJSONObject para una
// respuesta que debería ser un array top-level en vez de un objeto.
func extractJSONArray(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.IndexByte(s, '[')
	end := strings.LastIndexByte(s, ']')
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
