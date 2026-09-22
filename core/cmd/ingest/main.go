// Command ingest runs the daily pipeline: fetch top headlines from every
// configured source, prefilter for international/multinational potential,
// classify the survivors with Claude, and write the day's three-section
// Spanish report to disk.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"noticias/core/internal/filter"
	"noticias/core/internal/ingest"
	"noticias/core/internal/model"
	"noticias/core/internal/report"
)

const (
	defaultMaxHeadlinesPerSource = 10
	defaultFetchConcurrency      = 10
	defaultClassifyConcurrency   = 5
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// loadDotEnv reads KEY=VALUE pairs from an optional .env file and applies
// them via os.Setenv, without overriding variables already set in the real
// environment. Missing file is not an error — .env is just a convenience,
// export ANTHROPIC_API_KEY=... in the shell works exactly the same.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func main() {
	loadDotEnv(".env")

	ctx := context.Background()

	sourcesPath := envOrDefault("SOURCES_PATH", "../config/sources.yaml")
	gazetteerPath := envOrDefault("GAZETTEER_PATH", "../config/gazetteer.yaml")
	outDir := envOrDefault("OUT_DIR", "./out")
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")
	compatKey := os.Getenv("OPENAI_COMPAT_API_KEY")

	// LLM_PROVIDER=claude|gemini|openai_compat|ollama. Sin configurar, se
	// elige solo por la primera key que encuentre, en este orden, y si no
	// hay ninguna cae a Ollama local (gratis, sin key, requiere
	// `ollama serve` corriendo y los modelos ya bajados).
	provider := strings.ToLower(envOrDefault("LLM_PROVIDER", ""))
	if provider == "" {
		switch {
		case apiKey != "":
			provider = "claude"
		case geminiKey != "":
			provider = "gemini"
		case compatKey != "":
			provider = "openai_compat"
		default:
			provider = "ollama"
		}
	}

	var classifier filter.Classifier
	var synth report.Synthesizer
	classifyConcurrency := defaultClassifyConcurrency

	switch provider {
	case "claude":
		if apiKey == "" {
			log.Fatal("LLM_PROVIDER=claude pero ANTHROPIC_API_KEY no está configurada")
		}
		classifier = filter.NewClaudeClassifier(apiKey)
		synth = report.NewClaudeSynthesizer(apiKey)
	case "gemini":
		// Atajo sobre el cliente compatible con OpenAI: mismo cliente que
		// "openai_compat", pero con los defaults de Gemini precargados para
		// que alcance con GEMINI_API_KEY. Free tier: aistudio.google.com.
		if geminiKey == "" {
			log.Fatal("LLM_PROVIDER=gemini pero GEMINI_API_KEY no está configurada")
		}
		baseURL := envOrDefault("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai/")
		classifyModel := envOrDefault("GEMINI_CLASSIFY_MODEL", "gemini-3.5-flash-lite")
		synthModel := envOrDefault("GEMINI_SYNTHESIZE_MODEL", "gemini-3.5-flash-lite")
		classifier = filter.NewOpenAICompatClassifier(baseURL, geminiKey, classifyModel)
		synth = report.NewOpenAICompatSynthesizer(baseURL, geminiKey, synthModel)
		// Free tier: RPM bajo, se pisa enseguida con concurrencia alta.
		// El cliente ya reintenta con backoff ante 429, pero conviene no
		// generarlos de entrada.
		classifyConcurrency = envOrDefaultInt("GEMINI_CLASSIFY_CONCURRENCY", 2)
		log.Printf("usando Gemini — clasificación: %s, síntesis: %s", classifyModel, synthModel)
	case "openai_compat":
		if compatKey == "" {
			log.Fatal("LLM_PROVIDER=openai_compat pero OPENAI_COMPAT_API_KEY no está configurada")
		}
		baseURL := envOrDefault("OPENAI_COMPAT_BASE_URL", "https://api.openai.com/v1")
		classifyModel := envOrDefault("OPENAI_COMPAT_CLASSIFY_MODEL", "gpt-5-nano")
		synthModel := envOrDefault("OPENAI_COMPAT_SYNTHESIZE_MODEL", "gpt-5-nano")
		classifier = filter.NewOpenAICompatClassifier(baseURL, compatKey, classifyModel)
		synth = report.NewOpenAICompatSynthesizer(baseURL, compatKey, synthModel)
		log.Printf("usando %s — clasificación: %s, síntesis: %s", baseURL, classifyModel, synthModel)
	case "ollama":
		ollamaHost := envOrDefault("OLLAMA_HOST", "http://localhost:11434")
		classifyModel := envOrDefault("OLLAMA_CLASSIFY_MODEL", "qwen3:1.7b")
		synthModel := envOrDefault("OLLAMA_SYNTHESIZE_MODEL", "qwen3.5:9b")
		classifier = filter.NewOllamaClassifier(ollamaHost, classifyModel)
		synth = report.NewOllamaSynthesizer(ollamaHost, synthModel)
		classifyConcurrency = 2 // modelo local: evita saturar CPU/GPU con muchos requests en paralelo
		log.Printf("usando Ollama local (%s) — clasificación: %s, síntesis: %s", ollamaHost, classifyModel, synthModel)
	default:
		log.Fatalf("LLM_PROVIDER desconocido: %q (usá claude, gemini, openai_compat u ollama)", provider)
	}

	// Fallback opcional: si el proveedor principal falla (cuota agotada,
	// caída), cae a un modelo free de OpenRouter en vez de perder la
	// corrida del día. Se activa solo si hay key — no reemplaza al
	// proveedor principal, se suma como red de contención.
	if openrouterKey := os.Getenv("OPENROUTER_API_KEY"); openrouterKey != "" {
		baseURL := envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
		// Confirmá que sigan vigentes en openrouter.ai/models (filtro "Free")
		// antes de confiar en el default — los ids de modelos free rotan
		// (el anterior, deepseek/deepseek-chat-v3.1:free, dejó de ser free).
		// Modelos distintos para cada paso: así no comparten el mismo
		// límite de tasa del free tier entre clasificar (muchas llamadas
		// concurrentes chicas) y sintetizar (una sola llamada grande).
		classifyModel := envOrDefault("OPENROUTER_CLASSIFY_MODEL", "poolside/laguna-s-2.1:free")
		synthModel := envOrDefault("OPENROUTER_SYNTHESIZE_MODEL", "nvidia/nemotron-3-ultra-550b-a55b:free")
		// OpenRouter recomienda estos headers para su free tier (ranking /
		// prioridad de cupo); no son estrictamente obligatorios.
		headers := map[string]string{
			"HTTP-Referer": envOrDefault("OPENROUTER_REFERER", "https://github.com/patriciobo/noticias-geopolitica"),
			"X-Title":      "noticias",
		}

		orClassifier := filter.NewOpenAICompatClassifier(baseURL, openrouterKey, classifyModel)
		orClassifier.ExtraHeaders = headers
		orSynth := report.NewOpenAICompatSynthesizer(baseURL, openrouterKey, synthModel)
		orSynth.ExtraHeaders = headers

		classifier = &filter.FallbackClassifier{Primary: classifier, Secondary: orClassifier}
		synth = &report.FallbackSynthesizer{Primary: synth, Secondary: orSynth}
		log.Printf("fallback OpenRouter configurado — clasificación: %s, síntesis: %s", classifyModel, synthModel)
	}

	sources, err := ingest.LoadSources(sourcesPath)
	if err != nil {
		log.Fatalf("cargando sources: %v", err)
	}
	gaz, err := filter.LoadGazetteer(gazetteerPath)
	if err != nil {
		log.Fatalf("cargando gazetteer: %v", err)
	}

	log.Printf("fuentes cargadas: %d", len(sources))

	articles := fetchAll(sources, defaultMaxHeadlinesPerSource, defaultFetchConcurrency)
	log.Printf("titulares obtenidos: %d", len(articles))

	classified := classifyAll(ctx, articles, sources, gaz, classifier, classifyConcurrency)
	log.Printf("titulares con potencial internacional: %d", len(classified))

	if len(classified) == 0 {
		log.Fatal("ningún titular pasó el filtro internacional hoy — nada que sintetizar")
	}

	markdown, err := synth.Synthesize(ctx, classified)
	if err != nil {
		log.Fatalf("sintetizando reporte: %v", err)
	}

	markdown = report.AppendSources(markdown, classified)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("creando out dir: %v", err)
	}
	dateStr := time.Now().Format("2006-01-02")
	outPath := filepath.Join(outDir, dateStr+".md")
	if err := os.WriteFile(outPath, []byte(markdown), 0o644); err != nil {
		log.Fatalf("escribiendo reporte: %v", err)
	}
	log.Printf("reporte escrito en %s", outPath)

	consulted := distinctSources(articles)
	sourcesPath2 := filepath.Join(outDir, dateStr+".sources.json")
	sourcesJSON, err := json.Marshal(consulted)
	if err != nil {
		log.Fatalf("serializando medios consultados: %v", err)
	}
	if err := os.WriteFile(sourcesPath2, sourcesJSON, 0o644); err != nil {
		log.Fatalf("escribiendo medios consultados: %v", err)
	}
	log.Printf("medios consultados: %d (%s)", len(consulted), sourcesPath2)
}

