package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"noticias/core/internal/newsletter"
	"noticias/core/internal/subscriber"
)

// subscriptions agrupa lo que necesitan los endpoints del newsletter.
type subscriptions struct {
	repo  *subscriber.Repository
	brevo *newsletter.Brevo

	apiBaseURL string // base de los links de confirmación/baja
	siteURL    string

	// internalSecret, si está seteado, es obligatorio en POST /subscribers
	// (header X-Internal-Secret). Lo manda la route /api/subscribe de
	// Next.js: sin él, cualquiera podría pegarle directo a Render
	// salteándose el captcha y el rate limit del lado de Vercel.
	internalSecret string

	enabled     bool          // SUBSCRIPTIONS_ENABLED=false corta las altas nuevas (kill switch)
	dailyCap    int           // tope global de mails de confirmación por 24h
	resendAfter time.Duration // mínimo entre dos mails de confirmación al mismo email
}

type subscribeRequest struct {
	Email string `json:"email"`
}

const okJSON = `{"status":"ok"}`

func (s *subscriptions) checkSecret(r *http.Request) bool {
	if s.internalSecret == "" {
		return true
	}
	got := r.Header.Get("X-Internal-Secret")
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.internalSecret)) == 1
}

// subscribeKey identifica al cliente para el rate limit de altas. Si el
// request viene autenticado desde Next.js, la IP real del navegador está
// en X-Client-IP (desde Render todos los pedidos parecen venir de Vercel);
// sin el secreto ese header no se cree, lo podría inventar cualquiera.
func (s *subscriptions) subscribeKey(r *http.Request) string {
	if s.internalSecret != "" && s.checkSecret(r) {
		if ip := r.Header.Get("X-Client-IP"); ip != "" {
			return ip
		}
	}
	return clientIP(r)
}

func (s *subscriptions) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if !s.checkSecret(r) {
		http.Error(w, `{"error":"prohibido"}`, http.StatusForbidden)
		return
	}
	if !s.enabled {
		http.Error(w, `{"error":"las suscripciones están pausadas"}`, http.StatusServiceUnavailable)
		return
	}

	var req subscribeRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10) // 1KB alcanza de sobra para {"email":"..."}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"body inválido"}`, http.StatusBadRequest)
		return
	}
	if _, err := subscriber.NormalizeEmail(req.Email); err != nil {
		http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
		return
	}

	sent, err := s.repo.ConfirmationsSentSince(r.Context(), 24*time.Hour)
	if err != nil {
		log.Printf("subscribe: %v", err)
		http.Error(w, `{"error":"no se pudo procesar la suscripción"}`, http.StatusInternalServerError)
		return
	}
	if sent >= s.dailyCap {
		// Pico de altas (o abuso repartido entre muchas IPs): se corta acá
		// para no quemar la cuota diaria de Brevo que necesita el newsletter.
		log.Printf("subscribe: tope diario de confirmaciones alcanzado (%d)", s.dailyCap)
		http.Error(w, `{"error":"demasiadas altas hoy, probá mañana"}`, http.StatusServiceUnavailable)
		return
	}

	sub, outcome, err := s.repo.Subscribe(r.Context(), req.Email, s.resendAfter)
	if err != nil {
		log.Printf("subscribe: %v", err)
		http.Error(w, `{"error":"no se pudo procesar la suscripción"}`, http.StatusInternalServerError)
		return
	}

	if outcome == subscriber.OutcomeNeedsConfirmation && sub.ConfirmToken != nil {
		// En segundo plano: la respuesta sale igual de rápido haya o no
		// mail que mandar, así el tiempo de respuesta no delata si el
		// email ya estaba suscripto.
		go s.sendConfirmation(sub.Email, *sub.ConfirmToken)
	}

	// Respuesta uniforme siempre 200, sin distinguir nuevo/ya activo/
	// pendiente — evita que el endpoint sirva para enumerar qué emails ya
	// están suscriptos.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(okJSON))
}

func (s *subscriptions) sendConfirmation(email, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	confirmURL := s.apiBaseURL + "/subscribers/confirm?token=" + url.QueryEscape(token)
	html, err := newsletter.RenderConfirmEmail(s.siteURL, confirmURL)
	if err != nil {
		log.Printf("subscribe: %v", err)
		return
	}
	if err := s.brevo.SendOne(ctx, email, newsletter.ConfirmSubject, html); err != nil {
		log.Printf("subscribe: no se pudo mandar el mail de confirmación: %v", err)
	}
}

