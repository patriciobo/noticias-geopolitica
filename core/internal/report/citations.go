package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"noticias/core/internal/model"
)

// citationPattern reconoce las citas que escribe el modelo: [3], [3, 17] o
// [3][17] (cada corchete por separado), siempre que no sean ya el texto de
// un link markdown (no seguidas de "(").
var citationPattern = regexp.MustCompile(`\[(\d{1,4}(?:\s*,\s*\d{1,4})*)\](\()?`)

// CitationStats resume qué pasó con las citas de una edición.
type CitationStats struct {
	Resolved int // citas convertidas en link a la nota
	Invalid  int // números que no correspondían a ninguna nota (se sacan)
	Uncited  int // bullets o párrafos de resumen sin ninguna cita
}

// citationNumbers asigna a cada nota su número de cita (1-based, en el
// orden de items) — el mismo que ve el modelo en buildUserPrompt.
func citationNumbers(items []model.ClassifiedArticle) map[string]int {
	nums := make(map[string]int, len(items))
	for i, it := range items {
		nums[articleKey(it.Article)] = i + 1
	}
	return nums
}

// ResolveCitations convierte cada cita [n] del texto del modelo en un link a
// la nota n ([[n]](<url>)), saca los números que no existen y cuenta los
// bullets sin cita. Así cada afirmación del informe se puede rastrear hasta
// el titular del que sale, sin depender de que el modelo obedezca: un
// número inventado no se convierte en link.
func ResolveCitations(markdown string, items []model.ClassifiedArticle) (string, CitationStats) {
	var stats CitationStats
	link := func(n int) (string, bool) {
		if n < 1 || n > len(items) {
			return "", false
		}
		url := strings.TrimSpace(items[n-1].Article.Link)
		if url == "" {
			return fmt.Sprintf("[%d]", n), true
		}
		return fmt.Sprintf("[[%d]](<%s>)", n, url), true
	}

	// Se procesa solo el cuerpo del modelo, no la lista "Noticias
	// utilizadas" que se agrega después.
	out := citationPattern.ReplaceAllStringFunc(markdown, func(m string) string {
		sub := citationPattern.FindStringSubmatch(m)
		if sub[2] == "(" {
			return m // ya es el texto de un link markdown
		}
		var b strings.Builder
		for _, part := range strings.Split(sub[1], ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				stats.Invalid++
				continue
			}
			l, ok := link(n)
			if !ok {
				stats.Invalid++
				continue
			}
			stats.Resolved++
			b.WriteString(l)
		}
		return b.String()
	})

	stats.Uncited = countUncited(out)
	return out, stats
}

// countUncited cuenta bullets (y el párrafo del Resumen ejecutivo) que no
// tienen ninguna cita resuelta. Las líneas de "sin novedades", cobertura
// insuficiente o parcial no cuentan: no afirman nada.
func countUncited(markdown string) int {
	uncited := 0
	section := ""
	for _, line := range strings.Split(markdown, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			section = strings.TrimPrefix(t, "## ")
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") || section == "Noticias utilizadas" {
			continue
		}
		if strings.HasPrefix(t, "Sin novedades") || strings.HasPrefix(t, "Cobertura insuficiente") || strings.HasPrefix(t, "_Cobertura parcial") {
			continue
		}
		isBullet := strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")
		// Bullets, el resumen ejecutivo y los párrafos de los informes por
		// región (que desde el 2026-09-25 van en prosa) tienen que citar.
		prose := section == "Resumen ejecutivo" || section == "Resumen por región"
		if (isBullet || prose) && !strings.Contains(t, "[[") {
			uncited++
		}
	}
	return uncited
}
