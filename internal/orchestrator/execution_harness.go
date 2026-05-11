package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/prompt"
	"spec-wizard/internal/registry"
	"strings"
)

// GenerateHarnessPrompt compõe o prompt de execução seguindo a estratégia de Janela Limpa
func (o *Orchestrator) GenerateHarnessPrompt(task *Task, sprint *Sprint, config ProjectConfig, userQuestion string) (string, error) {
	if task == nil {
		return "", fmt.Errorf("tarefa não fornecida para geração de prompt")
	}

	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")

	// 1. Carregar documentos Markdown
	skills, _ := os.ReadFile(filepath.Join(wizardPath, "skills.md"))
	spec, _ := os.ReadFile(filepath.Join(wizardPath, "SPEC.md"))
	roadmapMD, _ := os.ReadFile(filepath.Join(wizardPath, "roadmap.md"))
	prd, _ := os.ReadFile(filepath.Join(wizardPath, "PRD.md"))

	// 2. Tenta carregar padrões de exclusão e stack do config.json
	var excludePatterns []string
	var stack *StackPlugin
	configPath := filepath.Join(wizardPath, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg struct {
			ExcludePatterns []string     `json:"excludePatterns"`
			Stack           *StackPlugin `json:"stack"`
		}
		json.Unmarshal(data, &cfg)
		excludePatterns = cfg.ExcludePatterns
		stack = cfg.Stack
	}

	// 3. Gerar Metadados de Execução
	fileTree := o.generateFileTree(excludePatterns)
	apiContracts := o.extractApiContracts(string(spec))

	// 4. Buscar Contexto da Base de Conhecimento
	knowledgeContext := ""
	if o.KM != nil {
		knowledgeContext = o.KM.GetRelevantContext(task.Title, task.Description)
	}

	// Converter Stack para prompt.LibraryInfo
	libs := make([]prompt.LibraryInfo, 0)
	if stack != nil {
		for _, lib := range stack.Libraries {
			libs = append(libs, prompt.LibraryInfo{
				Name:      lib.Name,
				Category:  lib.Category,
				Mandatory: lib.Mandatory,
				Disabled:  lib.Disabled,
			})
		}
	}

	// 5. Identificar nome do projeto (Agnóstico)
	finalProjectName := config.ProjectName
	if finalProjectName == "" {
		finalProjectName = filepath.Base(o.ProjectPath)
	}

	// 6. Resolver comando de teste (Agnóstico via Expert ou Fallback)
	testCommand := "go test ./..." // Fallback
	if strings.ToLower(config.Language) == "dart" || strings.ToLower(config.Language) == "flutter" {
		testCommand = "flutter test"
	}
	
	plugin, _ := registry.FindExpertByLanguage(config.Language, o.Plugins)
	if plugin != nil && plugin.TestConfig != nil && plugin.TestConfig.Command != "" {
		testCommand = plugin.TestConfig.Command
	}

	ctx := prompt.ExecutionContext{
		Language:           config.Language,
		ProjectName:        finalProjectName,
		Skills:             string(skills),
		Spec:               string(spec),
		Roadmap:            string(roadmapMD),
		PrdExcerpt:         string(prd),
		SprintID:           string(sprint.ID),
		SprintGoal:         sprint.Goal,
		TaskTitle:          task.Title,
		TaskDescription:    task.Description,
		AcceptanceCriteria: task.GetCriteriaStrings(),
		Architecture:       config.Architecture,
		FileTree:           fileTree,
		Dependencies:       fmt.Sprintf("%v", config.Patterns), // Fallback para padrões se stack for nula
		ApiContracts:       apiContracts,
		UserQuestion:       userQuestion,
		KnowledgeBase:      string(knowledgeContext),
		ReferenceFiles:     task.Files,
		RelatedFiles:       task.Files,
		TestFiles:          task.TestFiles,
		Libraries:          libs,
		TestLogs:           task.TestLogs,
		TestFailPrompt:     "",
		TestCommand:        testCommand,
	}

	// 7. Carrega Spec específica da tarefa
	taskSpecFile := fmt.Sprintf("tasks/task-%s.md", task.ID)
	taskSpec, _ := os.ReadFile(filepath.Join(wizardPath, taskSpecFile))
	if len(taskSpec) > 0 {
		ctx.TaskSpec = string(taskSpec)
	} else {
		ctx.TaskSpec = "[Siga os padrões globais do projeto.]"
	}

	// 8. Se houver falhas anteriores, injetar protocolo de reparo
	if task.TestStatus == "failure" {
		ctx.TestFailPrompt = o.generateRepairPrompt(task, config, finalProjectName)
	}

	factory := prompt.NewPromptFactory()
	return factory.GenerateExecutionPrompt(ctx), nil
}

// generateRepairPrompt cria instruções específicas de reparo baseadas no log de erro
func (o *Orchestrator) generateRepairPrompt(task *Task, config ProjectConfig, moduleName string) string {
	repair := "⚠️ **CRITICAL REPAIR MODE ACTIVATED** ⚠️\n"
	repair += "Sua tentativa anterior falhou. PRIORIZE corrigir os erros abaixo antes de qualquer nova lógica.\n\n"

	// Erro de imports no Go
	if strings.Contains(task.TestLogs, "is not in std") || strings.Contains(task.TestLogs, "no required module") {
		repair += "🚨 **IMPORT MISMATCH DETECTED:**\n"
		repair += fmt.Sprintf("- O módulo do seu projeto é: '%s'\n", moduleName)
		repair += "- **REGRA:** Imports de pacotes locais DEVEM começar com o nome do módulo.\n"
		repair += fmt.Sprintf("- **EXEMPLO:** Se o módulo é '%s' e o arquivo está em 'internal/domain', o import é '%s/internal/domain'.\n", moduleName, moduleName)
	}

	// Erro de ferramenta edit_file
	if strings.Contains(task.TestLogs, "bloco de pesquisa não encontrado") {
		repair += "🚨 **EDIT TOOL FAILURE:** O bloco de 'search' não foi encontrado exatamente.\n"
		repair += "- Use 'read_file' para copiar o conteúdo EXATO (incluindo espaços e tabs) antes de tentar 'edit_file'.\n"
	}

	repair += "\n**INSTRUÇÕES DE REPARO:**\n"
	repair += "1. Leia o LOG DE ERRO com atenção.\n"
	repair += "2. Se houver dúvida sobre o módulo, leia o 'go.mod'.\n"
	repair += "3. Corrija os arquivos afetados usando 'edit_file' ou 'write_file'.\n"

	return repair
}

func (o *Orchestrator) extractApiContracts(spec string) string {
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
