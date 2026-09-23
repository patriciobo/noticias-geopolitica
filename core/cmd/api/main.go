// Command api serves the reports written by cmd/ingest over HTTP, so the
// Next.js blog (and later, a mobile app) can consume them without touching
// the filesystem directly. Reports stay file-backed on purpose — no DB
// needed until report history/querying outgrows a directory of Markdown
// files. DB config (DATABASE_URL) is only used for the newsletter
// subscriber list, and is entirely optional.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"noticias/core/internal/config"
	"noticias/core/internal/model"
	"noticias/core/internal/newsletter"
	"noticias/core/internal/store"
	"noticias/core/internal/subscriber"
)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type reportResponse struct {
	Date        string                `json:"date"`
	Markdown    string                `json:"markdown"`
	Sources     []model.SourceSummary `json:"sources"`
	SourceCount int                   `json:"source_count"`
}

func main() {
	config.LoadDotEnv(".env")

	outDir := config.EnvOrDefault("OUT_DIR", "./out")
	addr := config.EnvOrDefault("API_ADDR", ":8080")

	// Límite general por IP para todo: holgado para un humano o para el
	// blog (que cachea), corta a un scraper/bot que martilla la API.
	perMinute := config.EnvInt("RATE_LIMIT_PER_MINUTE", 120)
	general := newRateLimiter(perMinute, time.Minute, perMinute)
	general.startSweeper(10 * time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /reports", listReportsHandler(outDir))
	mux.HandleFunc("GET /reports/latest", latestReportHandler(outDir))
	mux.HandleFunc("GET /reports/{date}", reportByDateHandler(outDir))

	// El newsletter es opcional: sin DATABASE_URL, cmd/api sigue sirviendo
	// /reports* igual que siempre — no hace falta DB para lo demás.
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		db, err := store.Open(ctx, dsn)
		cancel()
		if err != nil {
			log.Fatalf("no se pudo conectar a DATABASE_URL: %v", err)
		}

		subs := &subscriptions{
			repo: subscriber.NewRepository(db),
			brevo: newsletter.NewBrevo(
				os.Getenv("BREVO_API_KEY"),
				os.Getenv("BREVO_SENDER_EMAIL"),
				config.EnvOrDefault("BREVO_SENDER_NAME", "Radar Global"),
			),
			apiBaseURL:     config.EnvOrDefault("API_BASE_URL", "http://localhost"+addr),
			siteURL:        config.EnvOrDefault("SITE_URL", "http://localhost:3000"),
			internalSecret: os.Getenv("INTERNAL_API_SECRET"),
			enabled:        config.EnvOrDefault("SUBSCRIPTIONS_ENABLED", "true") != "false",
			dailyCap:       config.EnvInt("CONFIRM_EMAILS_PER_DAY", 100),
			resendAfter:    time.Duration(config.EnvInt("CONFIRM_RESEND_AFTER_MINUTES", 60)) * time.Minute,
		}
		if subs.brevo.APIKey == "" || subs.brevo.SenderEmail == "" {
			// Sin Brevo no hay forma de mandar el mail de confirmación, y
			// sin confirmación nadie queda activo: mejor cortar las altas
			// que acumular pendientes que nunca se van a poder confirmar.
			subs.enabled = false
			log.Print("BREVO_API_KEY/BREVO_SENDER_EMAIL no configurados: altas nuevas deshabilitadas")
		}
		if subs.internalSecret == "" {
			log.Print("ADVERTENCIA: INTERNAL_API_SECRET vacío — POST /subscribers acepta pedidos de cualquier origen")
		}

		// Altas: pocas por IP (un humano se anota una vez) además del
		// límite general.
		perHour := config.EnvInt("SUBSCRIBE_PER_HOUR", 5)
		signups := newRateLimiter(perHour, time.Hour, perHour)
		signups.startSweeper(10 * time.Minute)

		// Sin CORS: /subscribers lo llama web/'s /api/subscribe server-side
		// (mismo-origen para el browser, server-to-server hacia acá), y
		// confirm/unsubscribe los abre el navegador como link o form normal
		// (no fetch/XHR), ninguno de los dos está sujeto a CORS.
		mux.Handle("POST /subscribers", signups.limit(subs.subscribeKey, http.HandlerFunc(subs.handleSubscribe)))
		mux.HandleFunc("GET /subscribers/confirm", subs.handleConfirmPage)
		mux.HandleFunc("POST /subscribers/confirm", subs.handleConfirm)
		mux.HandleFunc("GET /subscribers/unsubscribe", subs.handleUnsubscribePage)
		mux.HandleFunc("POST /subscribers/unsubscribe", subs.handleUnsubscribe)
		log.Printf("newsletter habilitado (altas nuevas: %v)", subs.enabled)
	} else {
		log.Print("newsletter deshabilitado (sin DATABASE_URL)")
	}

	// Timeouts explícitos: http.ListenAndServe no pone ninguno, y un
	// cliente que abre conexiones y manda headers de a un byte (slowloris)
	// puede agotar los recursos de una instancia chica.
	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(general.limit(clientIP, mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	log.Printf("api escuchando en %s (fuente: %s)", addr, outDir)
	log.Fatal(srv.ListenAndServe())
}

// securityHeaders agrega headers defensivos a toda respuesta. La API sirve
// JSON y unas pocas páginas HTML mínimas (confirmación/baja): no carga
// scripts ni recursos externos, así que la CSP puede ser estricta.
// Referrer-Policy no-referrer importa: las URLs de confirmación y baja
// llevan tokens, y no deben filtrarse a otro sitio vía Referer.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func listDates(outDir string) ([]string, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	dates := []string{} // no nil: sin reportes debe serializar como [], no null
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		if name != e.Name() && dateRe.MatchString(name) {
			dates = append(dates, name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates, nil
}

func listReportsHandler(outDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dates, err := listDates(outDir)
		if err != nil {
			http.Error(w, "no se pudo listar reportes", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, dates)
	}
}

func latestReportHandler(outDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dates, err := listDates(outDir)
		if err != nil || len(dates) == 0 {
			http.Error(w, "no hay reportes todavía", http.StatusNotFound)
			return
		}
		serveReport(w, outDir, dates[0])
	}
}

func reportByDateHandler(outDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.PathValue("date")
		if !dateRe.MatchString(date) {
			http.Error(w, "fecha inválida, formato YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		serveReport(w, outDir, date)
	}
}

func serveReport(w http.ResponseWriter, outDir, date string) {
	path := filepath.Join(outDir, date+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "reporte no encontrado", http.StatusNotFound)
		return
	}

	// El sidecar de medios consultados es nuevo — reportes viejos pueden no
	// tenerlo, en ese caso queda una lista vacía en vez de romper.
	var sources []model.SourceSummary
	sourcesPath := filepath.Join(outDir, date+".sources.json")
	if raw, err := os.ReadFile(sourcesPath); err == nil {
		_ = json.Unmarshal(raw, &sources)
	}

	writeJSON(w, http.StatusOK, reportResponse{
		Date:        date,
		Markdown:    string(raw),
		Sources:     sources,
		SourceCount: len(sources),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	// Los reportes cambian una vez por día: 5 minutos de cache en CDN/
	// cliente sacan casi toda la carga de encima sin demorar la edición
	// nueva de forma notoria.
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
