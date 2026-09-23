package report

import (
	"regexp"
	"strings"
)

var topSectionRe = regexp.MustCompile(`(?m)^## .+$`)
var subSectionRe = regexp.MustCompile(`(?m)^### .+$`)

// Sections devuelve, por cada sección de nivel 2 ("## <heading>") del
// reporte, su contenido (sin la línea del heading) — para que otros
// paquetes (ej. newsletter) puedan extraer piezas puntuales sin
// reimplementar el parseo de markdown.
func Sections(markdown string) map[string]string {
	locs := topSectionRe.FindAllStringIndex(markdown, -1)
	out := make(map[string]string, len(locs))
	for i, loc := range locs {
		end := len(markdown)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		heading := strings.TrimSpace(strings.TrimPrefix(markdown[loc[0]:loc[1]], "## "))
		out[heading] = strings.TrimSpace(markdown[loc[1]:end])
	}
	return out
}

// Subsection es una subsección de nivel 3 ("### <heading>") dentro de una
// sección — ej. una región dentro de "Resumen por región".
type Subsection struct {
	Heading string
	Body    string
}

// SubSections divide el contenido de una sección en sus subsecciones de
// nivel 3, en el orden en que aparecen (a diferencia de Sections, acá el
// orden importa para el caller — ej. mantener el orden editorial de
// regiones — así que devuelve slice, no map).
func SubSections(sectionBody string) []Subsection {
	locs := subSectionRe.FindAllStringIndex(sectionBody, -1)
	out := make([]Subsection, 0, len(locs))
	for i, loc := range locs {
		end := len(sectionBody)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		heading := strings.TrimSpace(strings.TrimPrefix(sectionBody[loc[0]:loc[1]], "### "))
		out = append(out, Subsection{
			Heading: heading,
			Body:    strings.TrimSpace(sectionBody[loc[1]:end]),
		})
	}
	return out
}
