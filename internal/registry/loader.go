package registry

import (
	"os"
	"gopkg.in/yaml.v3" // Você precisará instalar: go get gopkg.in/yaml.v3
)

type Config struct {
	Experts []ExpertPlugin `yaml:"experts"`
}

// LoadRegistry lê o arquivo de configuração e retorna a lista de plugins
func LoadRegistry(configPath string) ([]ExpertPlugin, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return config.Experts, err
}
