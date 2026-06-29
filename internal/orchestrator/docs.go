package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/patterns"
	"spec-wizard/internal/prompt"
	"strings"
	"time"
)

// GeneratePRD generates the content of PRD.md (Requirements and Domain)
func GeneratePRD(config ProjectConfig) string {
	var sb strings.Builder
	sb.WriteString("# 📄 PRODUCT REQUIREMENT DOCUMENT (PRD)\n\n")
	sb.WriteString("## 🎯 Identity & Purpose\n")
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", config.ProjectName))
	sb.WriteString(fmt.Sprintf("- **Domain**: %s\n\n", config.Domain))

	sb.WriteString("## 🚀 Business Requirements\n")
	sb.WriteString("### ✅ Functional\n")
	sb.WriteString(config.FunctionalRequirements + "\n\n")
	sb.WriteString("### ⚙️ Non-Functional\n")
	sb.WriteString(config.NonFunctionalRequirements + "\n\n")

	return sb.String()
}

// GenerateSPEC generates the content of SPEC.md (Architecture and Stack)
func GenerateSPEC(config ProjectConfig, allRules map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("# 🏗️ SYSTEM SPECIFICATION (SPEC)\n\n")
	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", config.Language))
	sb.WriteString(fmt.Sprintf("- **Core Architecture**: %s\n", config.Architecture))
	sb.WriteString(fmt.Sprintf("- **Data Strategy**: %s\n", config.DataStrategy))
	sb.WriteString(fmt.Sprintf("- **State Management**: %s\n", config.StateManagement))
	sb.WriteString(fmt.Sprintf("- **API Contract**: %s\n\n", config.ApiContract))

	if config.Stack != nil {
		sb.WriteString("## 📦 Opinionated Stack\n")
		sb.WriteString(fmt.Sprintf("- **Template**: %s\n", config.Stack.Name))
		for _, lib := range config.Stack.Libraries {
			mandatory := ""
			if lib.Mandatory {
				mandatory = " (Obrigatório)"
			}
			sb.WriteString(fmt.Sprintf("- %s: %s%s\n", lib.Category, lib.Name, mandatory))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 📂 Patterns & Standards\n")
	for _, p := range config.Patterns {
		sb.WriteString(fmt.Sprintf("- %s\n", p))
	}

	if len(allRules) > 0 {
		sb.WriteString("\n## 🛠️ Architecture Constraints\n")
		for pattern, rules := range allRules {
			sb.WriteString(fmt.Sprintf("### %s\n", pattern))
			for _, rule := range rules {
				sb.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			sb.WriteString("\n")
		}
	}

	if config.Customization != "" {
		sb.WriteString("\n### 🛠️ Technical Customizations\n")
		sb.WriteString(config.Customization + "\n")
	}

	return sb.String()
}

// GenerateSkills generates the content of skills.md (Golden Rules)
func GenerateSkills(config ProjectConfig, allRules map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("# 📏 SKILLS & GOLDEN RULES\n\n")
	sb.WriteString("> Imperative rules that must be followed for every task.\n\n")

	if config.Instructions != "" {
		sb.WriteString("## 📝 Custom Instructions\n")
		sb.WriteString(config.Instructions + "\n\n")
	}

	if len(allRules) > 0 {
		sb.WriteString("## ⚓ Core Technical Rules\n")
		for pattern, rules := range allRules {
			sb.WriteString(fmt.Sprintf("### %s\n", pattern))
			for _, rule := range rules {
				sb.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GenerateSummary generates the executive summary SUMMARY.md
func GenerateSummary(config ProjectConfig) string {
	var sb strings.Builder
	sb.WriteString("# 📋 EXECUTIVE IMPLEMENTATION SUMMARY\n\n")
	sb.WriteString(fmt.Sprintf("## 🚀 Project: %s\n\n", config.ProjectName))

	sb.WriteString("### 🎯 Primary Objective\n")
	sb.WriteString(config.Domain + "\n\n")

	sb.WriteString("### 🏗️ Architecture & Stack\n")
	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", config.Language))
	sb.WriteString(fmt.Sprintf("- **Architecture**: %s\n", config.Architecture))
	sb.WriteString(fmt.Sprintf("- **State**: %s\n", config.StateManagement))
	sb.WriteString(fmt.Sprintf("- **Data**: %s\n\n", config.DataStrategy))

	sb.WriteString("### 🏁 Roadmap Progress\n")
	sb.WriteString("Refer to the [roadmap.md](./roadmap.md) file for sprint details.\n")

	return sb.String()
}

// GenerateWikiHome generates the Wiki Home
func GenerateWikiHome(config ProjectConfig) string {
	return fmt.Sprintf(`# 🏠 Project Wiki: %s

Welcome to the centralized project documentation. This space serves as the long-term memory for human developers and AI agents.

## 📌 Starting Point
- **[PRD.md](../PRD.md)**: Business Requirements.
- **[SPEC.md](../SPEC.md)**: Technical Specification.
- **[skills.md](../skills.md)**: Golden Rules.
- **[SUMMARY.md](../SUMMARY.md)**: Executive Summary.
- **[Execution Roadmap](../roadmap.md)**: What we are building and what has already been done.

## 🏗️ General Architecture
This system was designed under the **%s** paradigm with a focus on **%s**.
`, config.ProjectName, config.Architecture, config.Domain)
}

// GenerateRoadmapMarkdown converts the structured roadmap to Markdown
func GenerateRoadmapMarkdown(rawRoadmap interface{}) string {
	var roadmap ProjectRoadmap

	// Try to convert interface{} to ProjectRoadmap
	data, _ := json.Marshal(rawRoadmap)
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return "# 🗺️ Project Roadmap\n\nError processing the structured roadmap."
	}

	var sb strings.Builder
	title := "🗺️ STRATEGIC EXECUTION ROADMAP"
	if roadmap.ProjectName != "" {
		title = fmt.Sprintf("🗺️ ROADMAP: %s", roadmap.ProjectName)
	}
	sb.WriteString("# " + title + "\n\n")
	sb.WriteString("> This document details the development phases and the current progress of the system.\n\n")

	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", roadmap.Language))
	sb.WriteString(fmt.Sprintf("- **Pattern**: %s\n\n", roadmap.Pattern))

	for _, sprint := range roadmap.Sprints {
		sb.WriteString(fmt.Sprintf("## Sprint %s: %s\n", sprint.ID, sprint.Goal))
		for _, task := range sprint.Tasks {
			statusIcon := "⏳"
			if task.Status == "completed" {
				statusIcon = "✅"
			} else if task.Status == "in_progress" {
				statusIcon = "🚀"
			} else if task.Status == "failed" {
				statusIcon = "❌"
			}

			priorityTag := ""
			if task.Priority != "" {
				priorityTag = fmt.Sprintf(" `[%s]`", task.Priority)
			}

			sb.WriteString(fmt.Sprintf("### %s %s%s\n", statusIcon, task.Title, priorityTag))
			sb.WriteString(task.Description + "\n\n")

			if len(task.AcceptanceCriteria) > 0 {
				sb.WriteString("**Acceptance Criteria:**\n")
				for _, crit := range task.AcceptanceCriteria {
					sb.WriteString(fmt.Sprintf("- [ ] %s\n", crit))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("\n*End of Strategic Roadmap*")
	return sb.String()
}

// SaveAgentsDocs persiste toda a documentação em .spec-wizard/
func (o *Orchestrator) SaveAgentsDocs(config ProjectConfig, patternRepo *patterns.PatternRepository) error {
	// Feedback visual de segurança para o usuário
	fmt.Printf("💾 [ORQUESTRADOR] Salvando Snapshot de Segurança para: %s\n", config.ProjectName)

	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")
	os.MkdirAll(filepath.Join(wizardPath, "wiki"), 0755)
	os.MkdirAll(filepath.Join(wizardPath, "task-logs"), 0755)

	allRules := make(map[string][]string)
	if patternRepo != nil {
		allRules = patternRepo.GetMultiplePatternRules(config.Language, config.Patterns)
	}

	// Smart Merge: Se já existir um config.json, preserve campos que não vieram na nova config
	// Isso evita perda de dados por cliques acidentais no Dashboard ou falhas na UI
	existingPath := filepath.Join(wizardPath, "config.json")
	if data, err := os.ReadFile(existingPath); err == nil {
		var oldConfig ProjectConfig
		if err := json.Unmarshal(data, &oldConfig); err == nil {
			if config.Instructions == "" {
				config.Instructions = oldConfig.Instructions
			}
			if config.Domain == "" {
				config.Domain = oldConfig.Domain
			}
			if config.FunctionalRequirements == "" {
				config.FunctionalRequirements = oldConfig.FunctionalRequirements
			}
			if config.NonFunctionalRequirements == "" {
				config.NonFunctionalRequirements = oldConfig.NonFunctionalRequirements
			}
			if config.DataStrategy == "" {
				config.DataStrategy = oldConfig.DataStrategy
			}
			if config.ApiContract == "" {
				config.ApiContract = oldConfig.ApiContract
			}
			if config.Customization == "" {
				config.Customization = oldConfig.Customization
			}
			if config.Stack == nil || config.Stack.ID == "" {
				config.Stack = oldConfig.Stack
			}
			if config.Roadmap == nil {
				config.Roadmap = oldConfig.Roadmap
			}
			if config.ProjectName == "" {
				config.ProjectName = oldConfig.ProjectName
			}
		}
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(existingPath, configData, 0644)

	os.WriteFile(filepath.Join(wizardPath, "PRD.md"), []byte(GeneratePRD(config)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "SPEC.md"), []byte(GenerateSPEC(config, allRules)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "skills.md"), []byte(GenerateSkills(config, allRules)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "SUMMARY.md"), []byte(GenerateSummary(config)), 0644)

	wikiContent := GenerateWikiHome(config)
	os.WriteFile(filepath.Join(wizardPath, "wiki", "home.md"), []byte(wikiContent), 0644)

	if config.Roadmap != nil {
		roadmapMD := GenerateRoadmapMarkdown(config.Roadmap)
		os.WriteFile(filepath.Join(wizardPath, "roadmap.md"), []byte(roadmapMD), 0644)
	}

	if config.Stack != nil {
		stackData, _ := json.MarshalIndent(config.Stack, "", "  ")
		os.WriteFile(filepath.Join(wizardPath, "stack.json"), stackData, 0644)
	}

	// Persiste snapshot incremental se houver Roadmap
	if config.Roadmap != nil && o.SnapshotMgr != nil {
		o.SnapshotMgr.SaveSnapshot(config.ProjectName, "GLOBAL", "USER_SAVE", config.Roadmap)
	}

	return nil
}

// FinalizeAnchoring monta o conteúdo do skills.md com regras imperativas
func (o *Orchestrator) FinalizeAnchoring(config ProjectConfig, patternRepo *patterns.PatternRepository) error {
	allRules := patternRepo.GetMultiplePatternRules(config.Language, config.Patterns)
	var content strings.Builder
	content.WriteString("# Skills & Golden Rules - " + strings.Title(config.Language) + "\n\n")

	if config.ProjectName != "" {
		content.WriteString("## Contexto do Projeto: " + config.ProjectName + "\n")
	}

	// ... (Simplificado para brevidade, mas mantendo a lógica de regras)
	for patternName, rules := range allRules {
		content.WriteString("## " + patternName + "\n")
		for _, rule := range rules {
			content.WriteString("- " + rule + "\n")
		}
		content.WriteString("\n")
	}

	skillsFile := filepath.Join(o.ProjectPath, ".spec-wizard", "skills.md")
	return os.WriteFile(skillsFile, []byte(content.String()), 0644)
}

// readProjectDoc lê um documento da pasta .spec-wizard do projeto
func (o *Orchestrator) readProjectDoc(root, filename string) (string, error) {
	path := filepath.Join(root, ".spec-wizard", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("[Documento %s não encontrado]", filename), err
	}
	return string(data), nil
}

// isEmpty utilitário para verificar se um valor de interface é vazio
func isEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		return v == "" || v == "custom" || v == "domínio não identificado" || v == "unknown go project" || v == "yourmodule"
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	}
	return false
}

func (o *Orchestrator) BootstrapTaskSpec(config ProjectConfig, sprintID, taskID string) (string, error) {
	// Recuperação de pânico para evitar erro 500 silencioso
	defer func() {
		if r := recover(); r != nil {
			logger.Error("🔥 [BootstrapTaskSpec] CRITICAL PANIC", fmt.Errorf("%v", r))
		}
	}()

	logger.Info("🏗️  [BootstrapTaskSpec] Iniciando geração de SPEC para tarefa", "taskID", taskID, "sprintID", sprintID, "userLang", config.UserLanguage)

	// 1. Tenta carregar o roadmap (sprints.json ou memória)
	var roadmap ProjectRoadmap
	sprintsPath := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")

	foundRoadmap := false
	if data, err := os.ReadFile(sprintsPath); err == nil {
		if err := json.Unmarshal(data, &roadmap); err == nil {
			foundRoadmap = true
		} else {
			logger.Warn("⚠️ [BootstrapTaskSpec] Erro ao ler sprints.json, tentando fallback", "error", err)
		}
	}

	if !foundRoadmap && config.Roadmap != nil {
		logger.Info("ℹ️ [BootstrapTaskSpec] Usando roadmap da memória")
		roadmapData, _ := json.Marshal(config.Roadmap)
		if err := json.Unmarshal(roadmapData, &roadmap); err != nil {
			return "", fmt.Errorf("erro ao processar roadmap da configuração: %v", err)
		}
		foundRoadmap = true
	}

	if !foundRoadmap {
		return "", fmt.Errorf("roadmap estratégico não encontrado. Gere o plano estratégico primeiro")
	}

	var targetTask *Task
	var targetSprint *Sprint
	for i := range roadmap.Sprints {
		if string(roadmap.Sprints[i].ID) == sprintID {
			targetSprint = &roadmap.Sprints[i]
			for j := range roadmap.Sprints[i].Tasks {
				if string(roadmap.Sprints[i].Tasks[j].ID) == taskID {
					targetTask = &roadmap.Sprints[i].Tasks[j]
					break
				}
			}
		}
	}

	if targetTask == nil {
		return "", fmt.Errorf("tarefa %s não encontrada na sprint %s", taskID, sprintID)
	}

	// 2. Carrega SPEC Global para contexto
	globalSpec, _ := o.readProjectDoc(o.ProjectPath, "SPEC.md")

	// 3. Prepara contexto para o prompt factory
	factory := prompt.NewPromptFactory()
	bootstrapPrompt := factory.GenerateTaskSpecBootstrap(prompt.ExecutionContext{
		ProjectName:        config.ProjectName,
		Architecture:       config.Architecture,
		Language:           config.Language,
		Summary:            globalSpec,
		SprintGoal:         targetSprint.Goal,
		TaskTitle:          targetTask.Title,
		TaskDescription:    targetTask.Description,
		AcceptanceCriteria: targetTask.GetCriteriaStrings(),
		UserLanguage:       config.UserLanguage,
	})

	// LOG DO PROMPT (Resumo para o log não ficar gigante)
	promptPreview := bootstrapPrompt
	if len(promptPreview) > 500 {
		promptPreview = promptPreview[:500] + "..."
	}
	logger.Info("📡 [BootstrapTaskSpec] Prompt gerado para IA", "preview", promptPreview)

	// 4. Chama a IA
	llmClient, err := o.GetLLMClient(config)
	if err != nil {
		return "", fmt.Errorf("falha ao obter cliente LLM: %v", err)
	}

	logger.Info("⏳ [BootstrapTaskSpec] Aguardando resposta da IA...")
	startTime := time.Now()
	rawResponse, err := llmClient.Ask(bootstrapPrompt)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("❌ [BootstrapTaskSpec] Erro na chamada da IA", err, "duration", duration.String())
		return "", fmt.Errorf("IA falhou após %s: %v", duration.String(), err)
	}

	logger.Info("✅ [BootstrapTaskSpec] IA respondeu com sucesso", "duration", duration.String(), "chars", len(rawResponse))

	// 6. Validação de Segurança (Anti-Alucinação de Conclusão)
	forbiddenPatterns := []string{
		"concluída com sucesso",
		"successfully completed",
		"summary of changes",
		"resumo da implementação",
		"acceptance criteria met",
	}

	lowerResp := strings.ToLower(rawResponse)
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(lowerResp, pattern) {
			logger.Warn("🚨 [BootstrapTaskSpec] Alucinação detectada! IA tentou concluir a tarefa precocemente.", "pattern", pattern)
			return "", fmt.Errorf("IA gerou um relatório de conclusão em vez de uma especificação técnica. Por favor, tente rodar o 'refine' novamente.")
		}
	}

	// 5. Salva a spec gerada
	tasksDir := filepath.Join(o.ProjectPath, ".spec-wizard", "tasks")
	os.MkdirAll(tasksDir, 0755)
	specPath := filepath.Join(tasksDir, fmt.Sprintf("task-%s.md", taskID))

	err = os.WriteFile(specPath, []byte(rawResponse), 0644)
	if err != nil {
		logger.Error("❌ [BootstrapTaskSpec] Erro ao salvar arquivo da tarefa", err, "path", specPath)
		return "", fmt.Errorf("erro ao salvar SPEC da tarefa: %v", err)
	}

	return rawResponse, nil
}

func toSnakeCase(s string) string {
	var res strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			res.WriteByte('_')
		}
		res.WriteRune(r)
	}
	return strings.ToLower(res.String())
}
