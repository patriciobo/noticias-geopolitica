package newsletter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

// sendConcurrency limita cuántos envíos a Brevo corren en paralelo. Con el
// tope del plan gratis (300/día) la lista nunca es enorme; unos pocos en
// paralelo alcanzan sin acercarse a los rate limits de la API.
const sendConcurrency = 4

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

// Recipient es un destinatario ya resuelto: su email, el HTML final que le
// corresponde (con su propio link de unsubscribe ya embebido) y la URL de
// baja en un click para los headers List-Unsubscribe.
type Recipient struct {
	Email          string
	HTMLContent    string
	UnsubscribeURL string
}

type brevoContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoRecipient struct {
	Email string `json:"email"`
}

type brevoRequest struct {
	Sender      brevoContact      `json:"sender"`
	To          []brevoRecipient  `json:"to"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"htmlContent"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type brevoErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// unsubscribeHeaders arma los headers de baja en un click (RFC 2369 +
// RFC 8058). Gmail y Yahoo los exigen a remitentes masivos, y los clientes
// muestran un botón "Desuscribirse" nativo — mucho mejor que un reporte de
// spam, que es lo que hace la gente cuando no encuentra cómo darse de baja.
func unsubscribeHeaders(unsubscribeURL string) map[string]string {
	if unsubscribeURL == "" {
		return nil
	}
	return map[string]string{
		"List-Unsubscribe":      "<" + unsubscribeURL + ">",
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
	}
}

// Send manda subject+contenido a cada recipient, un request por
// destinatario: los headers List-Unsubscribe llevan el token de cada uno y
// Brevo no permite headers distintos por messageVersion. Un envío que
// falla no aborta los demás — se cuentan enviados/fallidos y se devuelven
// ambos números junto con el primer error, para que el caller decida si
// loguear o fallar.
func (b *Brevo) Send(ctx context.Context, subject string, recipients []Recipient) (sent, failed int, err error) {
	if b.APIKey == "" {
		return 0, 0, fmt.Errorf("newsletter: no hay BREVO_API_KEY configurada")
	}

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		sem      = make(chan struct{}, sendConcurrency)
	)
	for _, r := range recipients {
		wg.Add(1)
		sem <- struct{}{}
		go func(r Recipient) {
			defer wg.Done()
			defer func() { <-sem }()
			sendErr := b.send(ctx, r.Email, subject, r.HTMLContent, unsubscribeHeaders(r.UnsubscribeURL))
			mu.Lock()
			defer mu.Unlock()
			if sendErr != nil {
				failed++
				if firstErr == nil {
					firstErr = sendErr
				}
				return
			}
			sent++
		}(r)
	}
	wg.Wait()
	return sent, failed, firstErr
}

// SendOne manda un único mail transaccional (ej. la confirmación de alta).
func (b *Brevo) SendOne(ctx context.Context, to, subject, html string) error {
	if b.APIKey == "" {
		return fmt.Errorf("newsletter: no hay BREVO_API_KEY configurada")
	}
	return b.send(ctx, to, subject, html, nil)
}

func (b *Brevo) send(ctx context.Context, to, subject, html string, headers map[string]string) error {
	reqBody := brevoRequest{
		Sender:      brevoContact{Email: b.SenderEmail, Name: b.SenderName},
		To:          []brevoRecipient{{Email: to}},
		Subject:     subject,
		HTMLContent: html,
		Headers:     headers,
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
