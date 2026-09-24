package report

import (
	"context"
	"strings"
	"testing"

	"noticias/core/internal/model"
)

type stubCompleter struct {
	out     string
	prompts []string
}

func (s *stubCompleter) Complete(ctx context.Context, system, user string, temperature float64) (string, error) {
	s.prompts = append(s.prompts, user)
	return s.out, nil
}

func TestCheckFidelity(t *testing.T) {
	items := []model.ClassifiedArticle{
		{Article: model.Article{Title: "Irán anuncia X", Snippet: "Teherán dijo X.", Link: "https://a"}, Source: model.Source{Name: "IRNA", Country: "Irán", Ownership: "estatal"}},
		{Article: model.Article{Title: "EE. UU. responde", Link: "https://b"}, Source: model.Source{Name: "NYT", Country: "Estados Unidos"}},
	}
	md := "## Resumen ejecutivo\n\nIrán anunció X [[1]](<https://a>).\n\n## Resumen por región\n\n### Medio Oriente\n\n- según NYT, EE. UU. respondió con 40 sanciones [[2]](<https://b>)\n- sin cita\n\n## Noticias utilizadas\n\n- [x](<https://a>)\n"
	stub := &stubCompleter{out: "```json\n[{\"id\":1,\"verdict\":\"respaldada\"},{\"id\":2,\"verdict\":\"sin_respaldo\",\"problem\":\"la nota no dice 40\"}]\n```"}

	res, err := CheckFidelity(context.Background(), stub, md, items)
	if err != nil {
		t.Fatal(err)
	}
	if res.Checked != 2 || res.Supported != 1 || res.Unsupported != 1 {
		t.Fatalf("res = %+v", res)
	}
	if res.Issues[0].Problem != "la nota no dice 40" || res.Issues[0].Citations[0] != 2 || strings.Contains(res.Issues[0].Text, "[[") {
		t.Errorf("issue = %+v", res.Issues[0])
	}
	if !strings.Contains(stub.prompts[0], "Nota [1] — IRNA (Irán, medio estatal)") {
		t.Errorf("prompt sin las notas citadas:\n%s", stub.prompts[0])
	}
}
