// Command muestra arma la revisión humana semanal: elige al azar titulares
// clasificados y afirmaciones de los informes de los últimos días y los
// imprime como una lista para revisar a mano (el workflow semanal la abre
// como issue). Es el control independiente del modelo: el chequeo de
// fidelidad lo hace un modelo, esto lo hace una persona.
//
// Uso (desde core/): go run ./cmd/muestra [-dias 7] [-titulares 20] [-afirmaciones 10]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"noticias/core/internal/model"
)

var citationRe = regexp.MustCompile(`\[\[(\d+)\]\]\(<([^>]*)>\)`)

type headline struct {
	date string
	model.AuditEntry
}

type claimSample struct {
	date, section, text string
	links               []string
}

func main() {
	dir := flag.String("dir", "../reports", "carpeta de reportes")
	days := flag.Int("dias", 7, "días hacia atrás")
	nHeadlines := flag.Int("titulares", 20, "titulares a revisar")
	nClaims := flag.Int("afirmaciones", 10, "afirmaciones a revisar")
	flag.Parse()

	var headlines []headline
	var claims []claimSample
	for d := 0; d < *days; d++ {
		date := time.Now().UTC().AddDate(0, 0, -d).Format("2006-01-02")
		if raw, err := os.ReadFile(filepath.Join(*dir, date+".audit.json")); err == nil {
			var a model.Audit
			if json.Unmarshal(raw, &a) == nil {
				for _, e := range a.Headlines {
					if e.Stage == model.StageAccepted || e.Stage == model.StageClassifierRejected {
						headlines = append(headlines, headline{date, e})
					}
				}
			}
		}
		if raw, err := os.ReadFile(filepath.Join(*dir, date+".md")); err == nil {
			claims = append(claims, extractClaims(date, string(raw))...)
		}
	}
	if len(headlines) == 0 && len(claims) == 0 {
		log.Fatalf("no hay reportes en %s de los últimos %d días", *dir, *days)
	}

	rand.Shuffle(len(headlines), func(i, j int) { headlines[i], headlines[j] = headlines[j], headlines[i] })
	rand.Shuffle(len(claims), func(i, j int) { claims[i], claims[j] = claims[j], claims[i] })
	headlines = headlines[:min(*nHeadlines, len(headlines))]
	claims = claims[:min(*nClaims, len(claims))]
	sort.SliceStable(headlines, func(i, j int) bool { return headlines[i].date < headlines[j].date })

	fmt.Printf(`Revisión humana por muestreo de los últimos %d días. Marcá cada casilla si la decisión o la afirmación es correcta; si no, dejala sin marcar y explicá en un comentario. Al terminar, anotá el resultado (por ejemplo "18/20 titulares y 9/10 afirmaciones correctos") y cerrá el issue.

## Titulares clasificados (%d)

¿Estuvo bien incluirlo o descartarlo, según el criterio de alcance internacional?

`, *days, len(headlines))
	for _, h := range headlines {
		decision := "descartado"
		if h.Stage == model.StageAccepted {
			decision = "**incluido**"
		}
		title := strings.NewReplacer("[", `\[`, "]", `\]`).Replace(h.Title)
		link := title
		if h.Link != "" {
			link = fmt.Sprintf("[%s](%s)", title, h.Link)
		}
		fmt.Printf("- [ ] %s — %s (%s), %s. Motivo del modelo: %s\n", h.date, link, h.Source, decision, h.Reason)
	}

	fmt.Printf("\n## Afirmaciones de los informes (%d)\n\n¿Lo que dice está en las notas que cita, atribuido al medio correcto?\n\n", len(claims))
	for _, c := range claims {
		var refs []string
		for i, l := range c.links {
			refs = append(refs, fmt.Sprintf("[nota %d](%s)", i+1, l))
		}
		fmt.Printf("- [ ] %s, %s: %s — %s\n", c.date, c.section, c.text, strings.Join(refs, ", "))
	}
}

func extractClaims(date, markdown string) []claimSample {
	var out []claimSample
	section := ""
	for _, line := range strings.Split(markdown, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			section = strings.TrimPrefix(t, "## ")
			continue
		}
		// Bullets y párrafos (los informes por región van en prosa).
		if section == "Noticias utilizadas" || t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ms := citationRe.FindAllStringSubmatch(t, -1)
		if len(ms) == 0 {
			continue
		}
		var links []string
		for _, m := range ms {
			links = append(links, m[2])
		}
		text := strings.TrimSpace(citationRe.ReplaceAllString(strings.TrimPrefix(t, "- "), ""))
		out = append(out, claimSample{date: date, section: section, text: text, links: links})
	}
	return out
}
