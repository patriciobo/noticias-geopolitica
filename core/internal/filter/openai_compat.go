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
	// ExtraHeaders manda headers adicionales en cada request — OpenRouter
	// recomienda HTTP-Referer/X-Title para identificarse en su free tier.
	ExtraHeaders map[string]string
	// DisableJSONMode saltea el parámetro response_format: json_object.
	// Varios modelos free chicos de OpenRouter no lo soportan y tiran
	// "Provider returned error" en vez de ignorarlo. El prompt ya le pide
	// JSON puro por texto, así que funciona igual sin el parámetro — solo
	// se pierde la garantía dura del formato, compensada abajo con un
	// parseo más tolerante.
	DisableJSONMode bool
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
		Temperature: 0,
	}
	if !c.DisableJSONMode {
		reqBody.ResponseFormat = &compatRespFmt{Type: "json_object"}
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return model.Classification{}, fmt.Errorf("openai-compat classifier (%s): %w", c.Model, err)
	}

	var cls model.Classification
	if err := json.Unmarshal([]byte(text), &cls); err != nil {
		// Sin response_format forzado, un modelo puede envolver el JSON en
		// prosa o en fences de markdown pese a la instrucción — probamos
		// extraer el objeto antes de rendirnos.
		if err2 := json.Unmarshal([]byte(extractJSONObject(text)), &cls); err2 == nil {
			return cls, nil
		}
		return model.Classification{}, fmt.Errorf("openai-compat classifier: could not parse model output as JSON: %w (output: %s)", err, truncate(text, 300))
	}
	return cls, nil
}

// ClassifyBatch manda todos los items en UN solo request (numerados, ver
// buildBatchUserPrompt) en vez de uno por artículo — mismo trabajo pedido
// al modelo, muchas menos requests, que es lo que de verdad pisa los
// límites por minuto de los free tiers (no el volumen total del día).
//
// Devuelve siempre len(items) resultados. Si el modelo se saltea algún
// índice en la respuesta (raro, pero no hay garantía dura sin
// response_format en varios proveedores free), esos puntuales se completan
// con Classify() uno por uno en vez de perderlos en silencio — no vale la
// pena repetir el batch entero por uno o dos items.
func (c *OpenAICompatClassifier) ClassifyBatch(ctx context.Context, items []BatchItem) []BatchResult {
	results := make([]BatchResult, len(items))
	if len(items) == 0 {
		return results
	}

	reqBody := compatChatRequest{
		Model: c.Model,
		Messages: []compatMessage{
			{Role: "system", Content: batchClassifySystemPrompt},
			{Role: "user", Content: buildBatchUserPrompt(items)},
		},
		Temperature: 0,
	}
	// Sin response_format acá a propósito: la respuesta es un array
	// top-level, no un objeto — json_object de la API forzaría justo la
	// forma que no queremos.

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		wrapped := fmt.Errorf("openai-compat classifier batch (%s): %w", c.Model, err)
		for i := range results {
			results[i] = BatchResult{Err: wrapped}
		}
		return results
	}

	var parsed []batchClassificationItem
	if jsonErr := json.Unmarshal([]byte(text), &parsed); jsonErr != nil {
		if jsonErr2 := json.Unmarshal([]byte(extractJSONArray(text)), &parsed); jsonErr2 != nil {
			wrapped := fmt.Errorf("openai-compat classifier batch: no se pudo parsear la respuesta como array JSON: %w (output: %s)", jsonErr, truncate(text, 300))
			for i := range results {
				results[i] = BatchResult{Err: wrapped}
			}
			return results
		}
	}

	byIndex := make(map[int]model.Classification, len(parsed))
	for _, p := range parsed {
		byIndex[p.Index] = p.Classification
	}

	var missing []int
	for i := range items {
		if cls, ok := byIndex[i]; ok {
			results[i] = BatchResult{Classification: cls}
		} else {
			missing = append(missing, i)
		}
	}
	for _, i := range missing {
		cls, err := c.Classify(ctx, items[i].Article, items[i].Pre)
		results[i] = BatchResult{Classification: cls, Err: err}
	}

	return results
}

// batchClassificationItem embebe model.Classification: el JSON del batch
// trae los mismos campos que la clasificación individual, más "index" para
// poder reconciliar sin depender del orden de la respuesta.
type batchClassificationItem struct {
	Index int `json:"index"`
	model.Classification
}

// extractJSONObject pela fences de markdown (```json ... ```) y se queda
// con la primera llave abierta hasta la última cerrada — suficiente para
// rescatar un objeto JSON que vino acompañado de texto que no se pidió.
func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
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

// isTransientErrorMessage detecta errores conocidos como transitorios
// aunque vengan con un status HTTP que no dispara retry por sí solo — el
// caso real que motivó esto: OpenRouter devuelve "Provider returned error"
// cuando el modelo free de turno tuvo un hipo puntual (sobrecarga del lado
// del proveedor upstream), no necesariamente con status 429/502/503.
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
// cualquiera de los dos formatos que devuelven los proveedores compatibles
// (objeto plano o array envolvente) — "" si no hay ninguno reconocible.
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
		for k, v := range c.ExtraHeaders {
			req.Header.Set(k, v)
		}

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
