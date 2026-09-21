package model

// Source describes one outlet tracked from config/sources.yaml.
type Source struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Country  string `yaml:"country"`
	Region   string `yaml:"region"`
	Stance   string `yaml:"stance"`
	Lang     string `yaml:"lang"`
	Homepage string `yaml:"homepage"`
	RSS      string `yaml:"rss"`
}

type SourcesFile struct {
	Sources []Source `yaml:"sources"`
}
