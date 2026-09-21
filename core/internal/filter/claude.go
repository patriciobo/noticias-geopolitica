package filter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"noticias/core/internal/model"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"

// ClaudeClassifier calls the Claude API to make the precise international/
// multinational relevance call. Only config needed is the API key, read
// from the environment by the caller — never hardcoded, never logged.
type ClaudeClassifier struct {
	APIKey string
	Model  string // e.g. "claude-haiku-4-5-20251001"
	Client *http.Client
}

func NewClaudeClassifier(apiKey string) *ClaudeClassifier {
	return &ClaudeClassifier{
		APIKey: apiKey,
		Model:  "claude-haiku-4-5-20251001",
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const classifySystemPrompt = `Sos un clasificador de noticias. Te paso un titular y un snippet.
Respondé EXCLUSIVAMENTE con un objeto JSON (sin texto adicional, sin markdown) con esta forma exacta:
{
  "is_international": boolean,
  "countries": ["nombre de país", ...],
  "companies": ["nombre de empresa", ...],
  "relation_type": "treaty" | "trade" | "sanction" | "supply_chain" | "regulatory" | "none",
  "confidence": number entre 0 y 1,
  "reason": "una oración breve en español formal de Argentina (voseo: 'vos', 'tenés', etc. — nunca 'tú'; registro profesional, sin modismos coloquiales)"
}
Marcá is_international=true solo si la noticia afecta o involucra relaciones entre países (tratados, sanciones, comercio exterior, geopolítica) o empresas multinacionales (locales o extranjeras operando en múltiples países). Noticia puramente doméstica (política interna, sociedad, deportes locales) es is_international=false.`

func (c *ClaudeClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	if c.APIKey == "" {
		return model.Classification{}, fmt.Errorf("claude classifier: no API key configured")
	}

	userMsg := fmt.Sprintf("Título: %s\nSnippet: %s", a.Title, a.Snippet)

	reqBody := claudeRequest{
		Model:     c.Model,
		MaxTokens: 400,
		System:    classifySystemPrompt,
		Messages:  []claudeMessage{{Role: "user", Content: userMsg}},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return model.Classification{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(payload))
	if err != nil {
		return model.Classification{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.Client.Do(req)
	if err != nil {
		return model.Classification{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.Classification{}, err
	}

	var cr claudeResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return model.Classification{}, fmt.Errorf("claude classifier: bad response: %w", err)
	}
	if cr.Error != nil {
		return model.Classification{}, fmt.Errorf("claude classifier: api error: %s", cr.Error.Message)
	}
	if len(cr.Content) == 0 {
		return model.Classification{}, fmt.Errorf("claude classifier: empty response")
	}

	text := strings.TrimSpace(cr.Content[0].Text)
	var cls model.Classification
	if err := json.Unmarshal([]byte(text), &cls); err != nil {
		return model.Classification{}, fmt.Errorf("claude classifier: could not parse model output as JSON: %w", err)
	}
	return cls, nil
}
