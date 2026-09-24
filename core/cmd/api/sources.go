package main

import (
	"log"
	"net/http"
	"os"

	"noticias/core/internal/ingest"
	"noticias/core/internal/model"
)

// publicSource es lo que se publica de cada medio configurado: suficiente
// para que cualquiera audite el balance de la selección (países, regiones,
// orientación editorial), sin exponer detalles operativos como la URL del
// feed.
type publicSource struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	Region  string `json:"region"`
	Stance  string `json:"stance"`
	// Quién controla el medio (estatal, privado, partidario, ong, exilio) y
	// una aclaración opcional — ver config/sources.yaml.
	Ownership     string `json:"ownership"`
	OwnershipNote string `json:"ownership_note,omitempty"`
	Homepage      string `json:"homepage"`
	HasFeed       bool   `json:"has_feed"` // false = configurado pero sin feed utilizable hoy (no se consulta)
}

// resolveSourcesPath busca config/sources.yaml: SOURCES_PATH si está
// seteada, si no las ubicaciones habituales según desde dónde arranque el
// proceso (raíz del repo en Render, core/ en local).
func resolveSourcesPath() string {
	if p := os.Getenv("SOURCES_PATH"); p != "" {
		return p
	}
	for _, p := range []string{"./config/sources.yaml", "../config/sources.yaml"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func toPublicSources(sources []model.Source) []publicSource {
	out := make([]publicSource, 0, len(sources))
	for _, s := range sources {
		out = append(out, publicSource{
			Name:          s.Name,
			Country:       s.Country,
			Region:        s.Region,
			Stance:        s.Stance,
			Ownership:     s.Ownership,
			OwnershipNote: s.OwnershipNote,
			Homepage:      s.Homepage,
			HasFeed:       s.RSS != "" && s.RSS != "NO_RSS",
		})
	}
	return out
}

// sourcesHandler sirve la lista pública de medios. Se lee una sola vez al
// arrancar: la lista cambia con un commit, y cada commit redeploya la API.
func sourcesHandler() http.HandlerFunc {
	path := resolveSourcesPath()
	var list []publicSource
	if path == "" {
		log.Print("no se encontró config/sources.yaml (seteá SOURCES_PATH): /sources responde 404")
	} else if sources, err := ingest.LoadSources(path); err != nil {
		log.Printf("no se pudo leer %s: %v — /sources responde 404", path, err)
	} else {
		list = toPublicSources(sources)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if list == nil {
			http.Error(w, "lista de medios no disponible", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}
