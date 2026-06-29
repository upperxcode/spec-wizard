package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/governance"
	"strings"
)

// TaskContext representa o contexto limpo (Clean Window) para uma tarefa específica
type TaskContext struct {
	TaskID             string            `json:"task_id"`
	TaskTitle          string            `json:"task_title"`
	TaskDescription    string            `json:"task_description"`
	AcceptanceCriteria []string          `json:"acceptance_criteria"`
	Spec               string            `json:"spec_content"`      // Spec Global
	TaskSpec           string            `json:"task_spec_content"` // Spec Específica da Tarefa
	Skills             string            `json:"skills_content"`
	KnowledgeContent   string            `json:"knowledge_content,omitempty"`
	Stack              *StackPlugin      `json:"stack,omitempty"`
	PreviousContext    map[string]string `json:"previous_context,omitempty"` // Contexto de tarefas anteriores se necessário
	ProjectID          int               `json:"project_id"`            // ID interno do projeto para governança
	IsCodingTask       bool              `json:"is_coding_task"`            // Flag para injetar protocolo de governança
}

// ContextAssembler monta instruções ricas para a IA em uma janela limpa
type ContextAssembler struct {
	ProjectPath string
	WizardPath  string
}

func NewContextAssembler(projectPath string) *ContextAssembler {
	return &ContextAssembler{
		ProjectPath: projectPath,
		WizardPath:  filepath.Join(projectPath, ".spec-wizard"),
	}
}

// AssembleTaskContext monta o contexto completo para uma tarefa específica
func (ca *ContextAssembler) AssembleTaskContext(task Task, sprint Sprint, plugins []*registry.ExpertPlugin) (*TaskContext, error) {
	logger.Info("🧩 Montando contexto da tarefa", "task_id", task.ID, "title", task.Title)

	// 1. Carrega SPEC.md
	spec, _ := ca.loadFile("SPEC.md")
	if spec == "" {
		spec = "[No technical specification found in .spec-wizard/SPEC.md]"
	}


	// 2. Carrega Skills (Golden Rules)
	skills, _ := ca.loadFile("skills.md")
	if skills == "" {
		skills = "[No golden rules found in .spec-wizard/skills.md]"
	}


	// 3. Carrega PRD.md para contexto de negócio
	prd, _ := ca.loadFile("PRD.md")

	// 4. Busca ProjectID no Banco
	pID, _ := governance.GetOrCreateProject(ca.ProjectPath)
	if prd == "" {
		prd = "[No business requirements found in .spec-wizard/PRD.md]"
	}


	// 4. Busca contexto relevante na Base de Conhecimento
	km := NewKnowledgeManager(ca.ProjectPath)
	knowledge := km.GetRelevantContext(task.Title, task.Description)

	// 5. Carrega Spec específica da tarefa (se existir)
	// Primeiro tenta arquivo (legado), depois banco de dados (novo padrão)
	taskSpecFile := fmt.Sprintf("tasks/task-%s.md", task.ID)
	taskSpec, _ := ca.loadFile(taskSpecFile)
	if taskSpec == "" {
		// Busca no banco de dados SQLite
		db := governance.GetDB()
		if db != nil {
			var fullContext, techReqs string
			err := db.QueryRow("SELECT full_context, technical_requirements FROM task_details WHERE task_id = ?", task.ID).Scan(&fullContext, &techReqs)
			if err == nil {
				taskSpec = fullContext + "\n\n### Requisitos Técnicos Adicionais\n" + techReqs
			}
		}
	}

	if taskSpec == "" {
		taskSpec = "[No custom specification found for this specific task. Follow global patterns.]"
	}


	// 6. Carrega Stack Plugin (se houver)
	stack, _ := LoadStackPlugin(filepath.Join(ca.WizardPath, "stack.json"))

	// 7. Monta o contexto da tarefa
	taskContext := &TaskContext{
		TaskID:             fmt.Sprintf("task-%s", task.ID),
		TaskTitle:          task.Title,
		TaskDescription:    task.Description,
		AcceptanceCriteria: task.GetCriteriaStrings(),
		Spec:               spec,
		TaskSpec:           taskSpec,
		Skills:             skills,
		KnowledgeContent:   knowledge,
		Stack:              stack,
		ProjectID:          pID,
		PreviousContext: map[string]string{
			"business_requirements": prd,
		},
	}

	return taskContext, nil
}

