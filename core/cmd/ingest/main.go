// Command ingest runs the daily pipeline: fetch top headlines from every
// configured source, prefilter for international/multinational potential,
// classify the survivors with Claude, and write the day's three-section
// Spanish report to disk.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"noticias/core/internal/usage"
)

const (
	// Bajado de 10 a 6: menos titulares candidatos por fuente significa
	// menos requests de clasificación en total (además del batching, ver
	// classifyBatchSize) — las noticias con potencial internacional real
	// suelen estar cerca del tope de cada feed, así que el recall que se
	// pierde en la cola es acotado. Configurable por si hace falta más
	// profundidad para alguna fuente en particular.
	defaultMaxHeadlinesPerSource = 6
	defaultFetchConcurrency      = 10
	defaultClassifyConcurrency   = 5
	// Algunos feeds devuelven notas de hace años (El País sirvió notas de
	// 2020 y Xinhua de 2017 el 2026-09-24): sin prefiltro, el clasificador
	// las aceptaba como noticias del día.
	defaultMaxArticleAgeHours = 72
	// Tope de gasto por corrida en modelos pagos (el usuario pidió no pasar
	// de USD 0,10 por día; queda margen para una regeneración).
	defaultMaxCostUSD = 0.08
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultFloat(key string, def float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return def
	}
	return v
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

// envOrDefaultList lee una lista separada por comas (espacios alrededor de
// cada item se descartan, items vacíos se saltean). Sin la env var, usa def
// como si fuera la única entrada.
func envOrDefaultList(key, def string) []string {
	raw := envOrDefault(key, def)
	var out []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
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
	// Modelos configurados, en orden (principal y después fallbacks), para
	// el registro de auditoría de la edición.
	var classifyModelNames, synthModelNames []string

	switch provider {
	case "claude":
		if apiKey == "" {
			log.Fatal("LLM_PROVIDER=claude pero ANTHROPIC_API_KEY no está configurada")
		}
		c := filter.NewClaudeClassifier(apiKey)
		s := report.NewClaudeSynthesizer(apiKey)
		classifier, synth = c, s
		classifyModelNames = []string{c.Model}
		synthModelNames = []string{s.Model}
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
		classifyModelNames = []string{classifyModel}
		synthModelNames = []string{synthModel}
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
		classifyModelNames = []string{classifyModel}
		synthModelNames = []string{synthModel}
		log.Printf("usando %s — clasificación: %s, síntesis: %s", baseURL, classifyModel, synthModel)
	case "ollama":
		ollamaHost := envOrDefault("OLLAMA_HOST", "http://localhost:11434")
		classifyModel := envOrDefault("OLLAMA_CLASSIFY_MODEL", "qwen3:1.7b")
		synthModel := envOrDefault("OLLAMA_SYNTHESIZE_MODEL", "qwen3.5:9b")
		classifier = filter.NewOllamaClassifier(ollamaHost, classifyModel)
		synth = report.NewOllamaSynthesizer(ollamaHost, synthModel)
		classifyConcurrency = 2 // modelo local: evita saturar CPU/GPU con muchos requests en paralelo
		classifyModelNames = []string{classifyModel}
		synthModelNames = []string{synthModel}
		log.Printf("usando Ollama local (%s) — clasificación: %s, síntesis: %s", ollamaHost, classifyModel, synthModel)
	default:
		log.Fatalf("LLM_PROVIDER desconocido: %q (usá claude, gemini, openai_compat u ollama)", provider)
	}

	// Fallback opcional: si el proveedor principal falla (cuota agotada,
	// caída), cae a una cadena de modelos free de OpenRouter en vez de
	// perder la corrida del día. Se activa solo si hay key — no reemplaza
	// al proveedor principal, se suma como red de contención. Varios
	// modelos en la cadena (no solo uno) porque un modelo free individual
	// puede fallar puntualmente ("Provider returned error", timeout) sin
	// que el proveedor entero esté caído — con más de un candidato, eso
	// no pierde el artículo.
	if openrouterKey := os.Getenv("OPENROUTER_API_KEY"); openrouterKey != "" {
		baseURL := envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
		// Lista separada por comas. Confirmá que sigan vigentes en
		// openrouter.ai/models (filtro "Free") antes de confiar en el
		// default — los ids de modelos free rotan (deepseek-chat-v3.1:free
		// dejó de ser free a mitad de esta implementación). Listas
		// distintas por paso: clasificar dispara muchas llamadas
		// concurrentes chicas, sintetizar es una sola llamada grande que
		// tolera un modelo más pesado.
		// openrouter/free al final de cada lista: es el router gratis de
		// OpenRouter, reparte entre varios modelos free por dentro en vez de
		// fijar uno solo — con solo 1-2 candidatos nombrados, si todos fallan
		// a la vez (cuota agotada, error puntual del proveedor) el
		// ChainClassifier/ChainSynthesizer los banca a TODOS para el resto de
		// la corrida (ver comentario en internal/filter/fallback.go) y esa
		// región/corrida se queda sin nada — nos pasó el 2026-09-23, Europa y
		// Asia Oriental quedaron en cero por esto. openrouter/free no vence
		// como id nombrado (a diferencia de un modelo puntual) porque decide
		// él mismo a qué modelo free enrutar en cada request.
		//
		// Actualización 2026-09-24: openrouter/free se sacó de los defaults.
		// Además de modelos de chat, enruta a modelos de moderación
		// (nemotron-3.5-content-safety), que contestan "User Safety: safe"
		// en vez de clasificar — ese día se perdieron batches enteros por
		// eso. En su lugar, varios modelos free nombrados y multilingües.
		classifyModels := envOrDefaultList("OPENROUTER_CLASSIFY_MODELS", "qwen/qwen3.8-27b:free,nvidia/nemotron-3-super-120b-a12b:free,google/gemma-4-31b-it:free")
		synthModels := envOrDefaultList("OPENROUTER_SYNTHESIZE_MODELS", "nvidia/nemotron-3-ultra-550b-a55b:free,qwen/qwen3.8-27b:free,nvidia/nemotron-3-super-120b-a12b:free")
		// OpenRouter recomienda estos headers para su free tier (ranking /
		// prioridad de cupo); no son estrictamente obligatorios.
		headers := map[string]string{
			"HTTP-Referer": envOrDefault("OPENROUTER_REFERER", "https://github.com/patriciobo/noticias-geopolitica"),
			"X-Title":      "noticias",
		}

		// Modelos de OpenRouter que van ANTES del proveedor principal (no
		// como fallback): pensado para un modelo pago barato y confiable
		// (ej. deepseek/deepseek-v4-flash, centavos por día) que no sufra
		// los 503 por demanda del free tier de Gemini. Vacío por defecto:
		// un modelo pago gasta créditos, se activa a propósito.
		primaryClassifyModels := envOrDefaultList("OPENROUTER_PRIMARY_CLASSIFY_MODELS", "")
		primarySynthModels := envOrDefaultList("OPENROUTER_PRIMARY_SYNTHESIZE_MODELS", "")

		newClassifier := func(m string) filter.Classifier {
			c := filter.NewOpenAICompatClassifier(baseURL, openrouterKey, m)
			c.ExtraHeaders = headers
			// Varios modelos free de OpenRouter no soportan response_format
			// y tiran error en vez de ignorarlo — el prompt ya pide JSON
			// puro por texto, alcanza sin el parámetro forzado.
			c.DisableJSONMode = true
			c.DisableReasoning = true
			if strings.HasSuffix(m, ":free") {
				c.MaxRetries = 2
			}
			return c
		}
		newSynth := func(m string) report.Synthesizer {
			s := report.NewOpenAICompatSynthesizer(baseURL, openrouterKey, m)
			s.ExtraHeaders = headers
			return s
		}

		var classifyChain []filter.Classifier
		var chainClassifyNames []string
		for _, m := range primaryClassifyModels {
			classifyChain = append(classifyChain, newClassifier(m))
			chainClassifyNames = append(chainClassifyNames, "openrouter:"+m)
		}
		classifyChain = append(classifyChain, classifier)
		chainClassifyNames = append(chainClassifyNames, classifyModelNames...)
		for _, m := range classifyModels {
			classifyChain = append(classifyChain, newClassifier(m))
			chainClassifyNames = append(chainClassifyNames, "openrouter:"+m)
		}
		classifier = filter.NewChainClassifier(classifyChain...)
		classifyModelNames = chainClassifyNames

		var synthChain []report.Synthesizer
		var chainSynthNames []string
		for _, m := range primarySynthModels {
			synthChain = append(synthChain, newSynth(m))
			chainSynthNames = append(chainSynthNames, "openrouter:"+m)
		}
		synthChain = append(synthChain, synth)
		chainSynthNames = append(chainSynthNames, synthModelNames...)
		for _, m := range synthModels {
			synthChain = append(synthChain, newSynth(m))
			chainSynthNames = append(chainSynthNames, "openrouter:"+m)
		}
		synth = report.NewChainSynthesizer(synthChain...)
		synthModelNames = chainSynthNames

		log.Printf("cadena de clasificación: %s", strings.Join(classifyModelNames, " → "))
		log.Printf("cadena de síntesis: %s", strings.Join(synthModelNames, " → "))
	}

	// CLASSIFY_CONCURRENCY pisa el default del proveedor: con DeepSeek pago
	// por OpenRouter no hace falta el límite bajo pensado para el free tier
	// de Gemini, y de a 2 lotes la clasificación tardaba ~12 minutos.
	classifyConcurrency = envOrDefaultInt("CLASSIFY_CONCURRENCY", classifyConcurrency)

	sources, err := ingest.LoadSources(sourcesPath)
	if err != nil {
		log.Fatalf("cargando sources: %v", err)
	}
	gaz, err := filter.LoadGazetteer(gazetteerPath)
	if err != nil {
		log.Fatalf("cargando gazetteer: %v", err)
	}

	log.Printf("fuentes cargadas: %d", len(sources))

	maxHeadlines := envOrDefaultInt("MAX_HEADLINES_PER_SOURCE", defaultMaxHeadlinesPerSource)
	articles, fetchErrs := fetchAll(sources, maxHeadlines, defaultFetchConcurrency)
	log.Printf("titulares obtenidos: %d", len(articles))

	maxAge := time.Duration(envOrDefaultInt("MAX_ARTICLE_AGE_HOURS", defaultMaxArticleAgeHours)) * time.Hour
	articles = dropStale(articles, time.Now(), maxAge)
	coverage := regionCoverage(sources, articles)
	sourceStatuses := buildSourceStatuses(sources, articles, fetchErrs)

	// El prefiltro por palabras clave queda apagado por defecto con
	// proveedores en la nube: no entiende titulares en coreano, japonés,
	// ruso, alemán, etc., y el 2026-09-24 descartó 406 de 465 titulares
	// (Europa y Asia Oriental quedaron vacías). Clasificar todo son ~30
	// requests por día en batches, holgado para los free tiers. Con Ollama
	// local sigue prendido: ahí cada titular cuesta CPU/GPU real.
	defaultPrefilter := "off"
	if provider == "ollama" {
		defaultPrefilter = "on"
	}
	usePrefilter := strings.ToLower(envOrDefault("PREFILTER", defaultPrefilter)) == "on"
	log.Printf("prefiltro por palabras clave: %v", usePrefilter)

	// Presupuesto por corrida: si clasificar y redactar ya lo consumieron,
	// las etapas opcionales (agrupar por hecho, chequear fidelidad) se
	// saltean en vez de pasarse. El costo real lo informa OpenRouter.
	budget := envOrDefaultFloat("MAX_COST_USD", defaultMaxCostUSD)

	classified, auditEntries := classifyAll(usage.WithStage(ctx, "clasificacion"), articles, sources, gaz, usePrefilter, classifier, classifyConcurrency)
	log.Printf("titulares con potencial internacional: %d", len(classified))

	if len(classified) == 0 {
		log.Fatal("ningún titular pasó el filtro internacional hoy — nada que sintetizar")
	}

	// Agrupar por hecho con el modelo: el agrupamiento por entidades
	// (países + tipo de relación) mezclaba historias distintas en la
	// cobertura cruzada. Si falla, se usa ese agrupamiento como antes.
	var groups []report.StoryGroup
	if comp, ok := synth.(report.Completer); ok && usage.Total() >= budget {
		log.Printf("agrupamiento por hecho salteado: ya se gastaron USD %.4f de USD %.2f", usage.Total(), budget)
	} else if ok {
		g, gerr := report.GroupStories(usage.WithoutReasoning(usage.WithStage(ctx, "agrupamiento")), comp, classified)
		if gerr != nil {
			log.Printf("agrupamiento por hecho falló, uso el de entidades: %v", gerr)
		} else {
			groups = g
			log.Printf("agrupamiento por hecho: %d historias con más de una nota", len(groups))
		}
	}

	// Razonamiento en esfuerzo bajo: sin razonamiento DeepSeek inventaba
	// nombres de región y bajaba la fidelidad; con el razonamiento por
	// defecto tardaba más de 5 minutos (2026-09-25).
	markdown, err := synth.Synthesize(usage.WithReasoningEffort(usage.WithStage(ctx, "redaccion"), "low"), report.Input{Items: classified, Coverage: coverage, Groups: groups})
	if err != nil {
		log.Fatalf("sintetizando reporte: %v", err)
	}

	// Defensa contra prompt injection: el informe solo puede enlazar notas
	// que efectivamente se procesaron (ver report.StripUnknownLinks).
	markdown = report.EnsureAllRegionsPresent(markdown, coverage)
	markdown, linksRemoved := report.StripUnknownLinks(markdown, classified)
	// Después de StripUnknownLinks: los links de las citas los arma el
	// código a partir de notas procesadas, no el modelo.
	markdown, citeStats := report.ResolveCitations(markdown, classified)
	log.Printf("citas: %d resueltas, %d inventadas (sacadas), %d bullets sin cita", citeStats.Resolved, citeStats.Invalid, citeStats.Uncited)

	// Chequeo de fidelidad: una segunda pasada contrasta cada afirmación
	// citada con sus notas. No bloquea la publicación — el resultado se
	// publica en la edición para que cualquiera vea qué quedó sin respaldo.
	var fidelity report.FidelityResult
	if comp, ok := synth.(report.Completer); ok && os.Getenv("FIDELITY_CHECK") != "off" && usage.Total() >= budget {
		log.Printf("chequeo de fidelidad salteado: ya se gastaron USD %.4f de USD %.2f", usage.Total(), budget)
	} else if ok && os.Getenv("FIDELITY_CHECK") != "off" {
		var ferr error
		fidelity, ferr = report.CheckFidelity(usage.WithoutReasoning(usage.WithStage(ctx, "fidelidad")), comp, markdown, classified)
		if ferr != nil {
			log.Printf("chequeo de fidelidad falló: %v", ferr)
		}
		log.Printf("fidelidad: %d chequeadas, %d respaldadas, %d inferencias, %d sin respaldo", fidelity.Checked, fidelity.Supported, fidelity.Inference, fidelity.Unsupported)
	}
	if linksRemoved > 0 {
		log.Printf("se sacaron %d enlace(s) del texto del LLM que no correspondían a notas procesadas", linksRemoved)
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

	auditPath := filepath.Join(outDir, dateStr+".audit.json")
	audit := buildAudit(provider, classifyModelNames, synthModelNames, auditEntries, sourceStatuses, linksRemoved)
	audit.SynthesizeModelUsed = report.ModelName(synth)
	audit.Counts.CitationsResolved = citeStats.Resolved
	audit.Counts.CitationsInvalid = citeStats.Invalid
	audit.Counts.UncitedBullets = citeStats.Uncited
	audit.Counts.ClaimsChecked = fidelity.Checked
	audit.Counts.ClaimsSupported = fidelity.Supported
	audit.Counts.ClaimsInference = fidelity.Inference
	audit.Counts.ClaimsUnsupported = fidelity.Unsupported
	audit.FidelityIssues = fidelity.Issues
	audit.CostUSD = usage.Total()
	audit.CostByStage = map[string]model.StageCost{}
	for _, st := range usage.Stages() {
		t := usage.ByStage()[st]
		audit.CostByStage[st] = model.StageCost(t)
		log.Printf("costo %s: USD %.4f (%d pedidos, %d tokens de entrada, %d de salida)", st, t.CostUSD, t.Requests, t.PromptTokens, t.CompletionTokens)
	}
	log.Printf("costo total de la corrida: USD %.4f (presupuesto USD %.2f)", audit.CostUSD, budget)
	audit.Version, audit.Revisions = nextRevision(auditPath, audit.GeneratedAt, os.Getenv("REGENERATION_REASON"))
	if audit.Version > 1 {
		log.Printf("edición regenerada: versión %d (motivo: %s)", audit.Version, audit.Revisions[len(audit.Revisions)-1].Reason)
	}
	auditJSON, err := json.MarshalIndent(audit, "", " ")
	if err != nil {
		log.Fatalf("serializando registro de auditoría: %v", err)
	}
	if err := os.WriteFile(auditPath, auditJSON, 0o644); err != nil {
		log.Fatalf("escribiendo registro de auditoría: %v", err)
	}
	log.Printf("registro de auditoría: %+v (%s)", audit.Counts, auditPath)
}

// buildAudit arma el registro público de la edición (ver model.Audit).
// Commit y RunURL salen de las variables que GitHub Actions define en cada
// corrida; en una corrida local quedan vacíos.
func buildAudit(provider string, classifyModels, synthModels []string, entries []model.AuditEntry, sources []model.SourceStatus, linksRemoved int) model.Audit {
	prompts := map[string]string{}
	for name, text := range filter.SystemPrompts() {
		prompts[name] = sha256Hex(text)
	}
	for name, text := range report.SystemPrompts() {
		prompts[name] = sha256Hex(text)
	}

	var runURL string
	if server, repo, run := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID"); server != "" && repo != "" && run != "" {
		runURL = server + "/" + repo + "/actions/runs/" + run
	}

	// Orden estable (fuente, título) para que el archivo sea legible y los
	// diffs entre corridas tengan sentido.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].Title < entries[j].Title
	})

	counts := model.AuditCounts{Fetched: len(entries), LinksRemoved: linksRemoved, SourcesConfigured: len(sources)}
	used := map[string]int{}
	for _, e := range entries {
		if e.Model != "" {
			used[e.Model]++
		}
	}
	var problems []model.SourceStatus
	for _, s := range sources {
		if s.Status == model.SourceOK {
			counts.SourcesResponded++
		} else {
			problems = append(problems, s)
		}
	}
	for _, e := range entries {
		switch e.Stage {
		case model.StagePrefilterRejected:
			counts.PrefilterRejected++
		case model.StageClassifierRejected:
			counts.ClassifierRejected++
		case model.StageClassifierError:
			counts.ClassifierErrors++
		case model.StageAccepted:
			counts.Accepted++
		}
	}

	return model.Audit{
		Provenance: model.Provenance{
			GeneratedAt:        time.Now().UTC(),
			Commit:             os.Getenv("GITHUB_SHA"),
			RunURL:             runURL,
			Provider:           provider,
			ClassifyModels:     classifyModels,
			SynthesizeModels:   synthModels,
			PromptSHA256:       prompts,
			Counts:             counts,
			SourceProblems:     problems,
			ClassifyModelsUsed: used,
		},
		Sources:   sources,
		Headlines: entries,
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
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

func fetchAll(sources []model.Source, maxItems, concurrency int) ([]struct {
	Source  model.Source
	Article model.Article
}, map[string]string) {
	type result = struct {
		Source  model.Source
		Article model.Article
	}
	errs := map[string]string{}
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
				mu.Lock()
				errs[src.ID] = err.Error()
				mu.Unlock()
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
	return out, errs
}

// classifyBatchSize agrupa varios titulares por request en vez de uno por
// request: mismo trabajo pedido al modelo, muchas menos llamadas — lo que
// de verdad pisa los límites por minuto de los free tiers es la CANTIDAD de
// requests, no el volumen total del día (ver filter.ClassifyBatch). Con
// esto, `concurrency` pasa a limitar cuántos BATCHES corren en paralelo, no
// cuántos artículos — el número real de requests concurrentes es el mismo
// de antes o menor, para el mismo valor de concurrency.
//
// 16 (antes 12) desde que se clasifica todo sin prefiltro: ~460 titulares
// quedan en ~29 requests por día, lejos del tope diario de los free tiers
// de OpenRouter aun si Gemini se cae entero.
const classifyBatchSize = 16

func classifyAll(
	ctx context.Context,
	items []struct {
		Source  model.Source
		Article model.Article
	},
	_ []model.Source,
	gaz *filter.Gazetteer,
	usePrefilter bool,
	classifier filter.Classifier,
	concurrency int,
) ([]model.ClassifiedArticle, []model.AuditEntry) {
	type passedItem struct {
		Source  model.Source
		Article model.Article
		Pre     filter.PrefilterResult
	}
	entry := func(src model.Source, a model.Article, stage string) model.AuditEntry {
		return model.AuditEntry{Source: src.Name, Country: src.Country, Title: a.Title, Link: a.Link, Stage: stage}
	}

	var (
		passed []passedItem
		audit  = make([]model.AuditEntry, 0, len(items))
	)
	for _, it := range items {
		pre := filter.Prefilter(it.Article, gaz)
		if pre.Passed || !usePrefilter {
			passed = append(passed, passedItem{Source: it.Source, Article: it.Article, Pre: pre})
			continue
		}
		e := entry(it.Source, it.Article, model.StagePrefilterRejected)
		e.Reason = "no menciona dos o más países, una empresa multinacional de la lista ni una palabra clave de comercio/tratados"
		audit = append(audit, e)
	}

	var (
		mu  sync.Mutex
		out []model.ClassifiedArticle
		wg  sync.WaitGroup
		sem = make(chan struct{}, concurrency)
	)

	for start := 0; start < len(passed); start += classifyBatchSize {
		end := min(start+classifyBatchSize, len(passed))
		chunk := passed[start:end]

		wg.Add(1)
		go func(chunk []passedItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			batchItems := make([]filter.BatchItem, len(chunk))
			for i, it := range chunk {
				batchItems[i] = filter.BatchItem{Article: it.Article, Pre: it.Pre}
			}
			results := filter.ClassifyBatch(ctx, classifier, batchItems)

			mu.Lock()
			for i, r := range results {
				if r.Err != nil {
					log.Printf("classify %q: %v", chunk[i].Article.Title, r.Err)
					e := entry(chunk[i].Source, chunk[i].Article, model.StageClassifierError)
					e.Reason = r.Err.Error()
					audit = append(audit, e)
					continue
				}
				stage := model.StageClassifierRejected
				if r.Classification.IsInternational {
					stage = model.StageAccepted
					out = append(out, model.ClassifiedArticle{Article: chunk[i].Article, Source: chunk[i].Source, Classification: r.Classification})
				}
				e := entry(chunk[i].Source, chunk[i].Article, stage)
				e.Model = r.Model
				e.Reason = r.Classification.Reason
				e.RelationType = r.Classification.RelationType
				e.Confidence = r.Classification.Confidence
				audit = append(audit, e)
			}
			mu.Unlock()
		}(chunk)
	}
	wg.Wait()
	return out, audit
}

// dropStale saca los titulares con fecha de publicación más vieja que
// maxAge. Los que no traen fecha (PublishedAt cero) se quedan: muchos
// feeds no la informan y no hay forma de saber si son viejos.
func dropStale(items []struct {
	Source  model.Source
	Article model.Article
}, now time.Time, maxAge time.Duration) []struct {
	Source  model.Source
	Article model.Article
} {
	out := items[:0:0]
	perSource := map[string]int{}
	for _, it := range items {
		if pub := it.Article.PublishedAt; !pub.IsZero() && now.Sub(pub) > maxAge {
			perSource[it.Source.Name]++
			continue
		}
		out = append(out, it)
	}
	for name, n := range perSource {
		log.Printf("descartados %d titulares viejos (más de %s) de %s — revisar el feed", n, maxAge, name)
	}
	return out
}

// hasFeed indica si el medio tiene un feed configurado (los NO_RSS no se
// consultan y no cuentan para la cobertura).
func hasFeed(src model.Source) bool {
	return src.RSS != "" && src.RSS != "NO_RSS"
}

// regionCoverage cuenta, por región, los medios con feed configurados y
// cuántos aportaron al menos un titular vigente (después de dropStale).
func regionCoverage(sources []model.Source, items []struct {
	Source  model.Source
	Article model.Article
}) map[string]report.RegionCoverage {
	responded := map[string]bool{}
	for _, it := range items {
		responded[it.Source.ID] = true
	}
	cov := map[string]report.RegionCoverage{}
	for _, src := range sources {
		if !hasFeed(src) {
			continue
		}
		c := cov[src.Region]
		c.Configured++
		if responded[src.ID] {
			c.Responded++
		}
		cov[src.Region] = c
	}
	return cov
}

// buildSourceStatuses arma el estado de cada medio con feed en esta
// corrida, para el registro de auditoría: ok, error (no se pudo descargar)
// o sin_vigentes (respondió, pero sin titulares de las últimas horas).
func buildSourceStatuses(sources []model.Source, items []struct {
	Source  model.Source
	Article model.Article
}, fetchErrs map[string]string) []model.SourceStatus {
	count := map[string]int{}
	for _, it := range items {
		count[it.Source.ID]++
	}
	var out []model.SourceStatus
	for _, src := range sources {
		if !hasFeed(src) {
			continue
		}
		st := model.SourceStatus{Name: src.Name, Country: src.Country, Region: src.Region, Headlines: count[src.ID]}
		switch {
		case fetchErrs[src.ID] != "":
			st.Status = model.SourceError
			st.Detail = fetchErrs[src.ID]
		case count[src.ID] == 0:
			st.Status = model.SourceNoRecent
		default:
			st.Status = model.SourceOK
		}
		out = append(out, st)
	}
	return out
}

// nextRevision lee el registro de una edición anterior de la misma fecha
// (si existe) y devuelve la versión nueva con el historial acumulado. Sin
// motivo declarado en una regeneración, lo dice explícitamente.
func nextRevision(auditPath string, generatedAt time.Time, reason string) (int, []model.Revision) {
	var prev model.Provenance
	if raw, err := os.ReadFile(auditPath); err == nil {
		_ = json.Unmarshal(raw, &prev)
	}
	revisions := prev.Revisions
	if len(revisions) == 0 && !prev.GeneratedAt.IsZero() {
		// Edición anterior al registro de versiones: cuenta como la 1.
		revisions = []model.Revision{{Version: 1, GeneratedAt: prev.GeneratedAt, Reason: "edición original"}}
	}
	version := len(revisions) + 1
	switch {
	case version == 1:
		reason = "edición original"
	case reason == "":
		reason = "regenerada sin motivo declarado"
	}
	revisions = append(revisions, model.Revision{Version: version, GeneratedAt: generatedAt, Reason: reason})
	return version, revisions
}
