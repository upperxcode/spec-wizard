package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"spec-wizard/internal/llm"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/prompt"

	"path/filepath"
	"strings"
	"time"
)

// CleanJSONResponse extrai o conteúdo JSON de dentro de blocos de código Markdown, se presentes
func CleanJSONResponse(input string) string {
	// 1. Tenta encontrar o bloco {...} mais provável usando o primeiro { e o último }
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")

	if start != -1 && end != -1 && end > start {
		candidate := input[start : end+1]
		// Remove marcações markdown que possam ter ficado dentro (ex: ```json { ... } ```)
		candidate = strings.ReplaceAll(candidate, "```json", "")
		candidate = strings.ReplaceAll(candidate, "```", "")
		return strings.TrimSpace(candidate)
	}

	// 2. Fallback: Limpeza básica de markdown
	clean := input
	if idx := strings.Index(clean, "```json"); idx != -1 {
		clean = clean[idx+7:]
	} else if idx := strings.Index(clean, "```"); idx != -1 {
		clean = clean[idx+3:]
	}

	if idx := strings.LastIndex(clean, "```"); idx != -1 {
		clean = clean[:idx]
	}

	return strings.TrimSpace(clean)
}

// GenerateInitialRoadmap coordena a criação do plano de ação via IA
func (o *Orchestrator) GenerateInitialRoadmap(config ProjectConfig) (*ProjectRoadmap, error) {
	fmt.Printf("🧠 Gerando Roadmap estratégico para %s (%v)...\n", config.Language, config.Patterns)

	factory := prompt.NewPromptFactory()
	roadmapPrompt := factory.CreateRoadmapPrompt(prompt.RoadmapContext{
		Language:               config.Language,
		Patterns:               config.Patterns,
		Architecture:           config.Architecture,
		DataStrategy:           config.DataStrategy,
		StateManagement:        config.StateManagement,
		Customization:          config.Customization,
		AdditionalInstructions: config.Instructions,
		UserLanguage:           config.UserLanguage,
	})

	// Usa o factory do orchestrator para obter o cliente correto
	llmClient, err := o.GetLLMClient(config)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter LLM Client: %v", err)
	}
	rawResponse, err := llmClient.Ask(roadmapPrompt)
	if err != nil {
		return nil, err
	}
	if rawResponse == "" {
		return nil, fmt.Errorf("a IA retornou uma resposta vazia")
	}
	cleanedJSON := CleanJSONResponse(rawResponse)
	if cleanedJSON == "" {
		return nil, fmt.Errorf("a IA retornou uma resposta vazia ou incompleta | Resposta Bruta: %s", rawResponse)
	}

	var roadmap ProjectRoadmap
	if err := json.Unmarshal([]byte(cleanedJSON), &roadmap); err != nil {
		return nil, fmt.Errorf("erro ao processar JSON da IA: %v | Raw: %s", err, cleanedJSON)
	}

	// Persiste o arquivo para que o Dashboard e a IA possam consultar
	if err := o.saveRoadmapToFile(roadmap); err != nil {
		return nil, err
	}

	// Persiste também a documentação base (SPEC, PRD, Skills) para contexto futuro
	o.SaveAgentsDocs(config, nil)

	return &roadmap, nil
}

// saveRoadmapToFile grava o sprints.json na pasta de ancoragem
func (o *Orchestrator) saveRoadmapToFile(roadmap ProjectRoadmap) error {
	wizardDir := filepath.Join(o.ProjectPath, ".spec-wizard")
	if err := os.MkdirAll(wizardDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(roadmap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(wizardDir, "sprints.json"), data, 0644)
}

// SaveRoadmap exporta o estado do roadmap para JSON e MD
func (o *Orchestrator) SaveRoadmap(roadmap ProjectRoadmap) error {
	if err := o.saveRoadmapToFile(roadmap); err != nil {
		return err
	}
	return o.SaveRoadmapToMarkdown(&roadmap)
}

// UpdateTaskStatus permite que sensores ou o usuário marquem o progresso
func (o *Orchestrator) UpdateTaskStatus(sprintID, taskID FlexibleID, status string) error {
	path := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var roadmap ProjectRoadmap
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return err
	}

	// Atualiza a tarefa específica
	found := false
	for i := range roadmap.Sprints {
		if roadmap.Sprints[i].ID == sprintID {
			for j := range roadmap.Sprints[i].Tasks {
				if roadmap.Sprints[i].Tasks[j].ID == taskID {
					roadmap.Sprints[i].Tasks[j].Status = status
					found = true
					break
				}
			}
		}
	}

	if !found {
		return fmt.Errorf("tarefa %s na sprint %s não encontrada", taskID, sprintID)
	}

	return o.SaveRoadmap(roadmap)
}

