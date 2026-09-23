package report

import (
	"fmt"
	"regexp"
	"strings"
)

var topSectionRe = regexp.MustCompile(`(?m)^## .+$`)

// emailSections son los headings (tal cual aparecen en el markdown) que se
// mandan por correo, en el orden en que deben aparecer en el email —
// independientemente del orden en que aparezcan en el reporte completo.
var emailSections = []string{"## Resumen ejecutivo", "## Resumen por región"}

// ExtractEmailSections corta del reporte del día únicamente las secciones
// que van en el newsletter. "Resumen por región" incluye tal cual el
// sub-bloque opcional "### Cobertura cruzada" si está presente, porque cae
// dentro de su rango — no hace falta tratarlo aparte. El resto del reporte
// ("Clima internacional", "Empresas afectadas", "Noticias utilizadas") no
// se manda por correo.
func ExtractEmailSections(markdown string) (string, error) {
	sections := splitTopSections(markdown)

	picked := make([]string, 0, len(emailSections))
	for _, heading := range emailSections {
		body, ok := sections[heading]
		if !ok {
			return "", fmt.Errorf("extract: no se encontró la sección %q en el reporte", heading)
		}
		picked = append(picked, strings.TrimSpace(body))
	}
	return strings.Join(picked, "\n\n"), nil
}

// splitTopSections divide el markdown en un mapa heading ("## ...") ->
// cuerpo completo (heading + contenido) hasta el próximo heading de nivel 2
// o el final del documento.
func splitTopSections(markdown string) map[string]string {
	locs := topSectionRe.FindAllStringIndex(markdown, -1)
	out := make(map[string]string, len(locs))
	for i, loc := range locs {
		end := len(markdown)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		heading := strings.TrimSpace(markdown[loc[0]:loc[1]])
		out[heading] = markdown[loc[0]:end]
	}
	return out
}
