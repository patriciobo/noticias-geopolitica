package report

import (
	"fmt"
	"sort"
	"strings"

	"noticias/core/internal/model"
)

// SourcesHeading is the section appended after the LLM-written report.
const SourcesHeading = "## Noticias utilizadas"

// AppendSources adds a "Noticias utilizadas" section listing a link to every
// article that fed the synthesis, grouped by region. Built from the input
// data, not asked of the LLM, so links can't be invented or altered.
func AppendSources(markdown string, items []model.ClassifiedArticle) string {
	section := buildSourcesSection(items)
	if section == "" {
		return markdown
	}
	return strings.TrimRight(markdown, "\n") + "\n\n" + section
}

func buildSourcesSection(items []model.ClassifiedArticle) string {
	byRegion := map[string][]model.ClassifiedArticle{}
	seenLink := map[string]bool{}
	for _, it := range items {
		link := strings.TrimSpace(it.Article.Link)
		if link == "" || seenLink[link] {
			continue
		}
		seenLink[link] = true
		byRegion[it.Source.Region] = append(byRegion[it.Source.Region], it)
	}
	if len(byRegion) == 0 {
		return ""
	}

	regions := make([]string, 0, len(byRegion))
	for r := range byRegion {
		regions = append(regions, r)
	}
	sort.Strings(regions)

	var b strings.Builder
	b.WriteString(SourcesHeading + "\n")
	for _, region := range regions {
		arts := byRegion[region]
		sort.SliceStable(arts, func(i, j int) bool {
			if arts[i].Source.Country != arts[j].Source.Country {
				return arts[i].Source.Country < arts[j].Source.Country
			}
			return arts[i].Source.Name < arts[j].Source.Name
		})
		fmt.Fprintf(&b, "\n### %s\n\n", regionLabel(region))
		for _, a := range arts {
			fmt.Fprintf(&b, "- [%s](<%s>) — %s (%s)\n",
				escapeLinkText(a.Article.Title), a.Article.Link, a.Source.Name, a.Source.Country)
		}
	}
	return b.String()
}

// escapeLinkText keeps titles with brackets or newlines from breaking the
// markdown link syntax.
func escapeLinkText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := strings.NewReplacer(`[`, `\[`, `]`, `\]`)
	return r.Replace(s)
}