// BuildPromptForTask constrói o prompt final para a IA executar uma tarefa
func (ca *ContextAssembler) BuildPromptForTask(taskCtx *TaskContext) string {
	prompt := strings.Builder{}

	prompt.WriteString("# 🎯 DEVELOPMENT TASK - CLEAN WINDOW\n\n")

	// Instruction System
	prompt.WriteString("## Instruction System\n")
	prompt.WriteString("You are a senior developer. You must complete this task by STRICTLY following the golden rules below.\n")
	prompt.WriteString("Provide production-ready, tested, and documented code.\n")
	prompt.WriteString("**IMPORTANT**: Respond in the user's preferred language: pt-BR.\n\n")

	// Golden Rules
	prompt.WriteString("## 🔒 Golden Rules (Do Not Negotiate)\n")
	prompt.WriteString(taskCtx.Skills)
	prompt.WriteString("\n\n")

	// Workspace Context
	prompt.WriteString("## 📂 Current Workspace Structure\n")
	if tree := ca.generateSimpleTree(ca.ProjectPath, 3); tree != "" {
		prompt.WriteString("```\n" + tree + "```\n\n")
	}

	// Technical Specification
	prompt.WriteString("## 📐 Project Technical Specification\n")
	prompt.WriteString(taskCtx.Spec)
	prompt.WriteString("\n\n")

	// Task Specific Specification
	if taskCtx.TaskSpec != "" && !strings.Contains(taskCtx.TaskSpec, "Nenhuma especificação personalizada encontrada") {
		prompt.WriteString("## 📋 Detailed Task Specification (Implementation Contract)\n")
		prompt.WriteString("⚠️ **THIS SECTION HAS PRECEDENCE OVER THE GLOBAL SPEC FOR THIS TASK** ⚠️\n")
		prompt.WriteString(taskCtx.TaskSpec)
		prompt.WriteString("\n\n")
	}

	// Business Context
	prompt.WriteString("## 💼 Business Context\n")
	prompt.WriteString(taskCtx.PreviousContext["business_requirements"])
	prompt.WriteString("\n\n")

	// Knowledge Base
	if taskCtx.KnowledgeContent != "" {
		prompt.WriteString("## 🧠 Knowledge Base Context\n")
		prompt.WriteString(taskCtx.KnowledgeContent)
		prompt.WriteString("\n\n")
	}

	// DEPENDENCY MANIFEST
	if taskCtx.Stack != nil {
		prompt.WriteString("## 📦 DEPENDENCY MANIFEST (Stack: " + taskCtx.Stack.Name + ")\n")
		prompt.WriteString("You MUST use the libraries and frameworks below to implement this task.\n\n")

		for _, lib := range taskCtx.Stack.Libraries {
			mandatoryFlag := ""
			if lib.Mandatory {
				mandatoryFlag = " ⚠️ [STRICTLY MANDATORY]"
			}

			prompt.WriteString(fmt.Sprintf("### Library: %s%s\n", lib.Name, mandatoryFlag))
			prompt.WriteString(fmt.Sprintf("- **Category**: %s\n", lib.Category))
			prompt.WriteString(fmt.Sprintf("- **Description**: %s\n", lib.Description))
			prompt.WriteString(fmt.Sprintf("- **Installation Command**: `%s`\n", lib.InstallCmd))
			prompt.WriteString("- **Usage Example**:\n")
			prompt.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", taskCtx.Stack.Language, lib.UsageExample))
		}
	}

	// Specific Task
	prompt.WriteString("## 📋 Your Specific Task\n")
	prompt.WriteString("### Task ID: " + taskCtx.TaskID + "\n")
	prompt.WriteString("### Title: " + taskCtx.TaskTitle + "\n")
	prompt.WriteString("### Description:\n" + taskCtx.TaskDescription + "\n\n")

	// Acceptance Criteria
	if len(taskCtx.AcceptanceCriteria) > 0 {
		prompt.WriteString("### ✅ Acceptance Criteria (Mandatory):\n")
		for i, criteria := range taskCtx.AcceptanceCriteria {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, criteria))
		}
		prompt.WriteString("\n")
	}

	// Final Instructions
	prompt.WriteString("---\n\n")

	// SE FOR UMA TAREFA DE CODIFICAÇÃO, INJETA O PROTOCOLO DE GOVERNANÇA AUTÔNOMA
	if taskCtx.IsCodingTask {
		prompt.WriteString("## 🛡️ AUTONOMOUS GOVERNANCE PROTOCOL (STRICT)\n")
		prompt.WriteString("You are in AUTONOMOUS CODING MODE. You must execute the following protocol EXACTLY:\n\n")
		
		prompt.WriteString("### 1. Readiness Validation\n")
		prompt.WriteString("- Analyze the 'Detailed Task Specification' provided above.\n")
		prompt.WriteString("- If any technical detail is missing to implement a perfect solution, STOP and ask for clarification.\n\n")

		prompt.WriteString("### 2. Incremental implementation & Database Logging\n")
		prompt.WriteString(fmt.Sprintf("- **Project ID**: Use `%d` for all database operations.\n", taskCtx.ProjectID))
		prompt.WriteString("- **For each Acceptance Criteria implemented**:\n")
		prompt.WriteString("  - Use the tool `wz` with command `db_insert acceptance_criteria {\"project_id\": " + fmt.Sprint(taskCtx.ProjectID) + ", \"task_id\": \"" + strings.TrimPrefix(taskCtx.TaskID, "task-") + "\", \"status\": \"completed\", \"description\": \"Criteria description...\"}` to mark it as DONE immediately after coding.\n")
		prompt.WriteString("- **For each file created/modified**:\n")
		prompt.WriteString("  - Use the tool `wz` with command `db_insert files {\"project_id\": " + fmt.Sprint(taskCtx.ProjectID) + ", \"path\": \"path/to/file.go\"}` to register it.\n")
		prompt.WriteString("  - Link it to the task: `db_insert task_files {\"project_id\": " + fmt.Sprint(taskCtx.ProjectID) + ", \"task_id\": \"" + strings.TrimPrefix(taskCtx.TaskID, "task-") + "\", \"file_id\": \"$FID\", \"file_type\": \"source|test\", \"action\": \"created\"}`.\n\n")

		prompt.WriteString("### 3. Mandatory Testing Gate\n")
		prompt.WriteString("- You MUST create at least one unit test for your implementation.\n")
		prompt.WriteString("- Run the tests using the appropriate shell command (e.g., `go test ./...`).\n")
		prompt.WriteString("- If tests FAIL, you must fix the code before proceeding.\n\n")

		prompt.WriteString("### 4. Final Submission\n")
		prompt.WriteString("- Only after all tests PASS and all files are registered in the database, call the audit endpoint: `/api/project/audit-task`.\n\n")
		
		prompt.WriteString("## Expected Delivery\n\n")
		prompt.WriteString("Provide:\n")
		prompt.WriteString("1. **Governance Execution**: Log of database inserts for criteria and files.\n")
		prompt.WriteString("2. **Production Code**: High-quality implementation.\n")
		prompt.WriteString("3. **Test Code**: Files verifying the implementation.\n")
		prompt.WriteString("4. **Test Report**: Proof that tests passed.\n\n")
	} else {
		prompt.WriteString("## Expected Delivery\n\n")
		prompt.WriteString("Provide:\n")
		prompt.WriteString("1. **Code**: Complete implementation of the task\n")
		prompt.WriteString("2. **Structure**: File names and paths where the code should be saved\n")
		prompt.WriteString("3. **Tests**: Unit test examples if applicable\n")
		prompt.WriteString("4. **Documentation**: Comments in the code explaining critical decisions\n\n")
	}

	prompt.WriteString("Start now. Respond in pt-BR.\n")

	return prompt.String()
}

