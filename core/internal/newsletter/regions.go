package newsletter

import (
	"html/template"
	"strings"

	"noticias/core/internal/report"
)

// emptyRegionText debe coincidir EXACTO con el placeholder que
// core/internal/report/synthesize.go le pide al LLM (y que
// ensureAllRegionsPresent inserta si el LLM lo omite) para una región sin
// artículos ese día.
const emptyRegionText = "Sin novedades relevantes hoy."

// RegionBlock es una región ya renderizada a HTML, lista para el template
// del email.
type RegionBlock struct {
	Label string
	HTML  template.HTML
}

// SplitRegions parte el contenido de "Resumen por región" en subsecciones
// (una por región, más "Cobertura cruzada" si está presente) y separa las
// que no tuvieron novedades de las que sí tienen contenido real — el email
// las agrupa compactas en vez de darles el mismo espacio que a una región
// con noticias, así el layout no se ve roto en los días (frecuentes) donde
// una o más regiones no tienen nada que informar.
func SplitRegions(sectionBody string) (populated, empty []RegionBlock) {
	for _, sub := range report.SubSections(sectionBody) {
		if strings.TrimSpace(sub.Body) == emptyRegionText {
			empty = append(empty, RegionBlock{Label: sub.Heading})
			continue
		}
		populated = append(populated, RegionBlock{
			Label: sub.Heading,
			HTML:  template.HTML(MarkdownFragmentToHTML(sub.Body)),
		})
	}
	return populated, empty
}