// UpdateRoadmapFromCode gera ou atualiza o roadmap baseando-se no estado atual do código (Processo em 2 Fases)
func (o *Orchestrator) UpdateRoadmapFromCode(config ProjectConfig) (*ProjectRoadmap, error) {
	fmt.Printf("🧠 Iniciando Auditoria Técnica em 2 Fases para %s...\n", config.Language)

	fileTree := o.generateFileTree(config.ExcludePatterns)
	criticalContext := o.getCriticalContext()

	prd, _ := o.readProjectDoc(o.ProjectPath, "PRD.md")
	spec, _ := o.readProjectDoc(o.ProjectPath, "SPEC.md")
	skills, _ := o.readProjectDoc(o.ProjectPath, "skills.md")
	roadmapMD, _ := o.readProjectDoc(o.ProjectPath, "roadmap.md")

	factory := prompt.NewPromptFactory()
	llmClient, err := o.GetLLMClient(config)
	if err != nil {
		return nil, err
	}

	// --- FASE 1: MAPEAMENTO DE MÓDULOS ---
	// Injetamos o criticalContext junto com a fileTree para dar "visão" à IA
	mappingPrompt := factory.CreateModuleMappingPrompt(prompt.RoadmapContext{
		Language:     config.Language,
		PRD:          prd,
		Spec:         spec,
		UserLanguage: config.UserLanguage,
	}, fileTree+"\n\n### CRITICAL FILE CONTENT (PEEK):\n"+criticalContext)

	mappingResponse, err := llmClient.Ask(mappingPrompt)
	if err != nil {
		return nil, err
	}
	if mappingResponse == "" {
		return nil, fmt.Errorf("a IA retornou uma resposta vazia")
	}
	if err != nil {
		return nil, fmt.Errorf("erro na Fase 1 (Mapeamento): %v", err)
	}
	moduleMapping := CleanJSONResponse(mappingResponse)
	os.WriteFile(filepath.Join(o.ProjectPath, ".spec-wizard", "project_mapping.json"), []byte(moduleMapping), 0644)

	// --- FASE 2: GERAÇÃO DO ROADMAP ---
	roadmapPrompt := factory.CreateUpdateRoadmapPrompt(prompt.RoadmapContext{
		Language:               config.Language,
		Patterns:               config.Patterns,
		Architecture:           config.Architecture,
		DataStrategy:           config.DataStrategy,
		StateManagement:        config.StateManagement,
		Customization:          config.Customization,
		AdditionalInstructions: config.Instructions,
		PRD:                    prd,
		Spec:                   spec,
		Skills:                 skills,
		ExistingRoadmap:        roadmapMD,
		UserLanguage:           config.UserLanguage,
	}, fileTree, moduleMapping)

	roadmapResponse, err := llmClient.Ask(roadmapPrompt)
	if err != nil {
		return nil, err
	}
	if roadmapResponse == "" {
		return nil, fmt.Errorf("a IA retornou uma resposta vazia")
	}
	if err != nil {
		return nil, fmt.Errorf("erro na Fase 2 (Roadmap): %v", err)
	}

	cleanedJSON := CleanJSONResponse(roadmapResponse)

	var roadmap ProjectRoadmap
	if err := json.Unmarshal([]byte(cleanedJSON), &roadmap); err != nil {
		return nil, fmt.Errorf("erro ao processar JSON da IA: %v | Raw: %s", err, cleanedJSON)
	}

	config.RoadmapLastUpdated = time.Now().Format("2006-01-02 15:04:05")
	o.SaveAgentsDocs(config, nil)
	o.SaveRoadmap(roadmap)
	return &roadmap, nil
}

