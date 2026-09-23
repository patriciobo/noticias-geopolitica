// Package config trae helpers mínimos de configuración por variables de
// entorno, compartidos entre los distintos binarios de cmd/ (api, ingest,
// newsletter) para no duplicarlos.
package config

import (
	"log"
	"os"
	"strconv"
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

// EnvInt devuelve la variable de entorno key como entero positivo, o def si
// no está seteada o no es un entero positivo válido (en ese caso lo loguea,
// para que un typo en la config no pase desapercibido).
func EnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("%s=%q no es un entero positivo, uso %d", key, v, def)
		return def
	}
	return n
}
