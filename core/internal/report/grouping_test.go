package report

import (
	"context"
	"strings"
	"testing"

	"noticias/core/internal/model"
)

func TestGroupStoriesValidatesResponse(t *testing.T) {
	items := make([]model.ClassifiedArticle, 4)
	for i := range items {
		items[i] = model.ClassifiedArticle{Article: model.Article{Title: string(rune('a' + i)), SourceID: string(rune('a' + i))}}
	}
	// 9 fuera de rango, 1 repetido, grupo {4} queda de un solo item.
	stub := &stubCompleter{out: `[{"event":"Cumbre","items":[1,2,9]},{"event":"Otro","items":[1,4]}]`}
	groups, err := GroupStories(context.Background(), stub, items)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Event != "Cumbre" || len(groups[0].Items) != 2 || groups[0].Items[0] != 0 || groups[0].Items[1] != 1 {
		t.Errorf("groups = %+v", groups)
	}
}

func TestClusterStoriesUsesModelGroups(t *testing.T) {
	// Mismos países y relación: por entidades quedarían juntas; el modelo
	// dice que son hechos distintos salvo 0 y 2.
	items := []model.ClassifiedArticle{
		wireArt("a", "España", "privado", "Irán descarta atentado contra Trump", ""),
		wireArt("b", "Italia", "privado", "EE. UU. sanciona bancos iraníes", ""),
		wireArt("c", "Francia", "privado", "L'Iran nie tout complot contre Trump", ""),
	}
	clusters := clusterStories(items, []StoryGroup{{Event: "Irán niega un complot contra Trump", Items: []int{0, 2}}})
	if len(clusters) != 2 || clusters[0].event != "Irán niega un complot contra Trump" || clusters[0].sourceCount() != 2 {
		t.Fatalf("clusters: %d, primero %+v", len(clusters), clusters[0])
	}
	got := buildUserPrompt(Input{Items: items, Groups: []StoryGroup{{Event: "Irán niega un complot contra Trump", Items: []int{0, 2}}}})
	if !strings.Contains(got, "- Irán niega un complot contra Trump — 2 medios") {
		t.Errorf("la cobertura cruzada no usa el hecho del grupo:\n%s", got)
	}
}

func TestGroupStoriesNoGroupsIsNotNil(t *testing.T) {
	items := make([]model.ClassifiedArticle, 3)
	groups, err := GroupStories(context.Background(), &stubCompleter{out: "[]"}, items)
	if err != nil || groups == nil || len(groups) != 0 {
		t.Errorf("groups = %#v, err = %v", groups, err)
	}
}
