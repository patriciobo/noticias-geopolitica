package filter

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Gazetteer struct {
	Countries []string `yaml:"countries"`
	Companies []string `yaml:"companies"`
	Keywords  []string `yaml:"keywords"`
}

func LoadGazetteer(path string) (*Gazetteer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g Gazetteer
	if err := yaml.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	return &g, nil
}
