package registry

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Architecture representa um padrão de arquitetura de software disponível no catálogo
type Architecture struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	BestFor     []string `yaml:"best_for" json:"best_for"`
	Aliases     []string `yaml:"aliases" json:"aliases"`
}

// LoadArchitectures carrega o catálogo de arquiteturas de um arquivo YAML
func LoadArchitectures(path string) ([]*Architecture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config struct {
		Architectures []*Architecture `yaml:"architectures"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config.Architectures, nil
}
