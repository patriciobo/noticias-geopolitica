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

	"noticias/core/internal/model"
)

// OpenAICompatSynthesizer mirrors OpenAICompatClassifier: works with any
// provider speaking the OpenAI Chat Completions shape (OpenAI, DeepSeek,
// Gemini via /v1beta/openai/).
type OpenAICompatSynthesizer struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewOpenAICompatSynthesizer(baseURL, apiKey, modelName string) *OpenAICompatSynthesizer {
	return &OpenAICompatSynthesizer{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Model:   modelName,
		Client:  &http.Client{Timeout: 300 * time.Second},
	}
}

// maxRateLimitRetries and rateLimitBackoff handle free-tier rate limits —
// same policy as filter.OpenAICompatClassifier.
const maxRateLimitRetries = 6

var rateLimitBackoff = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second, 30 * time.Second, 30 * time.Second}

type compatChatRequest struct {
	Model       string          `json:"model"`
	Messages    []compatMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type compatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type compatChatResponse struct {
	Choices []struct {
		Message compatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *OpenAICompatSynthesizer) Synthesize(ctx context.Context, items []model.ClassifiedArticle) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("openai-compat synthesizer: no classified articles to synthesize")
	}

	reqBody := compatChatRequest{
		Model: s.Model,
		Messages: []compatMessage{
			{Role: "system", Content: synthesisSystemPrompt},
			{Role: "user", Content: buildUserPrompt(items)},
		},
		Temperature: 0.3,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	var body []byte
	var statusCode int
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/chat/completions", bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.APIKey)

		resp, err := s.Client.Do(req)
		if err != nil {
			return "", fmt.Errorf("openai-compat synthesizer (%s): %w", s.Model, err)
		}
		statusCode = resp.StatusCode
		body, err = io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return "", err
		}

		if statusCode != http.StatusTooManyRequests || attempt >= maxRateLimitRetries {
			break
		}
		select {
		case <-time.After(rateLimitBackoff[attempt]):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	var cr compatChatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("openai-compat synthesizer: respuesta inesperada (status %d)", statusCode)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("openai-compat synthesizer: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("openai-compat synthesizer: respuesta vacía")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}
