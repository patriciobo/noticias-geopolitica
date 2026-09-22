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

// OpenAICompatClassifier works with any provider that speaks the OpenAI
// Chat Completions API shape — OpenAI itself, DeepSeek, and Gemini via its
// /v1beta/openai/ compatibility endpoint all qualify. One client, pick the
// vendor via BaseURL/APIKey/Model.
type OpenAICompatClassifier struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewOpenAICompatClassifier(baseURL, apiKey, modelName string) *OpenAICompatClassifier {
	return &OpenAICompatClassifier{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Model:   modelName,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type compatChatRequest struct {
	Model          string          `json:"model"`
	Messages       []compatMessage `json:"messages"`
	ResponseFormat *compatRespFmt  `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type compatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type compatRespFmt struct {
	Type string `json:"type"`
}

type compatChatResponse struct {
	Choices []struct {
		Message compatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenAICompatClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	userMsg := fmt.Sprintf("Título: %s\nSnippet: %s", a.Title, a.Snippet)

	reqBody := compatChatRequest{
		Model: c.Model,
		Messages: []compatMessage{
			{Role: "system", Content: classifySystemPrompt},
			{Role: "user", Content: userMsg},
		},
		ResponseFormat: &compatRespFmt{Type: "json_object"},
		Temperature:    0,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return model.Classification{}, fmt.Errorf("openai-compat classifier (%s): %w", c.Model, err)
	}

	var cls model.Classification
	if err := json.Unmarshal([]byte(text), &cls); err != nil {
		return model.Classification{}, fmt.Errorf("openai-compat classifier: could not parse model output as JSON: %w (output: %s)", err, truncate(text, 300))
	}
	return cls, nil
}

// maxRateLimitRetries and rateLimitBackoff handle transient failures: rate
// limits (Gemini's free tier RPM is low enough that a burst of concurrent
// classify calls routinely trips 429s) and server-side overload (502/503,
// which Gemini also returns under load) — retry with backoff instead of
// dropping the article silently.
const maxRateLimitRetries = 6

var rateLimitBackoff = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second, 30 * time.Second, 30 * time.Second}

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusBadGateway || code == http.StatusServiceUnavailable
}

// isHardQuotaExceeded distingue un 429 de cuota de plan agotada (diaria o
// mensual, no se resetea en segundos) de un 429 de burst momentáneo (se
// resetea solo). Reintentar con backoff no sirve para el primer caso —
// Gemini frasea ese error con "check your plan and billing".
func isHardQuotaExceeded(body []byte) bool {
	return bytes.Contains(bytes.ToLower(body), []byte("check your plan and billing"))
}

func (c *OpenAICompatClassifier) chat(ctx context.Context, reqBody compatChatRequest) (string, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	var body []byte
	var statusCode int
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, err := c.Client.Do(req)
		if err != nil {
			return "", err
		}
		statusCode = resp.StatusCode
		body, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if err != nil {
			return "", err
		}

		if statusCode == http.StatusTooManyRequests && isHardQuotaExceeded(body) {
			break
		}
		if !isRetryableStatus(statusCode) || attempt >= maxRateLimitRetries {
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
		// Gemini a veces envuelve el error en un array ([{"error": {...}}])
		// en vez del objeto plano que usa el resto — probamos esa forma
		// antes de rendirnos a un mensaje genérico.
		var arr []compatChatResponse
		if json.Unmarshal(body, &arr) == nil && len(arr) > 0 && arr[0].Error != nil {
			return "", fmt.Errorf("%s", arr[0].Error.Message)
		}
		return "", fmt.Errorf("respuesta inesperada (status %d): %s", statusCode, truncate(string(body), 300))
	}
	if cr.Error != nil {
		return "", fmt.Errorf("%s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("respuesta vacía (status %d): %s", statusCode, truncate(string(body), 300))
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}
