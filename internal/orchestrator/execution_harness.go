package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/prompt"
	"strings"
)

// GenerateHarnessPrompt compõe o prompt de execução seguindo a estratégia de Janela Limpa
func (o *Orchestrator) GenerateHarnessPrompt(userQuestion string) (string, error) {
	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")

	// 1. Carregar documentos Markdown
	skills, _ := os.ReadFile(filepath.Join(wizardPath, "skills.md"))
	spec, _ := os.ReadFile(filepath.Join(wizardPath, "SPEC.md"))
	summary, _ := os.ReadFile(filepath.Join(wizardPath, "SUMMARY.md"))
	roadmapMD, _ := os.ReadFile(filepath.Join(wizardPath, "roadmap.md"))
	prd, _ := os.ReadFile(filepath.Join(wizardPath, "PRD.md"))

	// 2. Localizar tarefa atual no config.json ou sprints.json
	configPath := filepath.Join(wizardPath, "config.json")
	sprintsPath := filepath.Join(wizardPath, "sprints.json")

	var roadmap ProjectRoadmap
	foundRoadmap := false

	var excludePatterns []string
	// Tenta config.json primeiro
	if data, err := os.ReadFile(configPath); err == nil {
		var config struct {
			Roadmap         *ProjectRoadmap `json:"roadmap"`
			ExcludePatterns []string        `json:"excludePatterns"`
		}
		if err := json.Unmarshal(data, &config); err == nil {
			if config.Roadmap != nil {
				roadmap = *config.Roadmap
				foundRoadmap = true
			}
			excludePatterns = config.ExcludePatterns
		}
	}

	// Fallback para sprints.json
	if !foundRoadmap {
		if data, err := os.ReadFile(sprintsPath); err == nil {
			if err := json.Unmarshal(data, &roadmap); err == nil {
				foundRoadmap = true
			}
		}
	}

	if !foundRoadmap {
		return "", fmt.Errorf("roadmap não encontrado. Gere o roadmap no dashboard primeiro")
	}

	var currentTask *Task
	var currentSprint *Sprint
	for _, s := range roadmap.Sprints {
		for _, t := range s.Tasks {
			if t.Status == "pending" || t.Status == "in_progress" {
				currentTask = &t
				currentSprint = &s
				break
			}
		}
		if currentTask != nil {
			break
		}
	}

	if currentTask == nil {
		return "", fmt.Errorf("nenhuma tarefa pendente encontrada no roadmap")
	}

	// 3. Gerar Metadados de Execução
	fileTree := o.generateFileTree(excludePatterns)
	dependencies := o.getDependencies()
	apiContracts := o.extractApiContracts(string(spec))

	// 4. Buscar Contexto da Base de Conhecimento
	knowledgeContext := ""
	if o.KM != nil {
		knowledgeContext = o.KM.GetRelevantContext(currentTask.Title, currentTask.Description)
	}

	// 5. Montar Contexto
	ctx := prompt.ExecutionContext{
		Skills:             string(skills),
		Spec:               string(spec),
		Summary:            string(summary),
		Roadmap:            string(roadmapMD),
		PrdExcerpt:         string(prd),
		SprintID:           string(currentSprint.ID),
		SprintGoal:         currentSprint.Goal,
		TaskTitle:          currentTask.Title,
		TaskDescription:    currentTask.Description,
		AcceptanceCriteria: currentTask.AcceptanceCriteria,
		Architecture:       roadmap.Pattern,
		FileTree:           fileTree,
		Dependencies:       dependencies,
		ApiContracts:       apiContracts,
		UserQuestion:       userQuestion,
		KnowledgeBase:      knowledgeContext,
	}

	factory := prompt.NewPromptFactory()
	return factory.GenerateExecutionPrompt(ctx), nil
}

func (o *Orchestrator) getDependencies() string {
	// Implementação inicial para Flutter/Dart
	pubspecPath := filepath.Join(o.ProjectPath, "pubspec.yaml")
	data, err := os.ReadFile(pubspecPath)
	if err != nil {
		return "Dependências não identificadas (pubspec.yaml ausente)"
	}

	// Extração simples das dependências (apenas para o prompt)
	lines := strings.Split(string(data), "\n")
	var deps []string
	inDeps := false
	for _, line := range lines {
		if strings.HasPrefix(line, "dependencies:") {
			inDeps = true
			continue
		}
		if inDeps && (strings.HasPrefix(line, "dev_dependencies:") || !strings.HasPrefix(line, "  ")) {
			break
		}
		if inDeps && strings.TrimSpace(line) != "" {
			deps = append(deps, strings.TrimSpace(line))
		}
	}
	return strings.Join(deps, ", ")
}

func (o *Orchestrator) extractApiContracts(spec string) string {
	// Procura pela seção de Contrato de API no SPEC.md
	lines := strings.Split(spec, "\n")
	var contracts []string
	found := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "api contract") || strings.Contains(strings.ToLower(line), "contrato de api") {
			found = true
			continue
		}
		if found && strings.HasPrefix(line, "##") {
			break
		}
		if found && strings.TrimSpace(line) != "" {
			contracts = append(contracts, line)
		}
	}

	if len(contracts) == 0 {
		return "Nenhum contrato de API detalhado na SPEC.md"
	}
	return strings.Join(contracts, "\n")
}
