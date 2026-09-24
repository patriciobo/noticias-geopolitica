package report

import (
	"strings"
	"testing"

	"noticias/core/internal/model"
)

func TestBuildUserPromptIncludesAllRegionsEvenWhenEmpty(t *testing.T) {
	items := []model.ClassifiedArticle{
		{
			Article: model.Article{Title: "Acuerdo comercial"},
			Source:  model.Source{Region: "europe", Country: "Alemania", Name: "Der Spiegel"},
		},
	}

	got := buildUserPrompt(Input{Items: items})

	for _, region := range regionOrder {
		want := "REGIÓN: " + regionLabel(region)
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in prompt:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "sin artículos internacionales clasificados hoy") {
		t.Errorf("expected empty-region placeholder note in prompt:\n%s", got)
	}
}

func TestEnsureAllRegionsPresentRepairsMissingRegion(t *testing.T) {
	report := "## Resumen ejecutivo\n\ntexto\n\n## Resumen por región\n\n### América del Norte\n\n- algo\n\n## Clima internacional: comercio, industria y materias primas\n\notro texto\n"

	got := EnsureAllRegionsPresent(report, nil)

	for _, region := range regionOrder {
		want := "### " + regionLabel(region)
		if !strings.Contains(got, want) {
			t.Errorf("missing %q after repair:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "Sin novedades relevantes hoy.") {
		t.Errorf("expected placeholder text after repair:\n%s", got)
	}
	// La sección siguiente no debe duplicarse ni perderse.
	if strings.Count(got, "## Clima internacional") != 1 {
		t.Errorf("clima section corrupted:\n%s", got)
	}
}

func TestEnsureAllRegionsPresentNoopWhenComplete(t *testing.T) {
	var b strings.Builder
	b.WriteString("## Resumen ejecutivo\n\ntexto\n\n## Resumen por región\n\n")
	for _, region := range regionOrder {
		b.WriteString("### " + regionLabel(region) + "\n\n- algo\n\n")
	}
	report := b.String()

	if got := EnsureAllRegionsPresent(report, nil); got != report {
		t.Errorf("expected no changes when all regions present, got:\n%s", got)
	}
}

func TestBuildUserPromptIncludesSourceKind(t *testing.T) {
	items := []model.ClassifiedArticle{{
		Article: model.Article{Title: "Irán anuncia algo", SourceID: "irna"},
		Source:  model.Source{Name: "IRNA", Country: "Irán", Region: "middle_east", Stance: "oficialista", Ownership: "estatal", OwnershipNote: "agencia oficial"},
	}}
	got := buildUserPrompt(Input{Items: items})
	if !strings.Contains(got, "tipo: estatal (agencia oficial)") {
		t.Errorf("el prompt no incluye el tipo de medio:\n%s", got)
	}
}

func TestEmptyRegionTextDependsOnCoverage(t *testing.T) {
	if got := EmptyRegionText(RegionCoverage{Configured: 12, Responded: 3}); got != "Cobertura insuficiente hoy (3 de 12 medios respondieron)." {
		t.Errorf("cobertura baja: %q", got)
	}
	if got := EmptyRegionText(RegionCoverage{Configured: 12, Responded: 10}); got != "Sin novedades relevantes hoy." {
		t.Errorf("cobertura buena: %q", got)
	}
	if got := EmptyRegionText(RegionCoverage{}); got != "Sin novedades relevantes hoy." {
		t.Errorf("cobertura desconocida: %q", got)
	}
}

func TestEnsureAllRegionsPresentUsesCoverage(t *testing.T) {
	report := "## Resumen ejecutivo\n\nx\n\n## Resumen por región\n\n## Clima internacional\n"
	got := EnsureAllRegionsPresent(report, map[string]RegionCoverage{"europe": {Configured: 25, Responded: 4}})
	if !strings.Contains(got, "### Europa\n\nCobertura insuficiente hoy (4 de 25 medios respondieron).") {
		t.Errorf("no marcó la cobertura insuficiente de Europa:\n%s", got)
	}
}

func TestBuildUserPromptMentionsCoverage(t *testing.T) {
	got := buildUserPrompt(Input{Coverage: map[string]RegionCoverage{"europe": {Configured: 25, Responded: 4}}})
	if !strings.Contains(got, "REGIÓN: Europa (respondieron hoy 4 de 25 medios)") || !strings.Contains(got, "Cobertura insuficiente hoy (4 de 25 medios respondieron).") {
		t.Errorf("prompt sin cobertura:\n%s", got)
	}
}
