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

func TestFallbackSynthesizerUsesPrimaryOnSuccess(t *testing.T) {
	f := &FallbackSynthesizer{
		Primary:   stubSynthesizer{markdown: "primary"},
		Secondary: stubSynthesizer{markdown: "secondary"},
	}
	got, err := f.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "primary" {
		t.Errorf("expected primary result, got %q", got)
	}
}

func TestFallbackSynthesizerFallsBackOnPrimaryError(t *testing.T) {
	f := &FallbackSynthesizer{
		Primary:   stubSynthesizer{err: errors.New("cuota agotada")},
		Secondary: stubSynthesizer{markdown: "secondary"},
	}
	got, err := f.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secondary" {
		t.Errorf("expected secondary result, got %q", got)
	}
}

func TestFallbackSynthesizerPropagatesSecondaryError(t *testing.T) {
	wantErr := errors.New("secondary también falló")
	f := &FallbackSynthesizer{
		Primary:   stubSynthesizer{err: errors.New("primary falló")},
		Secondary: stubSynthesizer{err: wantErr},
	}
	_, err := f.Synthesize(context.Background(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected secondary error to propagate, got %v", err)
	}
}
