package newsletter

import (
	"html"
	"regexp"
	"strings"
)

var boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)

// MarkdownFragmentToHTML convierte a HTML el fragmento de markdown que
// arma ExtractEmailSections. Es un conversor mínimo hecho a mano, no una
// librería de markdown general (goldmark, etc): el contenido es
// completamente predecible porque lo genera nuestro propio LLM siguiendo
// un system prompt estricto (##/###, bullets "-", **negrita**, párrafos
// cortos) — traer una dependencia general sería sobre-ingeniería para este
// alcance. Si el reporte incorpora markdown más rico en el futuro (links,
// tablas), revisar esta decisión.
func MarkdownFragmentToHTML(md string) string {
	lines := strings.Split(md, "\n")

	var b strings.Builder
	inList := false
	closeList := func() {
		if inList {
			b.WriteString("</ul>\n")
			inList = false
		}
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			closeList()
		case strings.HasPrefix(trimmed, "### "):
			closeList()
			b.WriteString("<h3>" + inline(trimmed[4:]) + "</h3>\n")
		case strings.HasPrefix(trimmed, "## "):
			closeList()
			b.WriteString("<h2>" + inline(trimmed[3:]) + "</h2>\n")
		case strings.HasPrefix(trimmed, "- "):
			if !inList {
				b.WriteString("<ul>\n")
				inList = true
			}
			b.WriteString("<li>" + inline(trimmed[2:]) + "</li>\n")
		default:
			closeList()
			b.WriteString("<p>" + inline(trimmed) + "</p>\n")
		}
	}
	closeList()

	return b.String()
}

// inline escapa HTML y después aplica el único inline markdown que usa el
// reporte: **negrita**. El escape va primero para que un "<" o "&" literal
// en el texto no rompa el HTML resultante; **...** sigue intacto porque
// html.EscapeString no toca asteriscos.
func inline(text string) string {
	escaped := html.EscapeString(text)
	return boldRe.ReplaceAllString(escaped, "<strong>$1</strong>")
}