// loadFile carrega um arquivo da pasta .spec-wizard
func (ca *ContextAssembler) loadFile(filename string) (string, error) {
	filePath := filepath.Join(ca.WizardPath, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveTaskLog salva os detalhes e respostas de uma tarefa para auditoria
func (ca *ContextAssembler) SaveTaskLog(taskID string, aiResponse string) error {
	logsDir := filepath.Join(ca.WizardPath, "task-logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logsDir, fmt.Sprintf("%s.md", taskID))
	return os.WriteFile(logFile, []byte(aiResponse), 0644)
}

func (ca *ContextAssembler) generateSimpleTree(root string, maxDepth int) string {
	var sb strings.Builder
	ca.walkTree(root, "", 0, maxDepth, &sb)
	return sb.String()
}

func (ca *ContextAssembler) walkTree(path, indent string, depth, maxDepth int, sb *strings.Builder) {
	if depth > maxDepth {
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, f := range files {
		name := f.Name()
		// Ignora pastas comuns de build e ocultas (exceto .spec-wizard)
		if (strings.HasPrefix(name, ".") && name != ".spec-wizard") || name == "node_modules" || name == "vendor" || name == "bin" {
			continue
		}

		if f.IsDir() {
			sb.WriteString(fmt.Sprintf("%s📁 %s/\n", indent, name))
			ca.walkTree(filepath.Join(path, name), indent+"  ", depth+1, maxDepth, sb)
		} else {
			sb.WriteString(fmt.Sprintf("%s📄 %s\n", indent, name))
		}
	}
}
