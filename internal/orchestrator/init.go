package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/config"
	"spec-wizard/internal/patterns"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/storage"
	adatools "spec-wizard/tools"
	"time"
)

func NewOrchestrator(path string, plugins []*registry.ExpertPlugin, engine *config.Engine) *Orchestrator {
	reg := adatools.NewRegistry(path)
	adatools.RegisterFSTools(reg)
	adatools.RegisterShellTools(reg)

	// Inicializar SnapshotManager (DB de Histórico)
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".spec-wizard")
	os.MkdirAll(dbDir, 0755)
	sm, err := storage.NewSnapshotManager(filepath.Join(dbDir, "history.db"))
	if err != nil {
		fmt.Printf("⚠️ [Orchestrator] Falha ao iniciar SnapshotManager (histórico desativado): %v\n", err)
	}

	return &Orchestrator{
		ProjectPath: path,
		Plugins:     plugins,
		Tools:       reg,
		Engine:      engine,
		KM:          NewKnowledgeManager(path),
		SnapshotMgr: sm,
	}
}

// CheckInitialState verifica se o projeto já foi iniciado e carrega config + roadmap
func (o *Orchestrator) CheckInitialState() (*ProjectState, error) {
	rootConfigPath := filepath.Join(o.ProjectPath, ".spec-wizard", "config.json")
	configPath := filepath.Join(o.ProjectPath, ".spec-wizard", "project_config.json")
	roadmapPath := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")

	state := &ProjectState{
		IsInitialized: false,
		ProjectName:   filepath.Base(o.ProjectPath),
		CurrentPath:   o.ProjectPath,
	}

	// Verifica se a pasta tem algum código
	files, _ := os.ReadDir(o.ProjectPath)
	for _, f := range files {
		if f.Name() != ".spec-wizard" && f.Name() != ".git" {
			state.HasCode = true
			break
		}
	}

	// 1. Tenta carregar da nova fonte da verdade
	if _, err := os.Stat(rootConfigPath); err == nil {
		data, _ := os.ReadFile(rootConfigPath)
		var config ProjectConfig
		if err := json.Unmarshal(data, &config); err == nil {
			state.Config = config
			state.Language = config.Language
			state.Roadmap = config.Roadmap
			state.IsInitialized = true

			if config.RoadmapLastUpdated == "" {
				roadmapMDPath := filepath.Join(o.ProjectPath, ".spec-wizard", "roadmap.md")
				if info, err := os.Stat(roadmapMDPath); err == nil {
					config.RoadmapLastUpdated = info.ModTime().Format("2006-01-02 15:04:05")
					state.Config = config
				}
			}
		}
	}

	// 2. Fallback para .spec-wizard/project_config.json
	if !state.IsInitialized {
		if _, err := os.Stat(configPath); err == nil {
			data, _ := os.ReadFile(configPath)
			json.Unmarshal(data, &state)
			state.IsInitialized = true
		}
	}

	// 3. Tenta carregar o roadmap (sprints.json)
	if state.IsInitialized {
		if _, err := os.Stat(roadmapPath); err == nil {
			roadmapData, _ := os.ReadFile(roadmapPath)
			var roadmap interface{}
			json.Unmarshal(roadmapData, &roadmap)
			state.Roadmap = roadmap
		}
	}

	return state, nil
}

// InitializeProject cria a infraestrutura de ancoragem local
func (o *Orchestrator) InitializeProject(language string, plugin *registry.ExpertPlugin) {
	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")
	os.MkdirAll(wizardPath, 0755)

	config := map[string]interface{}{
		"is_initialized": true,
		"language":       language,
		"expert_id":      plugin.ID,
		"endpoint":       plugin.Endpoint,
		"created_at":     time.Now().Format("2006-01-02"),
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(wizardPath, "project_config.json"), configData, 0644)
}

// InitializeWithLanguage ancora o projeto com uma linguagem e múltiplos padrões definidos
func (o *Orchestrator) InitializeWithLanguage(config ProjectConfig, patternRepo *patterns.PatternRepository, plugin *registry.ExpertPlugin) error {
	o.InitializeProject(config.Language, plugin)
	if len(config.Patterns) > 0 && patternRepo != nil || config.Instructions != "" {
		return o.FinalizeAnchoring(config, patternRepo)
	}
	return nil
}

// ResumeProject retoma o projeto
func (o *Orchestrator) ResumeProject(state *ProjectState) {
	fmt.Printf("🚀 Retomando projeto (Expert: %s)\n", state.ExpertID)
}

// DeleteProjectAnchor remove toda a infraestrutura de ancoragem do Spec Wizard
func (o *Orchestrator) DeleteProjectAnchor() error {
	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")
	return os.RemoveAll(wizardPath)
}

// RenameProject altera o nome lógico do projeto no arquivo de configuração
func (o *Orchestrator) RenameProject(newName string) error {
	configPath := filepath.Join(o.ProjectPath, ".spec-wizard", "config.json")

	// Tenta ler o arquivo atual
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("projeto não inicializado ou config.json ausente: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("erro ao ler config.json: %v", err)
	}

	config["projectName"] = newName

	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, updatedData, 0644)
}
