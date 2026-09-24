package main

import "testing"

func TestAPIBaseURL(t *testing.T) {
	t.Setenv("API_BASE_URL", "")
	t.Setenv("RENDER_EXTERNAL_URL", "")
	if got := apiBaseURL(":8080"); got != "http://localhost:8080" {
		t.Errorf("sin envs: %q", got)
	}
	t.Setenv("RENDER_EXTERNAL_URL", "https://x.onrender.com/")
	if got := apiBaseURL(":10000"); got != "https://x.onrender.com" {
		t.Errorf("con RENDER_EXTERNAL_URL: %q", got)
	}
	t.Setenv("API_BASE_URL", "https://api.example.com")
	if got := apiBaseURL(":10000"); got != "https://api.example.com" {
		t.Errorf("API_BASE_URL gana: %q", got)
	}
}
