package report

import (
	"strings"
	"testing"

	"noticias/core/internal/model"
)

func TestAppendSources(t *testing.T) {
	mk := func(region, country, name, title, link string) model.ClassifiedArticle {
		return model.ClassifiedArticle{
			Article: model.Article{Title: title, Link: link},
			Source:  model.Source{Region: region, Country: country, Name: name},
		}
	}
	items := []model.ClassifiedArticle{
		mk("europe", "Alemania", "Der Spiegel", "Acuerdo [UE] y Mercosur", "https://a.example/1"),
		mk("europe", "Francia", "Le Monde", "Duplicada", "https://a.example/1"),
		mk("latin_america", "Argentina", "La Nación", "Soja", "https://b.example/2"),
		mk("europe", "Francia", "Le Monde", "Sin link", ""),
	}

	got := AppendSources("# Reporte\n\ntexto\n", items)

	for _, want := range []string{
		"\n\n## Noticias utilizadas\n",
		"### Europa",
		"### América Latina",
		`- [Acuerdo \[UE\] y Mercosur](<https://a.example/1>) — Der Spiegel (Alemania)`,
		"- [Soja](<https://b.example/2>) — La Nación (Argentina)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Duplicada") || strings.Contains(got, "Sin link") {
		t.Errorf("duplicate or link-less article listed:\n%s", got)
	}
}

func TestAppendSourcesEmpty(t *testing.T) {
	if got := AppendSources("x", nil); got != "x" {
		t.Errorf("got %q", got)
	}
}
