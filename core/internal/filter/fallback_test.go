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

// stubBatchClassifier implementa batchClassifier directamente, con fallos
// por título en vez de por posición — así sigue siendo válido aunque
// ChainClassifier le pase un subconjunto más chico en la segunda vuelta.
type stubBatchClassifier struct {
	calls   *int
	fail    map[string]bool
	failErr error
}

func (s stubBatchClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	if s.fail[a.Title] {
		return model.Classification{}, s.failErr
	}
	return model.Classification{Reason: "ok:" + a.Title}, nil
}

func (s stubBatchClassifier) ClassifyBatch(ctx context.Context, items []BatchItem) []BatchResult {
	if s.calls != nil {
		*s.calls++
	}
	out := make([]BatchResult, len(items))
	for i, it := range items {
		if s.fail[it.Article.Title] {
			out[i] = BatchResult{Err: s.failErr}
			continue
		}
		out[i] = BatchResult{Classification: model.Classification{Reason: "ok:" + it.Article.Title}}
	}
	return out
}

func TestClassifyBatchFallsBackToSequentialWhenNotSupported(t *testing.T) {
	c := stubClassifier{cls: model.Classification{Reason: "seq"}}
	items := []BatchItem{{Article: model.Article{Title: "a"}}, {Article: model.Article{Title: "b"}}}

	results := ClassifyBatch(context.Background(), c, items)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i, r := range results {
		if r.Err != nil || r.Classification.Reason != "seq" {
			t.Errorf("item %d: got %+v", i, r)
		}
	}
}

func TestChainClassifierBatchAllSucceed(t *testing.T) {
	calls := 0
	c := NewChainClassifier(stubBatchClassifier{calls: &calls})
	items := []BatchItem{{Article: model.Article{Title: "a"}}, {Article: model.Article{Title: "b"}}}

	results := c.ClassifyBatch(context.Background(), items)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("item %d: unexpected error %v", i, r.Err)
		}
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestChainClassifierBatchPartialFailureDoesNotKillLink(t *testing.T) {
	firstCalls, secondCalls := 0, 0
	link1 := stubBatchClassifier{calls: &firstCalls, fail: map[string]bool{"b": true}, failErr: errors.New("b falló")}
	link2 := stubBatchClassifier{calls: &secondCalls}
	c := NewChainClassifier(link1, link2)
	items := []BatchItem{{Article: model.Article{Title: "a"}}, {Article: model.Article{Title: "b"}}}

	results := c.ClassifyBatch(context.Background(), items)

	if results[0].Err != nil || results[0].Classification.Reason != "ok:a" {
		t.Errorf("item a: got %+v", results[0])
	}
	if results[1].Err != nil || results[1].Classification.Reason != "ok:b" {
		t.Errorf("item b: got %+v", results[1])
	}
	if firstCalls != 1 {
		t.Errorf("link1 calls = %d, want 1", firstCalls)
	}
	if secondCalls != 1 {
		t.Errorf("link2 calls = %d, want 1 (solo el item que falló en link1)", secondCalls)
	}

	// Un fallo parcial no debería bancar el link: en una segunda corrida
	// tiene que volver a intentarse.
	results2 := c.ClassifyBatch(context.Background(), items)
	if firstCalls != 2 {
		t.Errorf("link1 debería seguir consultándose tras un fallo parcial, calls=%d", firstCalls)
	}
	if results2[0].Classification.Reason != "ok:a" {
		t.Errorf("item a en segunda corrida: got %+v", results2[0])
	}
}

func TestChainClassifierBatchFullFailureMarksLinkDead(t *testing.T) {
	firstCalls, secondCalls := 0, 0
	link1 := stubBatchClassifier{calls: &firstCalls, fail: map[string]bool{"a": true, "b": true}, failErr: errors.New("caído")}
	link2 := stubBatchClassifier{calls: &secondCalls}
	c := NewChainClassifier(link1, link2)
	items := []BatchItem{{Article: model.Article{Title: "a"}}, {Article: model.Article{Title: "b"}}}

	c.ClassifyBatch(context.Background(), items)
	c.ClassifyBatch(context.Background(), items)

	if firstCalls != 1 {
		t.Errorf("link1 debería morir tras fallarle el batch completo una vez, calls=%d", firstCalls)
	}
	if secondCalls != 2 {
		t.Errorf("link2 debería atender las dos corridas, calls=%d", secondCalls)
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
