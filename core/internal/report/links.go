package report

import (
	"regexp"
	"strings"

	"noticias/core/internal/model"
)

// linkPattern reconoce, en orden de prioridad (regexp toma la coincidencia
// que empieza primero): links e imágenes markdown, autolinks <https://...> y
// URLs sueltas.
var linkPattern = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)\s]*)(?:\s+"[^"]*")?\)|<(https?://[^>\s]+)>|https?://[^\s<>()\[\]]+`)

// StripUnknownLinks saca del texto del LLM todo enlace o URL que no sea el
// de una nota que efectivamente entró a la síntesis. El prompt ya le pide
// al modelo no escribir enlaces, pero esto no depende de que obedezca: un
// titular manipulado (prompt injection) no puede colar un link a un sitio
// ajeno en el informe. Un link markdown desconocido queda como su texto;
// una imagen o URL suelta desconocida se elimina. Devuelve también cuántos
// se sacaron, para loguearlo.
func StripUnknownLinks(markdown string, items []model.ClassifiedArticle) (string, int) {
	allowed := make(map[string]bool, len(items))
	for _, it := range items {
		if link := strings.TrimSpace(it.Article.Link); link != "" {
			allowed[link] = true
		}
	}

	removed := 0
	out := linkPattern.ReplaceAllStringFunc(markdown, func(m string) string {
		sub := linkPattern.FindStringSubmatch(m)
		switch {
		case sub[2] != "" || strings.HasPrefix(m, "[") || strings.HasPrefix(m, "!["): // link o imagen markdown
			if allowed[sub[2]] && !strings.HasPrefix(m, "!") {
				return m
			}
			removed++
			if strings.HasPrefix(m, "!") {
				return ""
			}
			return sub[1]
		case sub[3] != "": // autolink <https://...>
			if allowed[sub[3]] {
				return m
			}
			removed++
			return ""
		default: // URL suelta
			if allowed[m] {
				return m
			}
			removed++
			return ""
		}
	})
	return out, removed
}
