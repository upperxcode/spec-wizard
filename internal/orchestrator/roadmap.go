package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"spec-wizard/internal/prompt"

	"path/filepath"
	"strings"
	"time"
)

// CleanJSONResponse extrai o conteúdo JSON de dentro de blocos de código Markdown, se presentes
func CleanJSONResponse(input string) string {
	// 1. Remove blocos de pensamento <thought>...</thought> se existirem
	clean := input
	for {
		startThought := strings.Index(clean, "<thought>")
		endThought := strings.Index(clean, "</thought>")
		if startThought != -1 && endThought != -1 && endThought > startThought {
			clean = clean[:startThought] + clean[endThought+10:]
			continue
		}
		break
	}

	clean = strings.TrimSpace(clean)

	// 2. Busca blocos de código Markdown
	if idx := strings.Index(clean, "```json"); idx != -1 {
		clean = clean[idx+7:]
	} else if idx := strings.Index(clean, "```"); idx != -1 {
		clean = clean[idx+3:]
	}

	// 3. Remove o fechamento do bloco de código
	if idx := strings.LastIndex(clean, "```"); idx != -1 {
		clean = clean[:idx]
	}

	// 4. Limpeza final de caracteres invisíveis
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
func (o *Orchestrator) UpdateTaskStatus(sprintID, taskID int, status string) error {
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
		return fmt.Errorf("tarefa %d na sprint %d não encontrada", taskID, sprintID)
	}

	return o.saveRoadmapToFile(roadmap)
}

// UpdateRoadmapFromCode gera ou atualiza o roadmap baseando-se no estado atual do código (Processo em 2 Fases)
func (o *Orchestrator) UpdateRoadmapFromCode(config ProjectConfig) (*ProjectRoadmap, error) {
	fmt.Printf("🧠 Iniciando Auditoria Técnica em 2 Fases para %s...\n", config.Language)

	fileTree := o.generateFileTree()
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
	o.saveRoadmapToFile(roadmap)
	o.SaveRoadmapToMarkdown(&roadmap)

	return &roadmap, nil
}

// getCriticalContext lê arquivos estratégicos para dar substância à auditoria
func (o *Orchestrator) getCriticalContext() string {
	var sb strings.Builder

	// 1. Pubspec (fundamental para Flutter)
	if data, err := os.ReadFile(filepath.Join(o.ProjectPath, "pubspec.yaml")); err == nil {
		sb.WriteString("FILE: pubspec.yaml\nCONTENT:\n")
		sb.WriteString(string(data))
		sb.WriteString("\n---\n")
	}

	// 2. Peek em controladores (para ver se há lógica real)
	modulesPath := filepath.Join(o.ProjectPath, "lib", "modules")
	entries, err := os.ReadDir(modulesPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				ctrlPath := filepath.Join(modulesPath, entry.Name(), "controllers")
				ctrlFiles, err := os.ReadDir(ctrlPath)
				if err == nil && len(ctrlFiles) > 0 {
					// Lê os primeiros 500 bytes do primeiro controlador encontrado
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

	return sb.String()
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
		sb.WriteString(fmt.Sprintf("## Sprint %d: %s\n", sprint.ID, sprint.Goal))
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
	return os.WriteFile(filepath.Join(wizardDir, "roadmap.md"), []byte(sb.String()), 0644)
}

// PreviewRoadmapPrompt apenas retorna o prompt que seria enviado, sem chamar a IA
func (o *Orchestrator) PreviewRoadmapPrompt(config ProjectConfig) string {
	fileTree := o.generateFileTree()
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
