package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
)

// Library representa uma dependência técnica específica
type Library struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Mandatory    bool   `json:"mandatory"`
	Disabled     bool   `json:"disabled,omitempty"`
	Link         string `json:"link"`
	Description  string `json:"description"`
	InstallCmd   string `json:"install_cmd"`
	UsageExample string `json:"usage_example"`
}

// StackPlugin define o conjunto de tecnologias opinativas para o projeto
type StackPlugin struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Language  string    `json:"language"`
	Rules     []string  `json:"rules"`
	Libraries []Library `json:"libraries"`
}

// LoadStackPlugin carrega as configurações de stack de um arquivo JSON
func LoadStackPlugin(path string) (*StackPlugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo de stack: %w", err)
	}

	var plugin StackPlugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON de stack: %w", err)
	}

	return &plugin, nil
}
