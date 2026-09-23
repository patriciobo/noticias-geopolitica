package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter es un token bucket por clave (IP) en memoria. Alcanza para
// una sola instancia de Render: no hace falta Redis para frenar a un bot
// que martilla el endpoint. Si algún día corren varias instancias, cada
// una tiene su propio límite (se multiplica por N) — sigue acotado.
type rateLimiter struct {
	mu      sync.Mutex
	rate    float64 // tokens por segundo
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// newRateLimiter permite `count` pedidos por `per` en régimen sostenido,
// con ráfagas de hasta `burst`.
func newRateLimiter(count int, per time.Duration, burst int) *rateLimiter {
	return &rateLimiter{
		rate:    float64(count) / per.Seconds(),
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}
	b.tokens = min(rl.burst, b.tokens+now.Sub(b.last).Seconds()*rl.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep borra buckets que ya se rellenaron del todo — equivalen a uno
// nuevo, así que no hace falta recordarlos. Evita que el map crezca sin
// límite con IPs que pasaron una sola vez.
func (rl *rateLimiter) sweep() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	for k, b := range rl.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*rl.rate >= rl.burst {
			delete(rl.buckets, k)
		}
	}
}

func (rl *rateLimiter) startSweeper(every time.Duration) {
	go func() {
		for range time.Tick(every) {
			rl.sweep()
		}
	}()
}

// limit envuelve un handler con el rate limiter, usando keyFn para
// identificar al cliente.
func (rl *rateLimiter) limit(keyFn func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(keyFn(r)) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"demasiados pedidos, probá en un rato"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP devuelve la IP del cliente para rate limiting. Render pone la IP
// real al final de X-Forwarded-For (lo que viene antes lo puede inventar el
// propio cliente), así que se usa la última entrada; sin el header (local)
// cae a RemoteAddr.
//
// Para POST /subscribers la IP real del navegador la manda la route de
// Next.js en X-Client-IP, pero solo se le cree si el request trae el
// secreto interno — ver trustedClientIP.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