// distinctSources dedupes the sources that returned at least one headline
// today — this is "medios consultados" as actually reached, not just the
// configured list (a source can be temporarily down without being NO_RSS).
func distinctSources(items []struct {
	Source  model.Source
	Article model.Article
}) []model.SourceSummary {
	seen := map[string]bool{}
	var out []model.SourceSummary
	for _, it := range items {
		if seen[it.Source.ID] {
			continue
		}
		seen[it.Source.ID] = true
		out = append(out, model.SourceSummary{Name: it.Source.Name, Country: it.Source.Country})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func fetchAll(sources []model.Source, maxItems, concurrency int) []struct {
	Source  model.Source
	Article model.Article
} {
	type result = struct {
		Source  model.Source
		Article model.Article
	}
	var (
		mu  sync.Mutex
		out []result
		wg  sync.WaitGroup
		sem = make(chan struct{}, concurrency)
	)

	for _, src := range sources {
		if src.RSS == "" || src.RSS == "NO_RSS" {
			continue
		}
		wg.Add(1)
		go func(src model.Source) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			arts, err := ingest.FetchHeadlines(src, maxItems)
			if err != nil {
				log.Printf("fetch %s: %v", src.ID, err)
				return
			}
			mu.Lock()
			for _, a := range arts {
				out = append(out, result{Source: src, Article: a})
			}
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	return out
}

func classifyAll(
	ctx context.Context,
	items []struct {
		Source  model.Source
		Article model.Article
	},
	_ []model.Source,
	gaz *filter.Gazetteer,
	classifier filter.Classifier,
	concurrency int,
) []model.ClassifiedArticle {
	var (
		mu  sync.Mutex
		out []model.ClassifiedArticle
		wg  sync.WaitGroup
		sem = make(chan struct{}, concurrency)
	)

	for _, it := range items {
		pre := filter.Prefilter(it.Article, gaz)
		if !pre.Passed {
			continue
		}

		wg.Add(1)
		go func(src model.Source, a model.Article, pre filter.PrefilterResult) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cls, err := classifier.Classify(ctx, a, pre)
			if err != nil {
				log.Printf("classify %q: %v", a.Title, err)
				return
			}
			if !cls.IsInternational {
				return
			}
			mu.Lock()
			out = append(out, model.ClassifiedArticle{Article: a, Source: src, Classification: cls})
			mu.Unlock()
		}(it.Source, it.Article, pre)
	}
	wg.Wait()
	return out
}
