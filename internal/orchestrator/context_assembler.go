package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/registry"
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
		spec = "[Nenhuma especificação técnica encontrada em .spec-wizard/SPEC.md]"
	}

	// 2. Carrega Skills (Golden Rules)
	skills, _ := ca.loadFile("skills.md")
	if skills == "" {
		skills = "[Nenhuma regra de ouro encontrada em .spec-wizard/skills.md]"
	}

	// 3. Carrega PRD.md para contexto de negócio
	prd, _ := ca.loadFile("PRD.md")
	if prd == "" {
		prd = "[Nenhum requisito de negócio encontrado em .spec-wizard/PRD.md]"
	}

	// 4. Busca contexto relevante na Base de Conhecimento
	km := NewKnowledgeManager(ca.ProjectPath)
	knowledge := km.GetRelevantContext(task.Title, task.Description)

	// 5. Carrega Spec específica da tarefa (se existir)
	taskSpecFile := fmt.Sprintf("tasks/task-%s.md", task.ID)
	taskSpec, _ := ca.loadFile(taskSpecFile)
	if taskSpec == "" {
		taskSpec = "[Nenhuma especificação personalizada encontrada para esta tarefa específica. Siga os padrões globais.]"
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
		PreviousContext: map[string]string{
			"business_requirements": prd,
		},
	}

	return taskContext, nil
}

// BuildPromptForTask constrói o prompt final para a IA executar uma tarefa
func (ca *ContextAssembler) BuildPromptForTask(taskCtx *TaskContext) string {
	prompt := strings.Builder{}

	prompt.WriteString("# 🎯 TAREFA DE DESENVOLVIMENTO - JANELA LIMPA\n\n")

	// Sistema de Instruções
	prompt.WriteString("## Sistema de Instruções\n")
	prompt.WriteString("Você é um desenvolvedor sênior. Você deve completar esta tarefa seguindo RIGOROSAMENTE as regras ouro abaixo.\n")
	prompt.WriteString("Responda com código pronto para produção, testado e documentado.\n\n")

	// Regras de Ouro
	prompt.WriteString("## 🔒 Golden Rules (Não Negocie)\n")
	prompt.WriteString(taskCtx.Skills)
	prompt.WriteString("\n\n")

	// Contexto do Workspace (Árvore de Arquivos)
	prompt.WriteString("## 📂 Estrutura Atual do Workspace\n")
	if tree := ca.generateSimpleTree(ca.ProjectPath, 3); tree != "" {
		prompt.WriteString("```\n" + tree + "```\n\n")
	}

	// Especificação Técnica
	prompt.WriteString("## 📐 Especificação Técnica do Projeto\n")
	prompt.WriteString(taskCtx.Spec)
	prompt.WriteString("\n\n")

	// Especificação Específica da Tarefa
	if taskCtx.TaskSpec != "" && !strings.Contains(taskCtx.TaskSpec, "Nenhuma especificação personalizada encontrada") {
		prompt.WriteString("## 📋 Especificação Detalhada da Tarefa (Contrato de Implementação)\n")
		prompt.WriteString("⚠️ **ESTA SEÇÃO TEM PRECEDÊNCIA SOBRE A SPEC GLOBAL PARA ESTA TAREFA** ⚠️\n")
		prompt.WriteString(taskCtx.TaskSpec)
		prompt.WriteString("\n\n")
	}

	// Contexto de Negócio
	prompt.WriteString("## 💼 Contexto de Negócio\n")
	prompt.WriteString(taskCtx.PreviousContext["business_requirements"])
	prompt.WriteString("\n\n")

	// Base de Conhecimento (Se disponível)
	if taskCtx.KnowledgeContent != "" {
		prompt.WriteString(taskCtx.KnowledgeContent)
		prompt.WriteString("\n\n")
	}

	// DEPENDENCY MANIFEST (Opinionated Stack Builder)
	if taskCtx.Stack != nil {
		prompt.WriteString("## 📦 DEPENDENCY MANIFEST (Stack: " + taskCtx.Stack.Name + ")\n")
		prompt.WriteString("Você DEVE utilizar as bibliotecas e frameworks abaixo para implementar esta tarefa.\n\n")

		for _, lib := range taskCtx.Stack.Libraries {
			mandatoryFlag := ""
			if lib.Mandatory {
				mandatoryFlag = " ⚠️ [ESTRITAMENTE OBRIGATÓRIO]"
			}

			prompt.WriteString(fmt.Sprintf("### Biblioteca: %s%s\n", lib.Name, mandatoryFlag))
			prompt.WriteString(fmt.Sprintf("- **Categoria**: %s\n", lib.Category))
			prompt.WriteString(fmt.Sprintf("- **Descrição**: %s\n", lib.Description))
			prompt.WriteString(fmt.Sprintf("- **Comando de Instalação**: `%s`\n", lib.InstallCmd))
			prompt.WriteString("- **Exemplo de Uso**:\n")
			prompt.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", taskCtx.Stack.Language, lib.UsageExample))
		}
	}

	// Tarefa Específica
	prompt.WriteString("## 📋 Sua Tarefa Específica\n")
	prompt.WriteString("### ID da Tarefa: " + taskCtx.TaskID + "\n")
	prompt.WriteString("### Título: " + taskCtx.TaskTitle + "\n")
	prompt.WriteString("### Descrição:\n" + taskCtx.TaskDescription + "\n\n")

	// Critérios de Aceitação
	if len(taskCtx.AcceptanceCriteria) > 0 {
		prompt.WriteString("### ✅ Critérios de Aceitação (Obrigatórios):\n")
		for i, criteria := range taskCtx.AcceptanceCriteria {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, criteria))
		}
		prompt.WriteString("\n")
	}

	// Instruções Finais
	prompt.WriteString("---\n\n")
	prompt.WriteString("## Entrega Esperada\n\n")
	prompt.WriteString("Forneça:\n")
	prompt.WriteString("1. **Código**: Implementação completa da tarefa\n")
	prompt.WriteString("2. **Estrutura**: Nomes de arquivos e pastas onde o código deve ser salvo\n")
	prompt.WriteString("3. **Testes**: Exemplos de teste unitário se aplicável\n")
	prompt.WriteString("4. **Documentação**: Comentários no código explicando decisões críticas\n\n")

	prompt.WriteString("Comece agora.\n")

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
