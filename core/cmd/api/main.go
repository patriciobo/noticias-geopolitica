// Command api serves the reports written by cmd/ingest over HTTP, so the
// Next.js blog (and later, a mobile app) can consume them without touching
// the filesystem directly. v1 is file-backed on purpose — no DB config
// needed until report history/querying outgrows a directory of Markdown
// files.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"noticias/core/internal/model"
)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type reportResponse struct {
	Date        string                `json:"date"`
	Markdown    string                `json:"markdown"`
	Sources     []model.SourceSummary `json:"sources"`
	SourceCount int                   `json:"source_count"`
}

func main() {
	outDir := envOrDefault("OUT_DIR", "./out")
	addr := envOrDefault("API_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /reports", listReportsHandler(outDir))
	mux.HandleFunc("GET /reports/latest", latestReportHandler(outDir))
	mux.HandleFunc("GET /reports/{date}", reportByDateHandler(outDir))

	log.Printf("api escuchando en %s (fuente: %s)", addr, outDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func listDates(outDir string) ([]string, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var dates []string
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
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
