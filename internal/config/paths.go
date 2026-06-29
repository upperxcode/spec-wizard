package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// peekLogDir tenta ler o diretório de log do arquivo de configuração sem carregar todo o motor
func peekLogDir(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var cfg struct {
		LogDir string `json:"log_dir"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.LogDir
}

// Paths define as localizações padrão do sistema seguindo convenções Linux/XDG
type Paths struct {
	ConfigDir    string
	ResourcesDir string
	BinDir       string
	UIDir        string
	LogDir       string
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

	// 4. Log Dir global (Tenta ler do config.json primeiro)
	logDir := peekLogDir(filepath.Join(appConfigDir, "config.json"))
	if logDir == "" {
		logDir = filepath.Join(resourcesDir, "logs")
	}

	return Paths{
		ConfigDir:    appConfigDir,
		ResourcesDir: resourcesDir,
		BinDir:       binDir,
		UIDir:        filepath.Join(resourcesDir, "ui"),
		LogDir:       logDir,
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
