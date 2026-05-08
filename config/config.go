package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type ModelConfig struct {
	Label   string `json:"label"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Context int    `json:"context"`
}

type ProviderConfig struct {
	Name      string        `json:"name"`
	APIURL    string        `json:"api_url"`
	APIKey    string        `json:"api_key"`
	EnvAPIKey string        `json:"env_api_key"`
	Enabled   bool          `json:"enabled"`
	Models    []ModelConfig `json:"models"`
}

type ActiveConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type ProjectEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type GlobalConfig struct {
	Active        ActiveConfig     `json:"active"`
	Providers     []ProviderConfig `json:"providers"`
	Theme         string           `json:"theme"`
	Projects      []ProjectEntry   `json:"projects"`
	ActiveProject string           `json:"active_project"`
}

type Engine struct {
	mu   sync.RWMutex
	cfg  GlobalConfig
	path string
}



func NewEngine(configPath string) (*Engine, error) {
	e := &Engine{
		path: configPath,
		cfg:  GlobalConfig{Providers: []ProviderConfig{}},
	}
	
	if err := e.LoadConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Se não existir, salva o default
		if err := e.SaveConfig(); err != nil {
			return nil, err
		}
	}
	if err := e.SetupThemesDir(); err != nil {
		fmt.Printf("⚠️ Aviso: não foi possível configurar a pasta global de temas: %v\n", err)
	}

	return e, nil
}

func (e *Engine) GetThemesDir() string {
	return filepath.Join(filepath.Dir(e.path), "themes")
}

func (e *Engine) SetupThemesDir() error {
	globalThemesDir := e.GetThemesDir()
	if err := os.MkdirAll(globalThemesDir, 0755); err != nil {
		return err
	}

	// Tenta copiar os temas da pasta local ./themes se estivermos rodando no diretório do projeto
	localThemesDir := "./themes"
	files, err := os.ReadDir(localThemesDir)
	if err != nil {
		// Se não achar a pasta local, apenas ignora (pode estar rodando de outro lugar já instalado)
		return nil
	}

	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			srcPath := filepath.Join(localThemesDir, f.Name())
			destPath := filepath.Join(globalThemesDir, f.Name())
			
			// Só copia se o arquivo não existir no destino
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				data, err := os.ReadFile(srcPath)
				if err == nil {
					os.WriteFile(destPath, data, 0644)
				}
			}
		}
	}
	return nil
}

func (e *Engine) LoadConfig() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	data, err := os.ReadFile(e.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &e.cfg)
}

func (e *Engine) ReloadConfig() error {
	return e.LoadConfig()
}


func (e *Engine) SaveConfig() error {
	dir := filepath.Dir(e.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(e.cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.path, data, 0644)
}

// GetActiveModel retorna o modelo configurado como ativo ou o primeiro habilitado
func (e *Engine) GetActiveModel() (ProviderConfig, ModelConfig, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	activeP := e.cfg.Active.Provider
	activeM := e.cfg.Active.Model

	// 1. Tenta buscar exatamente o que está na chave 'active'
	if activeP != "" && activeM != "" {
		for _, p := range e.cfg.Providers {
			if p.Name == activeP {
				for _, m := range p.Models {
					if m.Name == activeM {
						return p, m, nil
					}
				}
			}
		}
	}

	// 2. Fallback: Primeiro provedor habilitado e seu primeiro modelo habilitado
	for _, p := range e.cfg.Providers {
		if p.Enabled {
			for _, m := range p.Models {
				if m.Enabled {
					return p, m, nil
				}
			}
			if len(p.Models) > 0 {
				return p, p.Models[0], nil
			}
		}
	}

	return ProviderConfig{}, ModelConfig{}, fmt.Errorf("nenhum modelo ativo ou habilitado encontrado na configuração global")
}


func (e *Engine) GetProviders() []ProviderConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg.Providers
}

func (e *Engine) SetProviders(providers []ProviderConfig) error {
	e.mu.Lock()
	e.cfg.Providers = providers
	e.mu.Unlock()
	return e.SaveConfig()
}

func (e *Engine) PingProvider(name string) error {
	e.mu.RLock()
	var provider ProviderConfig
	found := false
	for _, p := range e.cfg.Providers {
		if p.Name == name {
			provider = p
			found = true
			break
		}
	}
	e.mu.RUnlock()

	if !found {
		return fmt.Errorf("provedor %s não encontrado", name)
	}

	if provider.APIURL == "" {
		return fmt.Errorf("URL base não configurada para %s", name)
	}
	
	// Tenta um ping básico (cada provedor pode ter um endpoint diferente, mas /models é comum)
	url := provider.APIURL
	if provider.Name == "ollama" {
		url = provider.APIURL + "/api/tags"
	} else if provider.Name == "openai" || provider.Name == "openrouter" {
		url = provider.APIURL + "/models"
	} else if provider.Name == "gemini" {
		// O ping do Gemini funciona melhor com o endpoint de listagem de modelos
		// e precisa da chave via query param ou header
		url = "https://generativelanguage.googleapis.com/v1beta/models"
		if provider.APIKey != "" {
			url = fmt.Sprintf("%s?key=%s", url, provider.APIKey)
		}
	}

	// Resolve API Key (Config or Env Var)
	apiKey := provider.APIKey
	if apiKey == "" {
		// 1. Try specifically configured env var name
		if provider.EnvAPIKey != "" {
			apiKey = os.Getenv(provider.EnvAPIKey)
		}
		
		// 2. Fallback to hardcoded defaults if still empty
		if apiKey == "" {
			switch provider.Name {
			case "openai":
				apiKey = os.Getenv("OPENAI_API_KEY")
			case "anthropic":
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			case "gemini":
				apiKey = os.Getenv("GEMINI_API_KEY")
			case "openrouter":
				apiKey = os.Getenv("OPENROUTER_API_KEY")
			}
		}
	}

	req, _ := http.NewRequest("GET", url, nil)
	if (provider.Name == "openai" || provider.Name == "openrouter") && apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}
	return nil
}

func (e *Engine) GetConfig() GlobalConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg
}

func (e *Engine) SetActiveModel(provider, model string) error {
	e.mu.Lock()
	e.cfg.Active.Provider = provider
	e.cfg.Active.Model = model
	e.mu.Unlock()
	return e.SaveConfig()
}

func (e *Engine) SetTheme(name string) error {
	e.mu.Lock()
	e.cfg.Theme = name
	e.mu.Unlock()
	return e.SaveConfig()
}

func (e *Engine) SetProviderStatus(name string, enabled bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	for i, p := range e.cfg.Providers {
		if p.Name == name {
			e.cfg.Providers[i].Enabled = enabled
			return e.SaveConfig()
		}
	}
	return fmt.Errorf("provedor %s não encontrado", name)
}

func (e *Engine) SetActiveProject(path string) error {

	e.mu.Lock()
	e.cfg.ActiveProject = path
	e.mu.Unlock()
	return e.SaveConfig()
}

func (e *Engine) GetProjects() []ProjectEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg.Projects
}

func (e *Engine) AddProject(name, path string) error {
	e.mu.Lock()
	// Evitar duplicatas
	for i, p := range e.cfg.Projects {
		if p.Path == path {
			e.cfg.Projects[i].Name = name
			e.mu.Unlock()
			return e.SaveConfig()
		}
	}
	e.cfg.Projects = append(e.cfg.Projects, ProjectEntry{Name: name, Path: path})
	e.mu.Unlock()
	return e.SaveConfig()
}

func (e *Engine) RemoveProject(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	newProjects := []ProjectEntry{}
	for _, p := range e.cfg.Projects {
		if p.Path != path {
			newProjects = append(newProjects, p)
		}
	}
	e.cfg.Projects = newProjects
	return e.SaveConfig()
}
