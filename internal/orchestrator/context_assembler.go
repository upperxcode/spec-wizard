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
	Spec               string            `json:"spec_content"`
	Skills             string            `json:"skills_content"`
	KnowledgeContent   string            `json:"knowledge_content,omitempty"`
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

	// 5. Monta o contexto da tarefa
	taskContext := &TaskContext{
		TaskID:             fmt.Sprintf("task-%s", task.ID),
		TaskTitle:          task.Title,
		TaskDescription:    task.Description,
		AcceptanceCriteria: task.AcceptanceCriteria,
		Spec:               spec,
		Skills:             skills,
		KnowledgeContent:   knowledge,
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

	// Contexto de Negócio
	prompt.WriteString("## 💼 Contexto de Negócio\n")
	prompt.WriteString(taskCtx.PreviousContext["business_requirements"])
	prompt.WriteString("\n\n")

	// Base de Conhecimento (Se disponível)
	if taskCtx.KnowledgeContent != "" {
		prompt.WriteString(taskCtx.KnowledgeContent)
		prompt.WriteString("\n\n")
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