// getCriticalContext lê arquivos estratégicos para dar substância à auditoria
func (o *Orchestrator) getCriticalContext() string {
	var sb strings.Builder

	// 1. Evidência de Raiz (Arquivos de Dependências)
	rootFiles := []string{"go.mod", "package.json", "composer.json", "pubspec.yaml", "requirements.txt", "Gemfile"}
	for _, rf := range rootFiles {
		if data, err := os.ReadFile(filepath.Join(o.ProjectPath, rf)); err == nil {
			sb.WriteString(fmt.Sprintf("FILE: %s (Root Evidence)\nCONTENT:\n", rf))
			// Pega apenas os primeiros 1000 bytes para prova de identidade
			content := string(data)
			if len(content) > 6000 {
				content = content[:6000] + "\n... [TRUNCATED]"
			}
			sb.WriteString(content)
			sb.WriteString("\n---\n")
		}
	}

	// 2. Peek em controladores/serviços para ver se há lógica real (Flutter específico)
	modulesPath := filepath.Join(o.ProjectPath, "lib", "modules")
	if entries, err := os.ReadDir(modulesPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				ctrlPath := filepath.Join(modulesPath, entry.Name(), "controllers")
				ctrlFiles, err := os.ReadDir(ctrlPath)
				if err == nil && len(ctrlFiles) > 0 {
					target := filepath.Join(ctrlPath, ctrlFiles[0].Name())
					if data, err := os.ReadFile(target); err == nil {
						content := string(data)
						if len(content) > 500 {
							content = content[:500] + "... [TRUNCATED]"
						}
						sb.WriteString(fmt.Sprintf("FILE PEEK: %s/%s\nCONTENT:\n%s\n---\n", entry.Name(), ctrlFiles[0].Name(), content))
					}
				}
			}
		}
	}

	// 3. Peek em arquitetura Go (se existir e não for projeto Flutter)
	internalPath := filepath.Join(o.ProjectPath, "internal")
	if entries, err := os.ReadDir(internalPath); err == nil && !strings.Contains(sb.String(), "lib/modules") {
		sb.WriteString("ARCHITECTURE DETECTION: Found 'internal' directory.\n")
		for i, entry := range entries {
			if i > 3 {
				break
			} // Apenas os primeiros 4 para evidência
			sb.WriteString(fmt.Sprintf("- internal/%s\n", entry.Name()))
		}
		sb.WriteString("---\n")
	}

	return sb.String()
}

// AuditResult representa o resultado de uma auditoria de tarefa via IA
type AuditResult struct {
	Status       string   `json:"status"`
	Confidence   float64  `json:"confidence"`
	FoundFiles   []string `json:"found_files"`
	Reasoning    string   `json:"reasoning"`
	MissingLogic []string `json:"missing_logic"`
	Feedback     string   `json:"feedback"`
}

