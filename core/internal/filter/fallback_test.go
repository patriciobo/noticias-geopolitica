package filter

import (
	"context"
	"errors"
	"testing"

	"noticias/core/internal/model"
)

type stubClassifier struct {
	cls model.Classification
	err error
}

func (s stubClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	return s.cls, s.err
}

// countingClassifier cuenta cuántas veces se llamó — usado para verificar
// qué links se siguen (o dejan de) invocar.
type countingClassifier struct {
	calls *int
	cls   model.Classification
	err   error
}

func (c countingClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	*c.calls++
	return c.cls, c.err
}

func TestChainClassifierUsesFirstOnSuccess(t *testing.T) {
	c := NewChainClassifier(
		stubClassifier{cls: model.Classification{Reason: "first"}},
		stubClassifier{cls: model.Classification{Reason: "second"}},
	)
	got, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reason != "first" {
		t.Errorf("expected first link's result, got %q", got.Reason)
	}
}

func TestChainClassifierFallsThroughOnError(t *testing.T) {
	c := NewChainClassifier(
		stubClassifier{err: errors.New("caído")},
		stubClassifier{err: errors.New("también caído")},
		stubClassifier{cls: model.Classification{Reason: "third"}},
	)
	got, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reason != "third" {
		t.Errorf("expected third link's result, got %q", got.Reason)
	}
}

func TestChainClassifierErrorsWhenAllFail(t *testing.T) {
	wantErr := errors.New("último error")
	c := NewChainClassifier(
		stubClassifier{err: errors.New("primero falló")},
		stubClassifier{err: wantErr},
	)
	_, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected last error to propagate, got %v", err)
	}
}

func TestChainClassifierStopsCallingDeadLinks(t *testing.T) {
	firstCalls, secondCalls := 0, 0
	c := NewChainClassifier(
		countingClassifier{calls: &firstCalls, err: errors.New("caído")},
		countingClassifier{calls: &secondCalls, cls: model.Classification{Reason: "second"}},
	)

	for i := 0; i < 5; i++ {
		got, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if got.Reason != "second" {
			t.Errorf("call %d: expected second link's result, got %q", i, got.Reason)
		}
	}

	if firstCalls != 1 {
		t.Errorf("expected first link called exactly once (dies after first failure), got %d calls", firstCalls)
	}
	if secondCalls != 5 {
		t.Errorf("expected second link called every time, got %d calls", secondCalls)
	}
}

func TestChainClassifierAllDeadReturnsFastWithoutCalling(t *testing.T) {
	calls := 0
	c := NewChainClassifier(countingClassifier{calls: &calls, err: errors.New("caído")})

	// primera llamada: banca el único link
	if _, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{}); err == nil {
		t.Fatal("expected error on first call")
	}
	// segunda llamada: no debería invocar el link de nuevo
	if _, err := c.Classify(context.Background(), model.Article{}, PrefilterResult{}); err == nil {
		t.Fatal("expected error on second call")
	}
	if calls != 1 {
		t.Errorf("expected the dead link to be called only once total, got %d calls", calls)
	}
}
