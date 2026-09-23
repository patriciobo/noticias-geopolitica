package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBurstThenRefill(t *testing.T) {
	now := time.Unix(0, 0)
	rl := newRateLimiter(60, time.Minute, 3) // 1 token/s, ráfaga de 3
	rl.now = func() time.Time { return now }

	for i := range 3 {
		if !rl.allow("a") {
			t.Fatalf("pedido %d dentro de la ráfaga rechazado", i)
		}
	}
	if rl.allow("a") {
		t.Fatal("pedido por encima de la ráfaga aceptado")
	}
	if !rl.allow("b") {
		t.Fatal("otra clave no debería compartir bucket")
	}

	now = now.Add(time.Second)
	if !rl.allow("a") {
		t.Fatal("tras 1s debería haber 1 token nuevo")
	}
	if rl.allow("a") {
		t.Fatal("solo se repuso 1 token")
	}
}

func TestRateLimiterSweep(t *testing.T) {
	now := time.Unix(0, 0)
	rl := newRateLimiter(60, time.Minute, 2)
	rl.now = func() time.Time { return now }

	rl.allow("a")
	rl.sweep()
	if _, ok := rl.buckets["a"]; !ok {
		t.Fatal("bucket con tokens consumidos no debería barrerse todavía")
	}
	now = now.Add(10 * time.Second)
	rl.sweep()
	if _, ok := rl.buckets["a"]; ok {
		t.Fatal("bucket lleno debería barrerse")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	if got := clientIP(r); got != "10.0.0.1" {
		t.Errorf("sin XFF: got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("con XFF debería usar la última entrada (la que agrega el proxy): got %q", got)
	}
}
