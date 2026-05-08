package orchestrator

import (
	"spec-wizard/config"
	"spec-wizard/internal/registry"
	adatools "spec-wizard/tools"
)

type KnowledgeSource struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // "file", "link"
	Name         string `json:"name"`
	Path         string `json:"path,omitempty"`
	URL          string `json:"url,omitempty"`
	Status       string `json:"status"` // "pending", "processed", "failed"
	LastModified string `json:"last_modified"`
}

// ProjectState representa o estado persistido do projeto
type ProjectState struct {
	IsInitialized bool        `json:"is_initialized"`
	HasCode       bool        `json:"has_code"`
	ProjectName   string      `json:"project_name"`
	Language      string      `json:"language"`
	ExpertID      string      `json:"expert_id"`
	Config        interface{} `json:"config,omitempty"`
	Roadmap       interface{} `json:"roadmap,omitempty"`
	CurrentPath   string      `json:"current_path"`
}

// Orchestrator é o cérebro agnóstico que coordena os Experts
type WizardMode int

const (
	ModeNormal WizardMode = iota
	ModePlanning
	ModeBuilding
	ModeWizard
)

type Orchestrator struct {
	ProjectPath string
	Plugins     []*registry.ExpertPlugin
	Tools       *adatools.Registry
	Engine      *config.Engine
	KM          *KnowledgeManager
	Mode        WizardMode
	onProgress  func(string)
}

type LLMConfig struct {
	Provider  string `json:"provider"` // "openai", "anthropic", "gemini", "ollama", "lmstudio"
	Model     string `json:"model"`
	APIKey    string `json:"apiKey,omitempty"`
	EnvAPIKey string `json:"envApiKey,omitempty"`
	BaseURL   string `json:"baseUrl,omitempty"`
}

// ProjectConfig define as configurações técnicas e de negócio do projeto
type ProjectConfig struct {
	Language                  string            `json:"language"`
	Patterns                  []string          `json:"patterns"`
	Architecture              string            `json:"architecture"`
	Instructions              string            `json:"instructions"`
	ProjectName               string            `json:"projectName"`
	Domain                    string            `json:"domain"`
	FunctionalRequirements    string            `json:"functionalRequirements"`
	NonFunctionalRequirements string            `json:"nonFunctionalRequirements"`
	DataStrategy              string            `json:"dataStrategy"`
	StateManagement           string            `json:"stateManagement"`
	ApiContract               string            `json:"apiContract"`
	Customization             string            `json:"customization"`
	RoadmapLastUpdated        string            `json:"roadmapLastUpdated,omitempty"`
	Roadmap                   interface{}       `json:"roadmap,omitempty"`
	SourceFolders             []string          `json:"sourceFolders,omitempty"`
	LLM                       LLMConfig         `json:"llm,omitempty"`
	UserLanguage              string            `json:"userLanguage,omitempty"`
	KnowledgeBase             []KnowledgeSource `json:"knowledgeBase,omitempty"`
}

// Task representa uma unidade de trabalho dentro de uma sprint
type Task struct {
	ID                 int      `json:"id"`
	Title              string   `json:"title"`
	Priority           string   `json:"priority"` // "HIGH", "MEDIUM", "LOW"
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Status             string   `json:"status"` // "pending", "in_progress", "completed", "failed"
}

// Sprint agrupa tarefas com um objetivo comum
type Sprint struct {
	ID    int    `json:"id"`
	Goal  string `json:"goal"`
	Tasks []Task `json:"tasks"`
}

// ProjectRoadmap é o plano mestre de execução
type ProjectRoadmap struct {
	Language string   `json:"language"`
	Pattern  string   `json:"pattern"`
	Sprints  []Sprint `json:"sprints"`
}
