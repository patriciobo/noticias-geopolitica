package report

import (
	"testing"

	"noticias/core/internal/model"
)

func art(sourceID, country, title string, companies, countries []string, relation string) model.ClassifiedArticle {
	return model.ClassifiedArticle{
		Article: model.Article{SourceID: sourceID, Title: title},
		Source:  model.Source{Country: country},
		Classification: model.Classification{
			IsInternational: true,
			Countries:       countries,
			Companies:       companies,
			RelationType:    relation,
		},
	}
}

func TestClusterStoriesGroupsSharedCompanyAcrossCountries(t *testing.T) {
	items := []model.ClassifiedArticle{
		art("nyt", "Estados Unidos", "Volkswagen anuncia recorte de personal", []string{"Volkswagen"}, []string{"Alemania"}, "supply_chain"),
		art("der-spiegel", "Alemania", "VW reduce plantilla en Wolfsburgo", []string{"Volkswagen"}, []string{"Alemania"}, "supply_chain"),
	}
	clusters := clusterStories(items, nil)
	if len(clusters) != 1 {
		t.Fatalf("esperaba 1 cluster, dio %d", len(clusters))
	}
	cl := clusters[0]
	if cl.sourceCount() != 2 {
		t.Errorf("sourceCount: got %d want 2", cl.sourceCount())
	}
	if cl.countryCount() != 2 {
		t.Errorf("countryCount: got %d want 2", cl.countryCount())
	}
	// 2 medios + 2 países*2 = 6
	if cl.weight() != 6 {
		t.Errorf("weight: got %v want 6", cl.weight())
	}
}

func TestClusterStoriesSameCountryWeighsLess(t *testing.T) {
	items := []model.ClassifiedArticle{
		art("la-nacion", "Argentina", "YPF firma acuerdo de litio", []string{"YPF"}, []string{"Argentina"}, "trade"),
		art("pagina12", "Argentina", "YPF avanza en exportación de litio", []string{"YPF"}, []string{"Argentina"}, "trade"),
	}
	clusters := clusterStories(items, nil)
	if len(clusters) != 1 {
		t.Fatalf("esperaba 1 cluster, dio %d", len(clusters))
	}
	cl := clusters[0]
	// 2 medios + 1 país*2 = 4, menos que el caso cross-country (6)
	if cl.weight() != 4 {
		t.Errorf("weight: got %v want 4", cl.weight())
	}
}

func TestClusterStoriesDoesNotMergeUnrelatedArticles(t *testing.T) {
	items := []model.ClassifiedArticle{
		art("nyt", "Estados Unidos", "Titular A", []string{"Tesla"}, []string{"Estados Unidos"}, "trade"),
		art("le-monde", "Francia", "Titular B", []string{"TotalEnergies"}, []string{"Francia"}, "regulatory"),
	}
	clusters := clusterStories(items, nil)
	if len(clusters) != 2 {
		t.Fatalf("esperaba 2 clusters separados, dio %d", len(clusters))
	}
	for _, cl := range clusters {
		if cl.sourceCount() != 1 {
			t.Errorf("esperaba clusters singleton, sourceCount=%d", cl.sourceCount())
		}
	}
}

func TestClusterStoriesSortedByWeightDescending(t *testing.T) {
	items := []model.ClassifiedArticle{
		art("s1", "Argentina", "Historia con una sola mención", []string{"EmpresaChica"}, []string{"Argentina"}, "trade"),
		art("s2", "Alemania", "Historia con cobertura cruzada", []string{"Volkswagen"}, []string{"Alemania"}, "supply_chain"),
		art("s3", "Estados Unidos", "Historia con cobertura cruzada", []string{"Volkswagen"}, []string{"Alemania"}, "supply_chain"),
	}
	clusters := clusterStories(items, nil)
	if len(clusters) != 2 {
		t.Fatalf("esperaba 2 clusters, dio %d", len(clusters))
	}
	if clusters[0].weight() < clusters[1].weight() {
		t.Errorf("clusters no están ordenados de mayor a menor peso")
	}
	if clusters[0].sourceCount() != 2 {
		t.Errorf("esperaba que el cluster de mayor peso sea el de cobertura cruzada")
	}
}
