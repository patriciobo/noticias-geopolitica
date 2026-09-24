package report

import (
	"context"
	"errors"
	"testing"

	"noticias/core/internal/model"
)

type stubSynthesizer struct {
	markdown string
	err      error
}

func (s stubSynthesizer) Synthesize(ctx context.Context, items []model.ClassifiedArticle) (string, error) {
	return s.markdown, s.err
}

// report arma un markdown mínimo que pasa validateReport, marcado con tag
// para distinguir qué link lo produjo.
func report(tag string) string {
	return "## Resumen ejecutivo\n\n" + tag + "\n\n## Resumen por región\n\n## Clima internacional: comercio\n\n## Empresas potencialmente afectadas por región\n"
}

func TestChainSynthesizerUsesFirstOnSuccess(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{markdown: report("first")},
		stubSynthesizer{markdown: report("second")},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != report("first") {
		t.Errorf("expected first link's result, got %q", got)
	}
}

func TestChainSynthesizerFallsThroughOnError(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{err: errors.New("caído")},
		stubSynthesizer{err: errors.New("también caído")},
		stubSynthesizer{markdown: report("third")},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != report("third") {
		t.Errorf("expected third link's result, got %q", got)
	}
}

func TestChainSynthesizerErrorsWhenAllFail(t *testing.T) {
	wantErr := errors.New("último error")
	c := NewChainSynthesizer(
		stubSynthesizer{err: errors.New("primero falló")},
		stubSynthesizer{err: wantErr},
	)
	_, err := c.Synthesize(context.Background(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected last error to propagate, got %v", err)
	}
}

func TestChainSynthesizerSkipsOutputWithoutRequiredSections(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{markdown: "User Safety: safe"},
		stubSynthesizer{markdown: report("second")},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != report("second") {
		t.Errorf("expected second link's result, got %q", got)
	}
}

func TestChainSynthesizerSkipsTruncatedReport(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{markdown: "## Resumen ejecutivo\n\nx\n\n## Resumen por región\n\n- corte a mitad"},
		stubSynthesizer{markdown: report("second")},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != report("second") {
		t.Errorf("expected second link's result, got %q", got)
	}
}
