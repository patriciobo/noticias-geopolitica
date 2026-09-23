package newsletter

import (
	"html"
	"regexp"
	"strings"
)

var boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)
var italicRe = regexp.MustCompile(`\*(.+?)\*`)

// MarkdownFragmentToHTML convierte a HTML un fragmento de markdown del
// reporte (una sección o subsección puntual, ver report.Sections/SubSections).
// Es un conversor mínimo hecho a mano, no una
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

// inline escapa HTML y después aplica el inline markdown que puede
// aparecer en el reporte: **negrita** y *cursiva*. El escape va primero
// para que un "<" o "&" literal en el texto no rompa el HTML resultante;
// los asteriscos no se tocan. Negrita se resuelve ANTES que cursiva a
// propósito: una vez reemplazado "**x**" por "<strong>x</strong>" no quedan
// pares de asteriscos dobles sueltos, así que la regex de cursiva (un solo
// asterisco) no puede matchear por error dentro de lo que ya era negrita.
func inline(text string) string {
	escaped := html.EscapeString(text)
	bold := boldRe.ReplaceAllString(escaped, "<strong>$1</strong>")
	return italicRe.ReplaceAllString(bold, "<em>$1</em>")
}
