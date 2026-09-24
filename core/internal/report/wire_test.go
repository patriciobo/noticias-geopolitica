package report

import (
	"testing"

	"noticias/core/internal/model"
)

func wireArt(id, country, ownership, title, snippet string) model.ClassifiedArticle {
	return model.ClassifiedArticle{
		Article:        model.Article{SourceID: id, Title: title, Snippet: snippet},
		Source:         model.Source{ID: id, Name: id, Country: country, Ownership: ownership},
		Classification: model.Classification{Countries: []string{"Irán", "EE. UU."}, RelationType: "sanction"},
	}
}

func TestSameWireCountsAsOneVoice(t *testing.T) {
	items := []model.ClassifiedArticle{
		wireArt("a", "España", "privado", "EE. UU. sanciona a Irán", "WASHINGTON (Reuters) - El Tesoro..."),
		wireArt("b", "Italia", "privado", "Sanzioni USA all'Iran", "(Reuters) Il Tesoro americano..."),
		wireArt("c", "Francia", "privado", "Sanctions américaines", "Selon Reuters, le Trésor..."),
	}
	cl := clusterStories(items)[0]
	if cl.sourceCount() != 3 || cl.independentCount() != 1 {
		t.Errorf("medios=%d independientes=%d; want 3 y 1", cl.sourceCount(), cl.independentCount())
	}
	if cl.wireSummary() != "Reuters en 3 medios" {
		t.Errorf("wireSummary = %q", cl.wireSummary())
	}
}

func TestStateMediaOfSameCountryIsOneVoice(t *testing.T) {
	items := []model.ClassifiedArticle{
		wireArt("rt", "Rusia", "estatal", "US sanctions Iran", "Washington announced..."),
		wireArt("sputnik", "Rusia", "estatal", "US sanctions Tehran", "New sanctions..."),
		wireArt("haaretz", "Israel", "privado", "US slaps sanctions on Iran", "The Treasury..."),
	}
	cl := clusterStories(items)[0]
	if cl.independentCount() != 2 {
		t.Errorf("independientes=%d; want 2 (Estado ruso + Haaretz)", cl.independentCount())
	}
}

func TestOwnWireIsNotForeignCable(t *testing.T) {
	it := wireArt("anadolu", "Turquía", "estatal", "Turkey says", "ANKARA (Anadolu) - ...")
	it.Source.Name = "Anadolu Agency"
	if w := detectWire(it); w != "" {
		t.Errorf("la nota propia de Anadolu se detectó como cable %q", w)
	}
}
