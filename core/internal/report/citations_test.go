package report

import (
	"strings"
	"testing"

	"noticias/core/internal/model"
)

func citeItems() []model.ClassifiedArticle {
	return []model.ClassifiedArticle{
		{Article: model.Article{Title: "a", Link: "https://a.example/1"}},
		{Article: model.Article{Title: "b", Link: "https://b.example/2"}},
	}
}

func TestResolveCitations(t *testing.T) {
	md := "## Resumen ejecutivo\n\nAlgo pasó [1][2].\n\n## Resumen por región\n\n### Europa\n\n- **Francia**: según Le Monde, x [2, 1].\n- **Italia**: sin cita.\n- **España**: número inventado [9].\n\n### Asia Oriental\n\nSin novedades relevantes hoy.\n"
	got, stats := ResolveCitations(md, citeItems())

	if !strings.Contains(got, "Algo pasó [[1]](<https://a.example/1>)[[2]](<https://b.example/2>).") {
		t.Errorf("citas contiguas mal resueltas:\n%s", got)
	}
	if !strings.Contains(got, "x [[2]](<https://b.example/2>)[[1]](<https://a.example/1>).") {
		t.Errorf("cita con coma mal resuelta:\n%s", got)
	}
	if strings.Contains(got, "[9]") {
		t.Errorf("no sacó la cita inventada:\n%s", got)
	}
	if stats.Resolved != 4 || stats.Invalid != 1 || stats.Uncited != 2 {
		t.Errorf("stats = %+v, want Resolved 4, Invalid 1, Uncited 2 (Italia y España)", stats)
	}
}

func TestResolveCitationsLeavesMarkdownLinksAlone(t *testing.T) {
	md := "- ver [3](https://x.example) [1]"
	got, stats := ResolveCitations(md, citeItems())
	if !strings.Contains(got, "[3](https://x.example)") || stats.Resolved != 1 {
		t.Errorf("got %q stats %+v", got, stats)
	}
}

func TestBuildUserPromptNumbersItems(t *testing.T) {
	items := []model.ClassifiedArticle{
		{Article: model.Article{Title: "uno", SourceID: "s1"}, Source: model.Source{Region: "europe"}},
		{Article: model.Article{Title: "dos", SourceID: "s2"}, Source: model.Source{Region: "europe"}},
	}
	got := buildUserPrompt(Input{Items: items})
	if !strings.Contains(got, "- [1] [") || !strings.Contains(got, "- [2] [") {
		t.Errorf("titulares sin número:\n%s", got)
	}
}

func TestCountUncitedProseParagraphs(t *testing.T) {
	md := "## Resumen por región\n\n### Europa\n\nUn párrafo con cita [[1]](<https://a>).\n\nOtro párrafo sin cita.\n\n### Asia Oriental\n\nSin novedades relevantes hoy.\n\n## Clima internacional\n\nIntroducción sin cita que no cuenta.\n"
	if got := countUncited(md); got != 1 {
		t.Errorf("countUncited = %d, want 1 (el párrafo de Europa sin cita)", got)
	}
}
