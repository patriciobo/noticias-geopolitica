package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaSynthesizer is the free/local alternative to ClaudeSynthesizer:
// same three-section report, backed by a model running in a local Ollama
// daemon. No API key, no network egress, no per-run cost.
type OllamaSynthesizer struct {
	BaseURL string
	Model   string // e.g. "qwen3.5:9b"
	Client  *http.Client
}

func NewOllamaSynthesizer(baseURL, modelName string) *OllamaSynthesizer {
	return &OllamaSynthesizer{
		BaseURL: baseURL,
		Model:   modelName,
		Client:  &http.Client{Timeout: 300 * time.Second},
	}
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Think    bool            `json:"think"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error"`
}

func (s *OllamaSynthesizer) Synthesize(ctx context.Context, in Input) (string, error) {
	if len(in.Items) == 0 {
		return "", fmt.Errorf("ollama synthesizer: no classified articles to synthesize")
	}

	reqBody := ollamaChatRequest{
		Model: s.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: synthesisSystemPrompt},
			{Role: "user", Content: buildUserPrompt(in)},
		},
		Stream:  false,
		Think:   false,
		Options: map[string]any{"temperature": 0.3},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo conectar con Ollama en %s (¿está corriendo `ollama serve`?): %w", s.BaseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}

	var cr ollamaChatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("respuesta inesperada de Ollama: %w", err)
	}
	if cr.Error != "" {
		return "", fmt.Errorf("ollama: %s (¿corriste `ollama pull %s`?)", cr.Error, s.Model)
	}
	return strings.TrimSpace(cr.Message.Content), nil
}
