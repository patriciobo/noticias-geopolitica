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

// OllamaClassifier is the free/local alternative to ClaudeClassifier: same
// interface, same prompt intent, backed by a model running in a local
// Ollama daemon instead of a paid API. No API key, no network egress.
type OllamaClassifier struct {
	BaseURL string // e.g. "http://localhost:11434"
	Model   string // e.g. "qwen3:1.7b"
	Client  *http.Client
}

func NewOllamaClassifier(baseURL, modelName string) *OllamaClassifier {
	return &OllamaClassifier{
		BaseURL: baseURL,
		Model:   modelName,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format,omitempty"`
	Think    bool            `json:"think"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error"`
}

func (c *OllamaClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	userMsg := fmt.Sprintf("Título: %s\nSnippet: %s", a.Title, a.Snippet)

	reqBody := ollamaChatRequest{
		Model: c.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: classifySystemPrompt},
			{Role: "user", Content: userMsg},
		},
		Stream:  false,
		Format:  "json",
		Think:   false,
		Options: map[string]any{"temperature": 0},
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return model.Classification{}, fmt.Errorf("ollama classifier: %w", err)
	}

	var cls model.Classification
	if err := json.Unmarshal([]byte(text), &cls); err != nil {
		return model.Classification{}, fmt.Errorf("ollama classifier: could not parse model output as JSON: %w (output: %s)", err, truncate(text, 300))
	}
	return cls, nil
}

func (c *OllamaClassifier) chat(ctx context.Context, reqBody ollamaChatRequest) (string, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo conectar con Ollama en %s (¿está corriendo `ollama serve`?): %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}

	var cr ollamaChatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("respuesta inesperada de Ollama: %w", err)
	}
	if cr.Error != "" {
		return "", fmt.Errorf("ollama: %s (¿corriste `ollama pull %s`?)", cr.Error, reqBody.Model)
	}
	return strings.TrimSpace(cr.Message.Content), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (c *OllamaClassifier) ModelName() string { return c.Model }
