package filter

import (
	"testing"

	"noticias/core/internal/model"
)

func testGazetteer() *Gazetteer {
	return &Gazetteer{
		Countries: []string{"Argentina", "China", "Estados Unidos"},
		Companies: []string{"Tesla", "Huawei"},
		Keywords:  []string{"arancel", "tratado"},
	}
}

func TestPrefilterPassesOnCompanyMention(t *testing.T) {
	a := model.Article{Title: "Tesla anuncia nueva planta", Snippet: "expansión en el mercado local"}
	res := Prefilter(a, testGazetteer())
	if !res.Passed {
		t.Fatalf("esperaba Passed=true por mención de empresa, dio false")
	}
	if len(res.MatchedCompanies) != 1 || res.MatchedCompanies[0] != "Tesla" {
		t.Fatalf("esperaba matched company Tesla, dio %v", res.MatchedCompanies)
	}
}

func TestPrefilterPassesOnKeyword(t *testing.T) {
	a := model.Article{Title: "Gobierno sube el arancel a importaciones", Snippet: ""}
	res := Prefilter(a, testGazetteer())
	if !res.Passed {
		t.Fatalf("esperaba Passed=true por keyword de comercio, dio false")
	}
}

func TestPrefilterPassesOnTwoCountries(t *testing.T) {
	a := model.Article{Title: "Argentina y China firman acuerdo", Snippet: ""}
	res := Prefilter(a, testGazetteer())
	if !res.Passed {
		t.Fatalf("esperaba Passed=true por dos países mencionados, dio false")
	}
}

func TestPrefilterRejectsSingleDomesticCountry(t *testing.T) {
	a := model.Article{Title: "Argentina define nuevo gabinete", Snippet: "cambios en ministerios"}
	res := Prefilter(a, testGazetteer())
	if res.Passed {
		t.Fatalf("esperaba Passed=false para noticia doméstica sin empresa/keyword, dio true")
	}
}

func TestPrefilterCaseInsensitive(t *testing.T) {
	a := model.Article{Title: "TESLA amplía operaciones", Snippet: ""}
	res := Prefilter(a, testGazetteer())
	if !res.Passed {
		t.Fatalf("esperaba matching case-insensitive, dio Passed=false")
	}
}
