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
	// ExtraHeaders manda headers adicionales en cada request — OpenRouter
	// recomienda HTTP-Referer/X-Title para identificarse en su free tier.
	ExtraHeaders map[string]string
}

func NewOpenAICompatSynthesizer(baseURL, apiKey, modelName string) *OpenAICompatSynthesizer {
	return &OpenAICompatSynthesizer{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Model:   modelName,
		Client:  &http.Client{Timeout: 300 * time.Second},
	}
}

// maxRateLimitRetries and rateLimitBackoff handle transient failures (rate
// limits and 502/503 server overload) — same policy as
// filter.OpenAICompatClassifier.
const maxRateLimitRetries = 6

var rateLimitBackoff = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second, 30 * time.Second, 30 * time.Second}

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusBadGateway || code == http.StatusServiceUnavailable
}

// isTransientErrorMessage detecta errores conocidos como transitorios
// aunque vengan con un status HTTP que no dispara retry por sí solo — mismo
// criterio que filter.OpenAICompatClassifier (ver ahí el porqué).
func isTransientErrorMessage(msg string) bool {
	lower := strings.ToLower(msg)
	for _, phrase := range []string{"provider returned error", "overloaded", "try again", "timeout", "internal server error"} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// extractErrorMessage saca el texto de error de un body de respuesta, en
// cualquiera de los dos formatos que devuelven los proveedores compatibles.
func extractErrorMessage(body []byte) string {
	var cr compatChatResponse
	if json.Unmarshal(body, &cr) == nil && cr.Error != nil {
		return cr.Error.Message
	}
	var arr []compatChatResponse
	if json.Unmarshal(body, &arr) == nil && len(arr) > 0 && arr[0].Error != nil {
		return arr[0].Error.Message
	}
	return ""
}

// isHardQuotaExceeded distingue un 429 de cuota de plan agotada (diaria o
// mensual, no se resetea en segundos) de un 429 de burst momentáneo (se
// resetea solo) — mismo criterio que filter.OpenAICompatClassifier.
func isHardQuotaExceeded(body []byte) bool {
	return bytes.Contains(bytes.ToLower(body), []byte("check your plan and billing"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

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
		for k, v := range s.ExtraHeaders {
			req.Header.Set(k, v)
		}

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

		if statusCode == http.StatusTooManyRequests && isHardQuotaExceeded(body) {
			break
		}
		retryable := isRetryableStatus(statusCode) || isTransientErrorMessage(extractErrorMessage(body))
		if !retryable || attempt >= maxRateLimitRetries {
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
			return "", fmt.Errorf("openai-compat synthesizer: %s", arr[0].Error.Message)
		}
		return "", fmt.Errorf("openai-compat synthesizer: respuesta inesperada (status %d): %s", statusCode, truncate(string(body), 300))
	}
	if cr.Error != nil {
		return "", fmt.Errorf("openai-compat synthesizer: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("openai-compat synthesizer: respuesta vacía")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}
