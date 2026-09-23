package main

import (
	"encoding/json"
	"net/http"

	"noticias/core/internal/subscriber"
)

type subscribeRequest struct {
	Email string `json:"email"`
}

func handleSubscribe(repo *subscriber.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req subscribeRequest
		r.Body = http.MaxBytesReader(w, r.Body, 1<<10) // 1KB alcanza de sobra para {"email":"..."}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"body inválido"}`, http.StatusBadRequest)
			return
		}

		if _, err := repo.Subscribe(r.Context(), req.Email); err != nil {
			if err == subscriber.ErrInvalidEmail {
				http.Error(w, `{"error":"email inválido"}`, http.StatusBadRequest)
				return
			}
			http.Error(w, `{"error":"no se pudo procesar la suscripción"}`, http.StatusInternalServerError)
			return
		}

		// Respuesta uniforme siempre 200, sin distinguir nuevo/ya activo/
		// reactivado — evita que el endpoint sirva para enumerar qué
		// emails ya están suscriptos.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

func handleUnsubscribe(repo *subscriber.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != "" {
			// Idempotente: token inválido o ya dado de baja se trata igual
			// que éxito, para no filtrar validez de tokens ajenos.
			_ = repo.Unsubscribe(r.Context(), token)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(unsubscribeConfirmationHTML))
	}
}

const unsubscribeConfirmationHTML = `<!doctype html>
<html lang="es"><head><meta charset="utf-8"><title>Baja confirmada</title></head>
<body style="font-family: sans-serif; max-width: 32rem; margin: 4rem auto; padding: 0 1rem;">
<h1>Listo</h1>
<p>Ya no vas a recibir más correos de Noticias Internacionales.</p>
</body></html>`
