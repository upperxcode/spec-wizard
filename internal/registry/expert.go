package registry

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3" // Certifique-se de rodar: go get gopkg.in/yaml.v3
)

type TestConfig struct {
	Command    string `yaml:"command" json:"command"`
	FailPrompt string `yaml:"fail_prompt" json:"fail_prompt"`
}

type ExpertPlugin struct {
	ID           string      `yaml:"id" json:"id"`
	Endpoint     string      `yaml:"endpoint" json:"endpoint"`
	Triggers     []string    `yaml:"triggers" json:"triggers"`
	Language     string      `yaml:"language" json:"language"` // Adicionado para suporte imperativo
	StartCommand      string      `yaml:"start_command" json:"start_command"`
	DependencyEndpoint string      `yaml:"dependency_endpoint" json:"dependency_endpoint"` // Novo: para consultar versões
	TestConfig        *TestConfig `yaml:"test_config" json:"test_config"`
}

// LoadExperts carrega a lista de plugins de um ou mais arquivos de configuração e os mescla
func LoadExperts(paths ...string) ([]*ExpertPlugin, error) {
	expertsMap := make(map[string]*ExpertPlugin)

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // Pula se o arquivo não existir
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var config struct {
			Experts []*ExpertPlugin `yaml:"experts"`
		}

		if err := yaml.Unmarshal(data, &config); err == nil {
			for _, p := range config.Experts {
				if p != nil && p.ID != "" {
					expertsMap[p.ID] = p
				}
			}
		}
	}

	var allExperts []*ExpertPlugin
	for _, p := range expertsMap {
		allExperts = append(allExperts, p)
	}

	return allExperts, nil
}

// FindExpertByLanguage valida se o MCP existe no registro
func FindExpertByLanguage(language string, plugins []*ExpertPlugin) (*ExpertPlugin, error) {
	for _, p := range plugins {
		if strings.EqualFold(p.Language, language) {
			return p, nil
		}
	}
	return nil, errors.New("MCP não encontrado para a linguagem especificada")
}
