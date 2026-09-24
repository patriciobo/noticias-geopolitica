package newsletter

import (
	"strings"
	"testing"
)

func TestMarkdownFragmentToHTML(t *testing.T) {
	md := "## Resumen ejecutivo\n\nUn párrafo con **negrita** y texto normal.\n\n## Resumen por región\n\n### América Latina\n\n- **Argentina**: algo pasó.\n- Otro punto sin negrita.\n"

	got := MarkdownFragmentToHTML(md)

	for _, want := range []string{
		"<h2>Resumen ejecutivo</h2>",
		"<p>Un párrafo con <strong>negrita</strong> y texto normal.</p>",
		"<h3>América Latina</h3>",
		"<ul>",
		"<li><strong>Argentina</strong>: algo pasó.</li>",
		"<li>Otro punto sin negrita.</li>",
		"</ul>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestMarkdownFragmentToHTMLItalic(t *testing.T) {
	got := MarkdownFragmentToHTML("Según *La Jornada*, algo pasó con **Estados Unidos**.")
	for _, want := range []string{"<em>La Jornada</em>", "<strong>Estados Unidos</strong>"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "*") {
		t.Errorf("leftover literal asterisk in:\n%s", got)
	}
}

func TestMarkdownFragmentToHTMLEscapesRawHTML(t *testing.T) {
	got := MarkdownFragmentToHTML("Texto con <script>alert(1)</script> adentro.")
	if strings.Contains(got, "<script>") {
		t.Errorf("raw HTML was not escaped:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag:\n%s", got)
	}
}

func TestInlineRendersCitationLinks(t *testing.T) {
	got := inline("**Francia**: según Le Monde, x [[2]](<https://b.example/2?a=1&b=2>) y <script>")
	want := `<strong>Francia</strong>: según Le Monde, x <a href="https://b.example/2?a=1&amp;b=2" style="color:#c81e1e; text-decoration:none;">[2]</a> y &lt;script&gt;`
	if got != want {
		t.Errorf("inline =\n%s\nwant\n%s", got, want)
	}
	if got := inline("[click](javascript:alert(1))"); strings.Contains(got, "href") {
		t.Errorf("aceptó un href no http: %s", got)
	}
}
