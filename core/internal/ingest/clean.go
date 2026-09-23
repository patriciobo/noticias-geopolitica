package ingest

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Límites de largo para lo que entra desde un feed. Un titular real no pasa
// de un par de cientos de caracteres; un snippet, de un párrafo. Cortar acá
// acota el costo de LLM y el espacio que tendría un feed malicioso (o
// comprometido) para meter instrucciones al modelo (prompt injection).
const (
	maxTitleRunes   = 300
	maxSnippetRunes = 1000
)

// cleanText normaliza texto de un feed antes de que llegue al LLM: saca
// caracteres de control, colapsa saltos de línea y espacios en uno solo
// (un titular con "\n1. Título: ..." adentro podría hacerse pasar por otro
// item de la lista que armamos para el clasificador) y corta al largo máximo.
func cleanText(s string, maxRunes int) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	out := b.String()
	if utf8.RuneCountInString(out) <= maxRunes {
		return out
	}
	runes := []rune(out)
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "…"
}

// cleanLink acepta solo URLs absolutas http(s). Un feed podría mandar un
// link "javascript:..." o "data:..." que después terminaría como enlace en
// el blog o en el mail; mejor descartarlo acá, en la entrada.
func cleanLink(s string) string {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.String()
}
