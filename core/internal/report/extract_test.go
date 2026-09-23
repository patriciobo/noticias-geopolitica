package report

import (
	"strings"
	"testing"
)

const fixtureReport = `## Resumen ejecutivo

Un párrafo breve con lo más importante del día.

## Resumen por región

### Cobertura cruzada

- 🌐 Algo importante confirmado por varios medios

### América del Norte

- **Estados Unidos**: algo pasó.

### América Latina

Sin novedades relevantes hoy.

## Clima internacional: comercio, industria y materias primas

### Comercio

- Algo de comercio.

## Empresas potencialmente afectadas por región

### Europa

- **Empresa X**: por tal motivo.

## Noticias utilizadas

- [Título](<https://a.example>) — Medio (País)
`

func TestSections(t *testing.T) {
	sections := Sections(fixtureReport)

	exec := sections["Resumen ejecutivo"]
	if exec != "Un párrafo breve con lo más importante del día." {
		t.Errorf("Resumen ejecutivo = %q", exec)
	}

	region := sections["Resumen por región"]
	for _, want := range []string{"### Cobertura cruzada", "### América del Norte", "### América Latina", "Sin novedades relevantes hoy."} {
		if !strings.Contains(region, want) {
			t.Errorf("missing %q in Resumen por región:\n%s", want, region)
		}
	}
	if strings.Contains(region, "Clima internacional") {
		t.Errorf("Resumen por región leaked next section:\n%s", region)
	}

	if _, ok := sections["Sección inexistente"]; ok {
		t.Error("expected missing section to be absent from map")
	}
}

func TestSubSections(t *testing.T) {
	region := Sections(fixtureReport)["Resumen por región"]
	subs := SubSections(region)

	wantHeadings := []string{"Cobertura cruzada", "América del Norte", "América Latina"}
	if len(subs) != len(wantHeadings) {
		t.Fatalf("got %d subsections, want %d: %+v", len(subs), len(wantHeadings), subs)
	}
	for i, want := range wantHeadings {
		if subs[i].Heading != want {
			t.Errorf("subs[%d].Heading = %q, want %q", i, subs[i].Heading, want)
		}
	}
	if subs[2].Body != "Sin novedades relevantes hoy." {
		t.Errorf("América Latina body = %q", subs[2].Body)
	}
}

func TestSubSectionsEmpty(t *testing.T) {
	if got := SubSections("solo un párrafo, sin subsecciones"); len(got) != 0 {
		t.Errorf("expected no subsections, got %+v", got)
	}
}
