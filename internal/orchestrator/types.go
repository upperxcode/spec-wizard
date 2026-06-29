package orchestrator

import (
	"encoding/json"
	"fmt"
	"spec-wizard/config"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/storage"
	adatools "spec-wizard/tools"
)

// FlexibleID permite decodificar IDs que venham como int ou string no JSON
type FlexibleID string

func (f *FlexibleID) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, (*string)(f))
	}
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*f = FlexibleID(fmt.Sprint(i))
		return nil
	}
	// Fallback para string se não for int nem string com aspas (raro)
	return json.Unmarshal(b, (*string)(f))
}

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
	SnapshotMgr *storage.SnapshotManager // Novo: Gerenciador de Snapshots
	Mode        WizardMode
	onProgress  func(string)
}

type LLMConfig struct {
	Provider   string `json:"provider"` // "openai", "anthropic", "gemini", "ollama", "lmstudio"
	Model      string `json:"model"`
	APIKey     string `json:"apiKey,omitempty"`
	EnvAPIKey  string `json:"envApiKey,omitempty"`
	BaseURL    string `json:"baseUrl,omitempty"`
	Sequential bool   `json:"sequential,omitempty"`
	UseCLI     bool   `json:"useCLI,omitempty"`
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
	UserLanguage              string            `json:"userLanguage,omitempty"`
	KnowledgeBase             []KnowledgeSource `json:"knowledgeBase,omitempty"`
	ExcludePatterns           []string          `json:"excludePatterns,omitempty"`
	Stack                     *StackPlugin      `json:"stack,omitempty"`
	CurrentSprint             *Sprint           `json:"currentSprint,omitempty"` // Fallback para execução sem arquivo
	CurrentTask               *Task             `json:"currentTask,omitempty"`   // Fallback para execução sem arquivo
}

// AcceptanceCriterion representa um critério individual e seu estado de validação
type AcceptanceCriterion struct {
	Description string `json:"description"`
	Status      string `json:"status"` // "pending", "met", "unmet"
	Feedback    string `json:"feedback,omitempty"`
}

type Task struct {
	ID                 FlexibleID            `json:"id"`
	Title              string                `json:"title"`
	Priority           string                `json:"priority"` // "HIGH", "MEDIUM", "LOW"
	Description        string                `json:"description"`
	AcceptanceCriteria []AcceptanceCriterion `json:"acceptance_criteria"`
	Status             string                `json:"status"` // "pending", "in_progress", "completed", "failed"
	TestLogs           string                `json:"test_logs,omitempty"`
	TestStatus         string                `json:"test_status,omitempty"` // "success", "failure"
	TestFiles          interface{}           `json:"test_files,omitempty"`
	Files              interface{}           `json:"files,omitempty"`
}

// UnmarshalJSON para Task para suportar migração de acceptance_criteria (string[] -> object[])
func (t *Task) UnmarshalJSON(data []byte) error {
	type Alias Task
	aux := &struct {
		AcceptanceCriteria interface{} `json:"acceptance_criteria"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.AcceptanceCriteria != nil {
		switch v := aux.AcceptanceCriteria.(type) {
		case []interface{}:
			var criteria []AcceptanceCriterion
			for _, item := range v {
				switch c := item.(type) {
				case string:
					criteria = append(criteria, AcceptanceCriterion{Description: c, Status: "pending"})
				case map[string]interface{}:
					desc, _ := c["description"].(string)
					status, _ := c["status"].(string)
					if status == "" {
						status = "pending"
					}
					criteria = append(criteria, AcceptanceCriterion{Description: desc, Status: status})
				}
			}
			t.AcceptanceCriteria = criteria
		}
	}
	return nil
}

// GetCriteriaStrings retorna os critérios de aceite apenas como strings
func (t *Task) GetCriteriaStrings() []string {
	var result []string
	for _, c := range t.AcceptanceCriteria {
		result = append(result, c.Description)
	}
	return result
}

// Sprint agrupa tarefas com um objetivo comum
type Sprint struct {
	ID    FlexibleID `json:"id"`
	Goal  string     `json:"goal"`
	Tasks []Task     `json:"tasks"`
}

// ProjectRoadmap é o plano mestre de execução
type ProjectRoadmap struct {
	ProjectName string   `json:"project_name"`
	Language    string   `json:"language"`
	Pattern     string   `json:"pattern"`
	Sprints     []Sprint `json:"sprints"`
}