// AuditTask verifica se uma tarefa específica já foi implementada no código
func (o *Orchestrator) AuditTask(ctx context.Context, sprintID, taskID FlexibleID, config ProjectConfig) (*AuditResult, error) {
	fmt.Printf("🔍 Auditando tarefa %s da Sprint %s...\n", taskID, sprintID)

	var roadmap ProjectRoadmap

	// 1. Tenta carregar do config.Roadmap se disponível
	if config.Roadmap != nil {
		// Converte interface{} para ProjectRoadmap
		roadmapData, _ := json.Marshal(config.Roadmap)
		json.Unmarshal(roadmapData, &roadmap)
	}

	// 2. Fallback para sprints.json se o roadmap estiver vazio
	if len(roadmap.Sprints) == 0 {
		path := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")
		data, err := os.ReadFile(path)
		if err == nil {
			json.Unmarshal(data, &roadmap)
		}
	}

	if len(roadmap.Sprints) == 0 {
		return nil, fmt.Errorf("roadmap não encontrado ou vazio")
	}

	var targetTask *Task
	var targetSprint *Sprint
	for i := range roadmap.Sprints {
		if roadmap.Sprints[i].ID == sprintID {
			targetSprint = &roadmap.Sprints[i]
			for j := range roadmap.Sprints[i].Tasks {
				if roadmap.Sprints[i].Tasks[j].ID == taskID {
					targetTask = &roadmap.Sprints[i].Tasks[j]
					break
				}
			}
		}
	}

	if targetTask == nil {
		return nil, fmt.Errorf("tarefa %s não encontrada na Sprint %s", taskID, sprintID)
	}

	// 2. Monta o contexto da tarefa (Clean Window)
	assembler := NewContextAssembler(o.ProjectPath)
	taskContext, err := assembler.AssembleTaskContext(*targetTask, *targetSprint, o.Plugins)
	if err != nil {
		return nil, err
	}

	// 3. Gera o prompt de auditoria
	factory := prompt.NewPromptFactory()
	auditPrompt := factory.GenerateAuditTaskPrompt(prompt.ExecutionContext{
		TaskTitle:          targetTask.Title,
		TaskDescription:    targetTask.Description,
		AcceptanceCriteria: targetTask.AcceptanceCriteria,
		Spec:               taskContext.Spec,
		FileTree:           o.generateFileTree(config.ExcludePatterns),
		KnowledgeBase:      o.getCriticalContext(), // Reutiliza o peek de arquivos para evidência
		Language:           config.Language,
		UserLanguage:       config.UserLanguage,
	})

	// 4. Consulta a IA
	llmClient, err := o.GetLLMClient(config)
	if err != nil {
		return nil, err
	}

	rawResponse, err := llmClient.AskWithContext(ctx, auditPrompt, nil)
	if err != nil {
		reasoning := fmt.Sprintf("Erro de processamento: %v", err)
		if err == llm.ErrEmptyResponse {
			reasoning = "O modelo de IA retornou uma resposta vazia. Isso geralmente ocorre por saturação de contexto ou instabilidade do provedor."
		}

		logger.Warn("⚠️ Falha na consulta inicial da auditoria. Retornando fallback.", "error", err)
		return &AuditResult{
			Status:     "pending",
			Confidence: 0,
			Reasoning:  reasoning,
			Feedback:   "A auditoria automática não conseguiu obter uma resposta válida. Tente simplificar a tarefa ou mudar o modelo nas configurações.",
		}, nil
	}

	logger.Info("✅ Resposta da IA recebida para auditoria", "length", len(rawResponse))

	cleanedJSON := CleanJSONResponse(rawResponse)
	var result AuditResult
	if err := json.Unmarshal([]byte(cleanedJSON), &result); err != nil {
		logger.Info("⚠️ Falha ao decodificar JSON da auditoria. Tentando correção automática...", "error", err)
		correctionMsg := fmt.Sprintf("Your previous response was not a valid JSON. Error: %v. Please provide ONLY the raw JSON object for the audit, without any additional text, reasoning or markdown blocks. Ensure all fields are present: status, confidence, found_files, reasoning, missing_logic.", err)

		// Faz uma segunda tentativa com o erro como feedback para a IA
		// Montamos o histórico correto: [Prompt Original] -> [Resposta Errada] -> [Pedido de Correção]
		history := []llm.Message{
			{Role: "user", Content: auditPrompt},
			{Role: "assistant", Content: rawResponse},
		}

		rawResponse, err = llmClient.AskWithContext(ctx, correctionMsg, nil, history...)

		if err == nil {
			cleanedJSON = CleanJSONResponse(rawResponse)
			err = json.Unmarshal([]byte(cleanedJSON), &result)
		}

		if err != nil {
			logger.Warn("❌ Falha na auditoria após autocorreção. Retornando estado de fallback.", "error", err)
			return &AuditResult{
				Status:     "pending",
				Confidence: 0,
				Reasoning:  fmt.Sprintf("Incapacidade técnica do modelo atual: %v", err),
				Feedback:   "A auditoria automática falhou em processar a resposta deste modelo. Você pode tentar novamente com um modelo diferente ou marcar a tarefa como concluída manualmente.",
			}, nil
		}
		logger.Info("✅ JSON corrigido com sucesso após tentativa de autocorreção")
	}

	// 5. Se a IA confirmou que está "completed" e a confiança é alta, atualiza o status
	if result.Status == "completed" && result.Confidence >= 0.7 {
		o.UpdateTaskStatus(sprintID, taskID, "completed")
	} else if result.Status == "in_progress" {
		o.UpdateTaskStatus(sprintID, taskID, "in_progress")
	}

	return &result, nil
}

