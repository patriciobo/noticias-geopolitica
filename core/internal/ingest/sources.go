package ingest

import (
	"os"

	"gopkg.in/yaml.v3"

	"noticias/core/internal/model"
)

// LoadSources reads config/sources.yaml into a slice of Source.
func LoadSources(path string) ([]model.Source, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f model.SourcesFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return f.Sources, nil
}
