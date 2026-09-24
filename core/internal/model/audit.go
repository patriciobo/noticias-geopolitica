package model

import "time"

// Audit es el registro público de cómo se armó una edición: qué titulares
// entraron, qué pasó con cada uno y con qué configuración corrió el
// pipeline. Se publica junto al reporte (reports/FECHA.audit.json) para que
// cualquiera pueda verificar qué quedó afuera y por qué — no solo lo que
// terminó en el informe.
type Audit struct {
	Provenance
	Sources   []SourceStatus `json:"sources"` // estado de cada medio con feed en esta corrida
	Headlines []AuditEntry   `json:"headlines"`
}

// Provenance describe la corrida que generó el reporte.
type Provenance struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	Commit           string            `json:"commit,omitempty"`  // commit del código que corrió (GITHUB_SHA)
	RunURL           string            `json:"run_url,omitempty"` // log público de la corrida en GitHub Actions
	Provider         string            `json:"provider"`
	ClassifyModels   []string          `json:"classify_models"`   // en orden: principal y fallbacks
	SynthesizeModels []string          `json:"synthesize_models"` // en orden: principal y fallbacks
	PromptSHA256     map[string]string `json:"prompt_sha256"`     // hash de cada prompt de sistema, verificable contra el código del commit
	Counts           AuditCounts       `json:"counts"`
	// SourceProblems son los medios que no aportaron titulares hoy (feed
	// caído o sin notas recientes) — va en Provenance para que la web lo
	// muestre sin tener que leer el registro completo.
	SourceProblems []SourceStatus `json:"source_problems,omitempty"`
	// Modelos que efectivamente se usaron: cuántos titulares clasificó cada
	// uno y cuál redactó el informe. Con una cadena de respaldo pueden no
	// ser los primeros de ClassifyModels/SynthesizeModels.
	// Version cuenta cuántas veces se generó la edición de esa fecha (1 =
	// original). Revisions guarda cada versión con su motivo, para que una
	// regeneración nunca reemplace una edición en silencio.
	Version   int        `json:"version,omitempty"`
	Revisions []Revision `json:"revisions,omitempty"`

	ClassifyModelsUsed  map[string]int `json:"classify_models_used,omitempty"`
	SynthesizeModelUsed string         `json:"synthesize_model_used,omitempty"`
}

type AuditCounts struct {
	SourcesConfigured  int `json:"sources_configured"`  // medios con feed consultados
	SourcesResponded   int `json:"sources_responded"`   // medios que aportaron al menos un titular vigente
	Fetched            int `json:"fetched"`             // titulares descargados de los feeds
	PrefilterRejected  int `json:"prefilter_rejected"`  // descartados por el prefiltro de palabras clave
	ClassifierRejected int `json:"classifier_rejected"` // descartados por el clasificador (no internacionales)
	ClassifierErrors   int `json:"classifier_errors"`   // el clasificador falló y el titular quedó afuera
	Accepted           int `json:"accepted"`            // entraron a la síntesis
	LinksRemoved       int `json:"links_removed"`       // enlaces que escribió el LLM y se sacaron por no ser de una nota procesada
}

// Etapas posibles de un AuditEntry.
const (
	StagePrefilterRejected  = "prefilter_rejected"
	StageClassifierRejected = "classifier_rejected"
	StageClassifierError    = "classifier_error"
	StageAccepted           = "accepted"
)

// AuditEntry es el destino de un titular descargado.
type AuditEntry struct {
	Source       string  `json:"source"`
	Country      string  `json:"country"`
	Title        string  `json:"title"`
	Link         string  `json:"link,omitempty"`
	Stage        string  `json:"stage"`
	Reason       string  `json:"reason,omitempty"`        // explicación del clasificador, o el error
	RelationType string  `json:"relation_type,omitempty"` // solo si pasó por el clasificador
	Model        string  `json:"model,omitempty"`         // modelo que lo clasificó
	Confidence   float64 `json:"confidence,omitempty"`
}

// Estados posibles de un medio en una corrida (ver SourceStatus).
const (
	SourceOK       = "ok"
	SourceError    = "error"        // el feed no se pudo descargar o leer
	SourceNoRecent = "sin_vigentes" // respondió, pero sin titulares recientes
)

// SourceStatus es el resultado de consultar un medio con feed en una
// corrida. Se publica en el registro de auditoría para que un medio caído
// no pase inadvertido (el feed de El País estuvo congelado desde 2020 sin
// que nada lo mostrara).
type SourceStatus struct {
	Name      string `json:"name"`
	Country   string `json:"country"`
	Region    string `json:"region"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Headlines int    `json:"headlines"`
}

// Revision es una versión de una edición.
type Revision struct {
	Version     int       `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Reason      string    `json:"reason,omitempty"`
}