// SaveRoadmapToMarkdown gera uma versão amigável do roadmap em .md
func (o *Orchestrator) SaveRoadmapToMarkdown(roadmap *ProjectRoadmap) error {
	var sb strings.Builder

	sb.WriteString("# 🛣️ STRATEGIC EXECUTION ROADMAP (AUDITED)\n\n")
	sb.WriteString(fmt.Sprintf("> **Project**: GCI Sindico\n"))
	sb.WriteString(fmt.Sprintf("> **Last Audit**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
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
			}

			sb.WriteString(fmt.Sprintf("### %s %s `[%s]`\n", statusIcon, task.Title, task.Priority))
			sb.WriteString(fmt.Sprintf("%s\n\n", task.Description))

			if len(task.AcceptanceCriteria) > 0 {
				sb.WriteString("**Acceptance Criteria:**\n")
				for _, c := range task.AcceptanceCriteria {
					sb.WriteString(fmt.Sprintf("- [ ] %s\n", c))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("*End of Audited Strategic Roadmap*\n")

	wizardDir := filepath.Join(o.ProjectPath, ".spec-wizard")
	os.WriteFile(filepath.Join(wizardDir, "roadmap.md"), []byte(sb.String()), 0644)

	// Também salva na raiz do projeto para visibilidade imediata do usuário
	return os.WriteFile(filepath.Join(o.ProjectPath, "roadmap.md"), []byte(sb.String()), 0644)
}

// generateStrategicRoadmap gera o conteúdo do roadmap baseado nos inputs
func (o *Orchestrator) generateStrategicRoadmap(config ProjectConfig) (string, error) {
	fileTree := o.generateFileTree(config.ExcludePatterns)
	prd, _ := o.readProjectDoc(o.ProjectPath, "PRD.md")
	spec, _ := o.readProjectDoc(o.ProjectPath, "SPEC.md")

	factory := prompt.NewPromptFactory()
	mappingPrompt := factory.CreateModuleMappingPrompt(prompt.RoadmapContext{
		Language:     config.Language,
		PRD:          prd,
		Spec:         spec,
		UserLanguage: config.UserLanguage,
	}, fileTree)

	return mappingPrompt, nil
}

// PreviewRoadmapPrompt apenas retorna o prompt que seria enviado, sem chamar a IA
func (o *Orchestrator) PreviewRoadmapPrompt(config ProjectConfig) string {
	fileTree := o.generateFileTree(config.ExcludePatterns)
	prd, _ := o.readProjectDoc(o.ProjectPath, "PRD.md")
	spec, _ := o.readProjectDoc(o.ProjectPath, "SPEC.md")
	skills, _ := o.readProjectDoc(o.ProjectPath, "skills.md")
	roadmapMD, _ := o.readProjectDoc(o.ProjectPath, "roadmap.md")

	factory := prompt.NewPromptFactory()

	// Fase 1 Mock
	mappingPrompt := factory.CreateModuleMappingPrompt(prompt.RoadmapContext{
		Language:     config.Language,
		PRD:          prd,
		Spec:         spec,
		UserLanguage: config.UserLanguage,
	}, fileTree)

	// Fase 2 Mock
	roadmapPrompt := factory.CreateUpdateRoadmapPrompt(prompt.RoadmapContext{
		Language:               config.Language,
		Patterns:               config.Patterns,
		Architecture:           config.Architecture,
		DataStrategy:           config.DataStrategy,
		StateManagement:        config.StateManagement,
		Customization:          config.Customization,
		AdditionalInstructions: config.Instructions,
		PRD:                    prd,
		Spec:                   spec,
		Skills:                 skills,
		ExistingRoadmap:        roadmapMD,
		UserLanguage:           config.UserLanguage,
	}, fileTree, "[PHASE 1 MAPPING DATA WILL BE INJECTED HERE]")

	return fmt.Sprintf("--- PHASE 1: MODULE MAPPING PROMPT ---\n%s\n\n--- PHASE 2: STRATEGIC ROADMAP PROMPT ---\n%s", mappingPrompt, roadmapPrompt)
}
