package report

import (
	"testing"

	"noticias/core/internal/model"
)

func TestStripUnknownLinks(t *testing.T) {
	items := []model.ClassifiedArticle{
		{Article: model.Article{Link: "https://medio.com/nota-1"}},
	}
	cases := []struct {
		name, in, want string
		removed        int
	}{
		{"sin links", "- **Brasil** firmó un acuerdo.", "- **Brasil** firmó un acuerdo.", 0},
		{"link conocido", "ver [la nota](https://medio.com/nota-1).", "ver [la nota](https://medio.com/nota-1).", 0},
		{"link desconocido queda el texto", "ver [esto](https://phishing.example/login) ya.", "ver esto ya.", 1},
		{"link javascript", "[click](javascript:alert(1))", "click)", 1}, // el paréntesis sobrante queda como texto, sin link
		{"imagen", "antes ![x](https://tracker.example/p.gif) después", "antes  después", 1},
		{"autolink desconocido", "fuente: <https://otro.example/a>", "fuente: ", 1},
		{"url suelta desconocida", "más en https://otro.example/a hoy", "más en  hoy", 1},
		{"url suelta conocida", "más en https://medio.com/nota-1 hoy", "más en https://medio.com/nota-1 hoy", 0},
	}
	for _, c := range cases {
		got, n := StripUnknownLinks(c.in, items)
		if got != c.want || n != c.removed {
			t.Errorf("%s: got (%q, %d), want (%q, %d)", c.name, got, n, c.want, c.removed)
		}
	}
}