// Confirmación y baja son dos pasos: el link del mail (GET) solo muestra un
// botón, y la acción ocurre con el POST del form. Los filtros antispam y
// antivirus corporativos abren todos los links de un mail para
// inspeccionarlos — si el GET confirmara o diera de baja, lo harían solos.

func (s *subscriptions) handleConfirmPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, http.StatusOK, pageData{
		Title:  "Confirmá tu suscripción",
		Body:   "Un click más y empezás a recibir el resumen diario de Radar Global.",
		Action: "/subscribers/confirm",
		Button: "Confirmar suscripción",
		Token:  r.URL.Query().Get("token"),
	})
}

func (s *subscriptions) handleConfirm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	token := r.FormValue("token")
	ok := false
	if token != "" {
		var err error
		ok, err = s.repo.Confirm(r.Context(), token)
		if err != nil {
			log.Printf("confirm: %v", err)
			renderPage(w, http.StatusInternalServerError, pageData{Title: "Algo falló", Body: "No se pudo confirmar. Probá de nuevo en un rato."})
			return
		}
	}
	if !ok {
		renderPage(w, http.StatusOK, pageData{
			Title: "Enlace vencido o ya usado",
			Body:  "Si ya confirmaste, no hace falta nada más. Si no, volvé a anotarte desde el sitio.",
			Link:  s.siteURL,
		})
		return
	}
	renderPage(w, http.StatusOK, pageData{
		Title: "¡Listo!",
		Body:  "Tu suscripción quedó confirmada. Vas a recibir el próximo resumen por email.",
		Link:  s.siteURL,
	})
}

func (s *subscriptions) handleUnsubscribePage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, http.StatusOK, pageData{
		Title:  "Darte de baja",
		Body:   "¿Querés dejar de recibir el resumen diario de Radar Global?",
		Action: "/subscribers/unsubscribe",
		Button: "Sí, darme de baja",
		Token:  r.URL.Query().Get("token"),
	})
}

// handleUnsubscribe atiende tanto el form de la página de baja como el
// "one-click" de RFC 8058 (el botón nativo de Gmail/Yahoo/Apple Mail hace
// POST a la URL de List-Unsubscribe, con el token en el query string).
// r.FormValue lee de los dos lados.
func (s *subscriptions) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	if token := r.FormValue("token"); token != "" {
		// Idempotente: token inválido o ya dado de baja se trata igual
		// que éxito, para no filtrar validez de tokens ajenos.
		if err := s.repo.Unsubscribe(r.Context(), token); err != nil {
			log.Printf("unsubscribe: %v", err)
			renderPage(w, http.StatusInternalServerError, pageData{Title: "Algo falló", Body: "No se pudo procesar la baja. Probá de nuevo en un rato."})
			return
		}
	}
	renderPage(w, http.StatusOK, pageData{
		Title: "Listo",
		Body:  "Ya no vas a recibir más correos de Radar Global.",
	})
}

type pageData struct {
	Title  string
	Body   string
	Action string // si no está vacío, se muestra un form POST con el token
	Button string
	Token  string
	Link   string
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex"><title>{{.Title}}</title></head>
<body style="font-family: Georgia, serif; max-width: 32rem; margin: 4rem auto; padding: 0 1rem; color:#171717;">
<h1>{{.Title}}</h1>
<p>{{.Body}}</p>
{{if .Action}}<form method="post" action="{{.Action}}">
<input type="hidden" name="token" value="{{.Token}}">
<button type="submit" style="background:#c81e1e; color:#fff; border:0; padding:12px 22px; font-size:16px; font-weight:bold; cursor:pointer;">{{.Button}}</button>
</form>{{end}}
{{if .Link}}<p><a href="{{.Link}}" style="color:#c81e1e;">Ir a Radar Global</a></p>{{end}}
</body></html>`))

func renderPage(w http.ResponseWriter, status int, d pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, d); err != nil {
		log.Printf("render page: %v", err)
	}
}
