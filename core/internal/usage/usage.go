// Package usage lleva la cuenta del costo de los modelos de lenguaje en una
// corrida, por etapa (clasificación, agrupamiento, redacción, fidelidad).
// OpenRouter informa el costo real de cada pedido cuando se lo pide
// (usage.include); con eso la corrida sabe cuánto gastó, lo publica en el
// registro de la edición y puede saltear etapas opcionales si se pasa del
// presupuesto.
package usage

import (
	"context"
	"sort"
	"sync"
)

type ctxKey int

const (
	stageKey ctxKey = iota
	noReasoningKey
	effortKey
)

// WithStage marca el contexto con la etapa a la que se le carga el costo.
func WithStage(ctx context.Context, stage string) context.Context {
	return context.WithValue(ctx, stageKey, stage)
}

// Stage devuelve la etapa del contexto, o "otros".
func Stage(ctx context.Context) string {
	if s, ok := ctx.Value(stageKey).(string); ok && s != "" {
		return s
	}
	return "otros"
}

// WithoutReasoning pide que los pedidos de este contexto no usen
// razonamiento (tokens de salida extra que se pagan).
func WithoutReasoning(ctx context.Context) context.Context {
	return context.WithValue(ctx, noReasoningKey, true)
}

// WithReasoningEffort pide razonamiento con un esfuerzo acotado ("low",
// "medium", "high"): menos tokens y menos tiempo que el razonamiento por
// defecto, sin apagarlo.
func WithReasoningEffort(ctx context.Context, effort string) context.Context {
	return context.WithValue(ctx, effortKey, effort)
}

// ReasoningEffort devuelve el esfuerzo pedido, o "".
func ReasoningEffort(ctx context.Context) string {
	s, _ := ctx.Value(effortKey).(string)
	return s
}

// ReasoningDisabled indica si el contexto pidió no razonar.
func ReasoningDisabled(ctx context.Context) bool {
	v, _ := ctx.Value(noReasoningKey).(bool)
	return v
}

// Totals acumula lo gastado en una etapa.
type Totals struct {
	CostUSD          float64 `json:"cost_usd"`
	Requests         int     `json:"requests"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
}

var (
	mu     sync.Mutex
	totals = map[string]*Totals{}
)

// Record suma un pedido a la etapa del contexto.
func Record(ctx context.Context, costUSD float64, promptTokens, completionTokens int) {
	mu.Lock()
	defer mu.Unlock()
	t := totals[Stage(ctx)]
	if t == nil {
		t = &Totals{}
		totals[Stage(ctx)] = t
	}
	t.CostUSD += costUSD
	t.Requests++
	t.PromptTokens += promptTokens
	t.CompletionTokens += completionTokens
}

// Total devuelve el costo acumulado de toda la corrida.
func Total() float64 {
	mu.Lock()
	defer mu.Unlock()
	var sum float64
	for _, t := range totals {
		sum += t.CostUSD
	}
	return sum
}

// ByStage devuelve una copia de lo acumulado por etapa.
func ByStage() map[string]Totals {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]Totals, len(totals))
	for k, v := range totals {
		out[k] = *v
	}
	return out
}

// Stages lista las etapas con gasto, ordenadas.
func Stages() []string {
	m := ByStage()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Reset borra lo acumulado (para tests).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	totals = map[string]*Totals{}
}
