package config

import (
	"os"
	"path/filepath"
)

// Paths define as localizações padrão do sistema seguindo convenções Linux/XDG
type Paths struct {
	ConfigDir    string
	ResourcesDir string
	BinDir       string
	UIDir        string
}

// GetPaths resolve os caminhos dinamicamente
func GetPaths() Paths {
	home, _ := os.UserHomeDir()

	// 1. Config Dir (XDG_CONFIG_HOME)
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(home, ".config")
	}
	appConfigDir := filepath.Join(configDir, "spec-wizard")

	// 2. Resources Dir (Baseado no desejo do usuário: ~/.spec-wizard)
	// Aqui ficarão experts.yaml, plugins/, themes/, logs/
	resourcesDir := filepath.Join(home, ".spec-wizard")

	// 3. Bin Dir (~/.local/bin)
	binDir := filepath.Join(home, ".local", "bin")

	return Paths{
		ConfigDir:    appConfigDir,
		ResourcesDir: resourcesDir,
		BinDir:       binDir,
		UIDir:        filepath.Join(resourcesDir, "ui"),
	}
}

// EnsureDirs garante que as pastas base existam
func (p Paths) EnsureDirs() error {
	dirs := []string{p.ConfigDir, p.ResourcesDir, filepath.Join(p.ResourcesDir, "plugins"), filepath.Join(p.ResourcesDir, "logs")}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// ResolveResource retorna o caminho absoluto de um recurso
func (p Paths) ResolveResource(name string) string {
	return filepath.Join(p.ResourcesDir, name)
}

// ResolveConfig retorna o caminho absoluto de um arquivo de configuração
func (p Paths) ResolveConfig(name string) string {
	return filepath.Join(p.ConfigDir, name)
}
