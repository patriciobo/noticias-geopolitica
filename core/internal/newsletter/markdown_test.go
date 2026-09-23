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

func TestMarkdownFragmentToHTMLEscapesRawHTML(t *testing.T) {
	got := MarkdownFragmentToHTML("Texto con <script>alert(1)</script> adentro.")
	if strings.Contains(got, "<script>") {
		t.Errorf("raw HTML was not escaped:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag:\n%s", got)
	}
}
