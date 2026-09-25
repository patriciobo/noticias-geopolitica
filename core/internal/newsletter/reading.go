package newsletter

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// wordsPerMinute: lectura en español de texto periodístico. Mismo valor
// que web/src/lib/readingTime.ts.
const wordsPerMinute = 200

var (
	citationLinkRe = regexp.MustCompile(`\[\[\d+\]\]\(<[^>]*>\)`)
	anyLinkRe      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	markupRe       = regexp.MustCompile("[#*_>`|-]")
)

// ReadingMinutes estima los minutos de lectura de la edición completa:
// cuenta las palabras sin la lista "Noticias utilizadas", sin citas ni
// sintaxis de markdown. Mismo cálculo que la web.
func ReadingMinutes(markdown string) int {
	if i := strings.Index(markdown, "\n## Noticias utilizadas"); i != -1 {
		markdown = markdown[:i]
	}
	text := citationLinkRe.ReplaceAllString(markdown, " ")
	text = anyLinkRe.ReplaceAllString(text, "$1")
	text = markupRe.ReplaceAllString(text, " ")
	words := 0
	for _, w := range strings.Fields(text) {
		if strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }) != -1 {
			words++
		}
	}
	return max(1, int(math.Round(float64(words)/wordsPerMinute)))
}
