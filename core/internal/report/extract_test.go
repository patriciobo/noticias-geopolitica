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

func TestExtractEmailSections(t *testing.T) {
	got, err := ExtractEmailSections(fixtureReport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"## Resumen ejecutivo",
		"Un párrafo breve con lo más importante del día.",
		"## Resumen por región",
		"### Cobertura cruzada",
		"### América del Norte",
		"### América Latina",
		"Sin novedades relevantes hoy.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"Clima internacional",
		"Empresas potencialmente afectadas",
		"Noticias utilizadas",
	} {
		if strings.Contains(got, notWant) {
			t.Errorf("unexpected %q leaked into email sections:\n%s", notWant, got)
		}
	}
}

func TestExtractEmailSectionsMissingSection(t *testing.T) {
	if _, err := ExtractEmailSections("## Resumen ejecutivo\n\ntexto\n"); err == nil {
		t.Error("expected error when 'Resumen por región' is missing")
	}
}
