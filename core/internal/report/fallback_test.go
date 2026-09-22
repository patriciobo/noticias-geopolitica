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

func TestChainSynthesizerUsesFirstOnSuccess(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{markdown: "first"},
		stubSynthesizer{markdown: "second"},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "first" {
		t.Errorf("expected first link's result, got %q", got)
	}
}

func TestChainSynthesizerFallsThroughOnError(t *testing.T) {
	c := NewChainSynthesizer(
		stubSynthesizer{err: errors.New("caído")},
		stubSynthesizer{err: errors.New("también caído")},
		stubSynthesizer{markdown: "third"},
	)
	got, err := c.Synthesize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "third" {
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
