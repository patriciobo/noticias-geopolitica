package newsletter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

// maxVersionsPerCall limita el tamaño de cada llamada batch a Brevo.
// Defensivo: hoy la lista de suscriptores es chica, pero evita un límite
// sorpresa de la API más adelante sin tener que tocar este código.
const maxVersionsPerCall = 500

// Brevo manda el correo vía la API transaccional de Brevo — mismo patrón
// simple que el resto del proyecto (struct de request/response +
// http.Client sin SDK de terceros), como en internal/report/synthesize.go.
// Brevo se usa solo como motor de envío: la lista de suscriptores la
// administramos nosotros (Neon), no Brevo Contacts/Audiences.
type Brevo struct {
	APIKey      string
	SenderEmail string
	SenderName  string
	Client      *http.Client
}

func NewBrevo(apiKey, senderEmail, senderName string) *Brevo {
	return &Brevo{
		APIKey:      apiKey,
		SenderEmail: senderEmail,
		SenderName:  senderName,
		Client:      &http.Client{Timeout: 60 * time.Second},
	}
}

// Recipient es un destinatario ya resuelto: su email y el HTML final que le
// corresponde (con su propio link de unsubscribe ya embebido).
type Recipient struct {
	Email       string
	HTMLContent string
}

type brevoContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoRecipient struct {
	Email string `json:"email"`
}

type brevoMessageVersion struct {
	To          []brevoRecipient `json:"to"`
	HTMLContent string           `json:"htmlContent"`
}

type brevoRequest struct {
	Sender          brevoContact          `json:"sender"`
	Subject         string                `json:"subject"`
	HTMLContent     string                `json:"htmlContent"`
	MessageVersions []brevoMessageVersion `json:"messageVersions"`
}

type brevoErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Send manda subject+contenido a cada recipient, en batches de
// maxVersionsPerCall vía el mecanismo messageVersions de Brevo (una sola
// llamada HTTP por batch en vez de una por destinatario: menos round-trips
// y menor superficie de fallo parcial). Un batch que falla no aborta los
// siguientes — se cuentan enviados/fallidos y se devuelven ambos números
// junto con el primer error, para que el caller decida si loguear o fallar.
func (b *Brevo) Send(ctx context.Context, subject string, recipients []Recipient) (sent, failed int, err error) {
	if b.APIKey == "" {
		return 0, 0, fmt.Errorf("newsletter: no hay BREVO_API_KEY configurada")
	}

	var firstErr error
	for start := 0; start < len(recipients); start += maxVersionsPerCall {
		end := min(start+maxVersionsPerCall, len(recipients))
		batch := recipients[start:end]

		if sendErr := b.sendBatch(ctx, subject, batch); sendErr != nil {
			failed += len(batch)
			if firstErr == nil {
				firstErr = sendErr
			}
			continue
		}
		sent += len(batch)
	}
	return sent, failed, firstErr
}

func (b *Brevo) sendBatch(ctx context.Context, subject string, batch []Recipient) error {
	versions := make([]brevoMessageVersion, 0, len(batch))
	for _, r := range batch {
		versions = append(versions, brevoMessageVersion{
			To:          []brevoRecipient{{Email: r.Email}},
			HTMLContent: r.HTMLContent,
		})
	}

	reqBody := brevoRequest{
		Sender:  brevoContact{Email: b.SenderEmail, Name: b.SenderName},
		Subject: subject,
		// Fallback exigido por la API a nivel de request; cada messageVersion
		// lo pisa con su HTML personalizado (link de unsubscribe propio).
		HTMLContent:     batch[0].HTMLContent,
		MessageVersions: versions,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("newsletter: no se pudo serializar el request a Brevo: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoAPIURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", b.APIKey)
	req.Header.Set("accept", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("newsletter: error de red llamando a Brevo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		var be brevoErrorResponse
		_ = json.Unmarshal(body, &be)
		return fmt.Errorf("newsletter: brevo respondió %d: %s", resp.StatusCode, be.Message)
	}
	return nil
}
