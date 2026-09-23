package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"noticias/core/internal/model"
)

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain json", `{"a":1}`, `{"a":1}`},
		{"markdown fences", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"bare fences", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"leading prose", `Claro, acá está: {"a":1}`, `{"a":1}`},
		{"trailing prose", `{"a":1} espero que ayude`, `{"a":1}`},
		{"no braces", "sin json acá", "sin json acá"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractJSONObject(c.in); got != c.want {
				t.Errorf("extractJSONObject(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestIsTransientErrorMessage(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"Provider returned error", true},
		{"upstream request timeout", true},
		{"Model is overloaded, try again later", true},
		{"Internal Server Error", true},
		{"", false},
		{"you exceeded your current quota, check your plan and billing", false},
		{"invalid API key", false},
	}
	for _, c := range cases {
		if got := isTransientErrorMessage(c.msg); got != c.want {
			t.Errorf("isTransientErrorMessage(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestExtractErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"plain object", `{"error":{"message":"Provider returned error"}}`, "Provider returned error"},
		{"wrapping array", `[{"error":{"message":"boom"}}]`, "boom"},
		{"no error", `{"choices":[{"message":{"content":"ok"}}]}`, ""},
		{"not json", `not json at all`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractErrorMessage([]byte(c.body)); got != c.want {
				t.Errorf("extractErrorMessage(%q) = %q, want %q", c.body, got, c.want)
			}
		})
	}
}

// TestOpenAICompatClassifierClassifyBatch cubre el camino real contra HTTP:
// la primera respuesta (el batch) viene con un índice faltante a propósito
// — el segundo request tiene que ser el top-up individual de ESE item
// puntual, no un reintento del batch entero.
func TestOpenAICompatClassifierClassifyBatch(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		var req compatChatRequest
		_ = json.Unmarshal(body, &req)

		var content string
		switch requests {
		case 1:
			if req.Messages[0].Content != batchClassifySystemPrompt {
				t.Errorf("request 1 debería usar el prompt de batch")
			}
			content = `[{"index":0,"is_international":true,"reason":"a internacional"}]`
		case 2:
			if req.Messages[0].Content != classifySystemPrompt {
				t.Errorf("request 2 (top-up) debería usar el prompt individual")
			}
			content = `{"is_international":false,"reason":"b domestico"}`
		default:
			t.Fatalf("request inesperado número %d", requests)
		}

		encoded, _ := json.Marshal(content)
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%s}}]}`, encoded)
	}))
	defer server.Close()

	c := NewOpenAICompatClassifier(server.URL, "test-key", "test-model")
	items := []BatchItem{
		{Article: model.Article{Title: "a"}},
		{Article: model.Article{Title: "b"}},
	}

	results := c.ClassifyBatch(context.Background(), items)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Err != nil || !results[0].Classification.IsInternational {
		t.Errorf("item a: got %+v", results[0])
	}
	if results[1].Err != nil || results[1].Classification.IsInternational {
		t.Errorf("item b (top-up): got %+v", results[1])
	}
	if requests != 2 {
		t.Errorf("expected 2 requests (1 batch + 1 top-up), got %d", requests)
	}
}
