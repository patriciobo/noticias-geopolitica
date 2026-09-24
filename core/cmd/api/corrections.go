package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"noticias/core/internal/model"
)

// correction es una edición que se generó más de una vez, con su historial.
type correction struct {
	Date      string           `json:"date"`
	Revisions []model.Revision `json:"revisions"`
}

// correctionsHandler lista las ediciones regeneradas (versión > 1), de la
// más nueva a la más vieja, leyendo los registros de auditoría. Es la base
// de /correcciones en la web: ninguna regeneración queda escondida.
func correctionsHandler(outDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paths, _ := filepath.Glob(filepath.Join(outDir, "*.audit.json"))
		out := []correction{}
		for _, p := range paths {
			raw, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var prov model.Provenance
			if json.Unmarshal(raw, &prov) != nil || len(prov.Revisions) < 2 {
				continue
			}
			date := strings.TrimSuffix(filepath.Base(p), ".audit.json")
			out = append(out, correction{Date: date, Revisions: prov.Revisions})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
		writeJSON(w, http.StatusOK, out)
	}
}
