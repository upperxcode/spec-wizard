package registry

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3" // Certifique-se de rodar: go get gopkg.in/yaml.v3
)

type ExpertPlugin struct {
	ID           string   `yaml:"id" json:"id"`
	Endpoint     string   `yaml:"endpoint" json:"endpoint"`
	Triggers     []string `yaml:"triggers" json:"triggers"`
	Language     string   `yaml:"language" json:"language"` // Adicionado para suporte imperativo
	StartCommand string   `yaml:"start_command" json:"start_command"`
}

// LoadExperts carrega a lista de plugins do arquivo de configuração
func LoadExperts(configPath string) ([]*ExpertPlugin, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config struct {
		Experts []*ExpertPlugin `yaml:"experts"`
	}

	err = yaml.Unmarshal(data, &config)
	return config.Experts, err
}

// FindExpertByLanguage valida se o MCP existe no registro
func FindExpertByLanguage(language string, plugins []*ExpertPlugin) (*ExpertPlugin, error) {
	for _, p := range plugins {
		if strings.ToLower(p.Language) == language {
			return p, nil
		}
	}
	return nil, errors.New("MCP não encontrado para a linguagem especificada")
}
