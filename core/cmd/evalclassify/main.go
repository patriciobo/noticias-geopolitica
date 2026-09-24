// Command evalclassify mide el clasificador contra un conjunto de
// titulares etiquetados a mano (eval/classify.jsonl): precisión, cobertura
// (recall) y F1 por modelo, y la lista de desacuerdos. Sirve para comparar
// modelos antes de cambiar el de producción y para publicar cuánto se
// equivoca el que se usa.
//
// Uso (desde core/):
//
//	EVAL_API_KEY=... EVAL_MODELS=deepseek/deepseek-v4-flash,google/gemma-4-31b-it:free \
//	  go run ./cmd/evalclassify > ../eval/resultados.md
//
// EVAL_BASE_URL por defecto es OpenRouter; para Gemini usar
// https://generativelanguage.googleapis.com/v1beta/openai/ y su key.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"noticias/core/internal/filter"
	"noticias/core/internal/model"
)

type example struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Snippet  string `json:"snippet"`
	Source   string `json:"source"`
	Country  string `json:"country"`
	Expected bool   `json:"expected"`
	Note     string `json:"note,omitempty"`
}

const batchSize = 16

func main() {
	path := envOr("EVAL_PATH", "../eval/classify.jsonl")
	baseURL := envOr("EVAL_BASE_URL", "https://openrouter.ai/api/v1")
	key := os.Getenv("EVAL_API_KEY")
	if key == "" {
		key = os.Getenv("OPENROUTER_API_KEY")
	}
	models := strings.Split(envOr("EVAL_MODELS", "deepseek/deepseek-v4-flash"), ",")
	if key == "" {
		log.Fatal("falta EVAL_API_KEY (o OPENROUTER_API_KEY)")
	}

	examples, err := load(path)
	if err != nil {
		log.Fatalf("leyendo %s: %v", path, err)
	}

	fmt.Printf("# Evaluación del clasificador\n\n%d titulares etiquetados (%s), %d internacionales.\n\n", len(examples), path, countExpected(examples))
	fmt.Println("| Modelo | Precisión | Cobertura | F1 | Aciertos | Errores técnicos |")
	fmt.Println("|---|---|---|---|---|---|")

	var details strings.Builder
	for _, m := range models {
		m = strings.TrimSpace(m)
		c := filter.NewOpenAICompatClassifier(baseURL, key, m)
		c.DisableJSONMode = true
		c.DisableReasoning = strings.Contains(baseURL, "openrouter.ai")
		got, errs := classify(c, examples)

		var tp, fp, fn, tn int
		fmt.Fprintf(&details, "\n## %s — desacuerdos\n\n", m)
		for i, ex := range examples {
			if errs[i] {
				continue
			}
			switch {
			case got[i] && ex.Expected:
				tp++
			case got[i] && !ex.Expected:
				fp++
				fmt.Fprintf(&details, "- **Falso positivo** — %s (%s): %s%s\n", ex.Source, ex.Country, ex.Title, note(ex))
			case !got[i] && ex.Expected:
				fn++
				fmt.Fprintf(&details, "- **Falso negativo** — %s (%s): %s%s\n", ex.Source, ex.Country, ex.Title, note(ex))
			default:
				tn++
			}
		}
		nErr := 0
		for _, e := range errs {
			if e {
				nErr++
			}
		}
		precision := ratio(tp, tp+fp)
		recall := ratio(tp, tp+fn)
		f1 := 0.0
		if precision+recall > 0 {
			f1 = 2 * precision * recall / (precision + recall)
		}
		fmt.Printf("| `%s` | %.0f%% | %.0f%% | %.2f | %d/%d | %d |\n", m, precision*100, recall*100, f1, tp+tn, tp+tn+fp+fn, nErr)
	}
	fmt.Println(details.String())
}

func classify(c *filter.OpenAICompatClassifier, examples []example) ([]bool, []bool) {
	got := make([]bool, len(examples))
	errs := make([]bool, len(examples))
	for start := 0; start < len(examples); start += batchSize {
		end := min(start+batchSize, len(examples))
		items := make([]filter.BatchItem, 0, end-start)
		for _, ex := range examples[start:end] {
			items = append(items, filter.BatchItem{Article: model.Article{Title: ex.Title, Snippet: ex.Snippet}})
		}
		for i, r := range filter.ClassifyBatch(context.Background(), c, items) {
			if r.Err != nil {
				errs[start+i] = true
				continue
			}
			got[start+i] = r.Classification.IsInternational
		}
	}
	return got, errs
}

func load(path string) ([]example, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []example
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ex example
		if err := json.Unmarshal([]byte(line), &ex); err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, sc.Err()
}

func countExpected(examples []example) int {
	n := 0
	for _, ex := range examples {
		if ex.Expected {
			n++
		}
	}
	return n
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func note(ex example) string {
	if ex.Note == "" {
		return ""
	}
	return " _(" + ex.Note + ")_"
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
