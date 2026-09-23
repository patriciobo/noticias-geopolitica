// Package config trae helpers mínimos de configuración por variables de
// entorno, compartidos entre los distintos binarios de cmd/ (api, ingest,
// newsletter) para no duplicarlos.
package config

import (
	"os"
	"strings"
)

// LoadDotEnv lee pares KEY=VALUE de un archivo .env opcional y los aplica
// vía os.Setenv, sin pisar variables ya seteadas en el entorno real. Que
// falte el archivo no es un error — .env es solo una comodidad para
// desarrollo local; en prod (Render, GitHub Actions) las variables reales
// ya vienen seteadas.
func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// EnvOrDefault devuelve la variable de entorno key, o def si no está seteada.
func EnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
