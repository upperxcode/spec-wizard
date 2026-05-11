package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"spec-wizard/internal/llm"
	"spec-wizard/internal/registry"
	"strings"
)

// FeedbackLoop gerencia a iteração entre validação e correção do código
type FeedbackLoop struct {
	MaxAttempts  int
	ProjectPath  string
	Language     string
	UserLanguage string
	LLMClient    *llm.LLMClient
	Assembler    *ContextAssembler
	Plugin       *registry.ExpertPlugin
}

// ExecutionAttempt representa uma tentativa de execução de uma tarefa
type ExecutionAttempt struct {
	AttemptNumber      int
	OriginalPrompt     string
	AIResponse         string
	SensorResults      []SensorResult
	ValidationPassed   bool
	CorrectionPrompt   string
	CorrectionResponse string
	Diff               string // Novo campo para o diff git
}

func NewFeedbackLoop(projectPath, language, userLanguage string, llmClient *llm.LLMClient, plugin *registry.ExpertPlugin) *FeedbackLoop {
	return &FeedbackLoop{
		MaxAttempts:  3,
		ProjectPath:  projectPath,
		Language:     language,
		UserLanguage: userLanguage,
		LLMClient:    llmClient,
		Assembler:    NewContextAssembler(projectPath),
		Plugin:       plugin,
	}
}

// ExecuteTaskWithFeedback executa uma tarefa com validação e auto-correção
func (fl *FeedbackLoop) ExecuteTaskWithFeedback(ctx context.Context, taskCtx *TaskContext, task Task, sprint Sprint) (*ExecutionAttempt, error) {
	attempt := &ExecutionAttempt{
		AttemptNumber: 1,
	}

	// 1. Primeira tentativa: gera o código
	fmt.Printf("🚀 Tentativa %d: Executando tarefa %s\n", attempt.AttemptNumber, task.Title)

	prompt := fl.Assembler.BuildPromptForTask(taskCtx)
	attempt.OriginalPrompt = prompt

	aiResponse, err := fl.LLMClient.AskWithContext(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("erro na execução inicial: %v", err)
	}

	attempt.AIResponse = aiResponse

	// 1.1 Aplica as mudanças no sistema de arquivos
	if err := fl.applyAIChanges(aiResponse); err != nil {
		return nil, fmt.Errorf("falha ao aplicar mudanças da IA: %v", err)
	}

	// 2. Valida o código gerado
	fmt.Println("🔍 Validando código gerado...")
	sensor := NewSensorManager(fl.ProjectPath, fl.Language, fl.Plugin)
	if err := sensor.RunValidation(nil); err != nil {
		return nil, fmt.Errorf("erro ao rodar sensores: %v", err)
	}

	attempt.SensorResults = sensor.SensorResults

	// 3. Se passou na validação, retorna sucesso
	if !sensor.HasFailures() {
		attempt.ValidationPassed = true
		attempt.Diff = fl.captureDiff()
		fmt.Println("✅ Código validado com sucesso na primeira tentativa!")
		return attempt, nil
	}

	// 4. Se falhou, tenta corrigir automaticamente (até MaxAttempts)
	fmt.Printf("⚠️  Validação falhou. Iniciando loop de auto-correção...\n")

	for attempt.AttemptNumber = 2; attempt.AttemptNumber <= fl.MaxAttempts; attempt.AttemptNumber++ {
		fmt.Printf("\n🔄 Tentativa %d de %d: Corrigindo erros\n", attempt.AttemptNumber, fl.MaxAttempts)

		// Monta o prompt de correção com os erros dos sensores
		correctionPrompt := fl.buildCorrectionPrompt(taskCtx, attempt)
		attempt.CorrectionPrompt = correctionPrompt

		correctionResponse, err := fl.LLMClient.AskWithContext(ctx, correctionPrompt, nil)
		if err != nil {
			return nil, fmt.Errorf("erro na tentativa de correção %d: %v", attempt.AttemptNumber, err)
		}

		attempt.CorrectionResponse = correctionResponse
		attempt.AIResponse = correctionResponse // Atualiza com a versão corrigida

		// 4.1 Aplica as mudanças corrigidas
		if err := fl.applyAIChanges(correctionResponse); err != nil {
			return nil, fmt.Errorf("falha ao aplicar correções da IA: %v", err)
		}

		// Valida novamente
		fmt.Println("🔍 Revalidando código corrigido...")
		sensor := NewSensorManager(fl.ProjectPath, fl.Language, fl.Plugin)
		if err := sensor.RunValidation(nil); err != nil {
			fmt.Printf("❌ Sensores falharam na tentativa %d\n", attempt.AttemptNumber)
			continue
		}

		attempt.SensorResults = sensor.SensorResults

		if !sensor.HasFailures() {
			attempt.ValidationPassed = true
			attempt.Diff = fl.captureDiff()
			fmt.Printf("✅ Código corrigido e validado na tentativa %d!\n", attempt.AttemptNumber)
			return attempt, nil
		}

		fmt.Printf("❌ Validação falhou na tentativa %d. Gerando nova correção...\n", attempt.AttemptNumber)
	}

	// 5. Se chegou aqui, esgotou as tentativas
	fmt.Printf("❌ Falha crítica: Não foi possível corrigir o código após %d tentativas.\n", fl.MaxAttempts)
	return attempt, fmt.Errorf("código não passou na validação após %d tentativas", fl.MaxAttempts)
}

// buildCorrectionPrompt constrói um prompt pedindo ao modelo que corrija os erros
func (fl *FeedbackLoop) buildCorrectionPrompt(taskCtx *TaskContext, attempt *ExecutionAttempt) string {
	prompt := strings.Builder{}

	prompt.WriteString("# 🔄 TAREFA DE CORREÇÃO - JANELA LIMPA\n\n")

	prompt.WriteString("Você gerou código anterior que não passou na validação.\n")
	prompt.WriteString("Abaixo estão os ERROS ESPECÍFICOS que você precisa corrigir:\n\n")

	// Agrupa os erros por sensor
	for _, result := range attempt.SensorResults {
		if result.Status == "fail" {
			prompt.WriteString(fmt.Sprintf("## ❌ %s\n", result.SensorName))
			prompt.WriteString(fmt.Sprintf("**Erro:** %s\n\n", result.Message))
			if result.ErrorLog != "" {
				prompt.WriteString("```\n")
				prompt.WriteString(result.ErrorLog)
				prompt.WriteString("\n```\n\n")
			}
		}
	}

	prompt.WriteString("---\n\n")
	prompt.WriteString("## 🔒 Golden Rules (Não Negocie)\n")
	prompt.WriteString(taskCtx.Skills)
	prompt.WriteString("\n\n")

	prompt.WriteString("## 📋 Sua Tarefa Corrigida\n")
	prompt.WriteString("**Título:** " + taskCtx.TaskTitle + "\n")
	prompt.WriteString("**Descrição:** " + taskCtx.TaskDescription + "\n\n")

	if len(taskCtx.AcceptanceCriteria) > 0 {
		prompt.WriteString("**Critérios de Aceitação (Obrigatórios):**\n")
		for i, criteria := range taskCtx.AcceptanceCriteria {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, criteria))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("---\n\n")
	prompt.WriteString("## ⚠️ Seu Código Anterior\n")
	prompt.WriteString("```\n")
	prompt.WriteString(attempt.AIResponse)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("## 📝 Instruções de Correção\n\n")
	prompt.WriteString("1. **Identifique** cada erro listado acima\n")
	prompt.WriteString("2. **Corrija** o código original para resolver os problemas\n")
	prompt.WriteString("3. **Mantenha** a estrutura geral e a lógica\n")
	prompt.WriteString("4. **Siga** rigorosamente as Golden Rules\n")
	prompt.WriteString("5. **Forneça** o código completo e corrigido\n\n")

	prompt.WriteString("Responda com o código corrigido agora.\n")
	prompt.WriteString(fmt.Sprintf("TUDO (descrições de erro, logs, etc) deve ser tratado com o usuário final em %s.\n", strings.ToUpper(fl.UserLanguage)))

	return prompt.String()
}

// SaveExecutionLog persiste todos os detalhes da execução (com tentativas)
func (fl *FeedbackLoop) SaveExecutionLog(projectPath string, taskID string, attempt *ExecutionAttempt) error {
	logFile := fmt.Sprintf("%s-execution-log.md", taskID)
	logPath := fmt.Sprintf("%s/.spec-wizard/task-logs/%s", projectPath, logFile)

	var logContent strings.Builder

	logContent.WriteString("# 📊 Execution Log\n\n")
	logContent.WriteString(fmt.Sprintf("**Task ID:** %s\n", taskID))
	logContent.WriteString(fmt.Sprintf("**Status:** %v\n", attempt.ValidationPassed))
	logContent.WriteString(fmt.Sprintf("**Attempts:** %d\n\n", attempt.AttemptNumber))

	logContent.WriteString("## Tentativa 1 (Inicial)\n")
	logContent.WriteString("### Prompt\n```\n")
	logContent.WriteString(attempt.OriginalPrompt)
	logContent.WriteString("\n```\n\n")

	logContent.WriteString("### Resposta da IA\n```\n")
	logContent.WriteString(attempt.AIResponse)
	logContent.WriteString("\n```\n\n")

	logContent.WriteString("### Resultados de Validação\n")
	for _, result := range attempt.SensorResults {
		logContent.WriteString(fmt.Sprintf("- **%s**: %s\n", result.SensorName, result.Status))
	}

	if attempt.AttemptNumber > 1 {
		logContent.WriteString(fmt.Sprintf("\n## Tentativa %d+ (Correção)\n", attempt.AttemptNumber))
		logContent.WriteString("### Prompt de Correção\n```\n")
		logContent.WriteString(attempt.CorrectionPrompt)
		logContent.WriteString("\n```\n\n")

		logContent.WriteString("### Resposta Corrigida\n```\n")
		logContent.WriteString(attempt.CorrectionResponse)
		logContent.WriteString("\n```\n")
	}

	return fl.Assembler.saveFile(logPath, logContent.String())
}

// saveFile helper para salvar arquivo
func (ca *ContextAssembler) saveFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// applyAIChanges extrai blocos de código markdown e salva no filesystem
func (fl *FeedbackLoop) applyAIChanges(aiResponse string) error {
	// Procuramos por padrões como:
	// FILE: caminho/do/arquivo.ext
	// ```language
	// código
	// ```

	lines := strings.Split(aiResponse, "\n")
	var currentFile string
	var currentCode strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detecta o marcador de arquivo (pode ser em comentários ou texto livre)
		if strings.Contains(strings.ToUpper(trimmed), "FILE:") || strings.Contains(strings.ToUpper(trimmed), "PATH:") {
			parts := strings.Split(trimmed, ":")
			if len(parts) > 1 {
				currentFile = strings.TrimSpace(parts[1])
				// Limpa possíveis backticks ou aspas
				currentFile = strings.Trim(currentFile, "`\"' ")
			}
		}

		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				currentCode.Reset()
			} else {
				inCodeBlock = false
				if currentFile != "" {
					fullPath := filepath.Join(fl.ProjectPath, currentFile)

					// Cria diretórios se necessário
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return err
					}

					// Salva o arquivo
					if err := os.WriteFile(fullPath, []byte(currentCode.String()), 0644); err != nil {
						return err
					}
					fmt.Printf("💾 Arquivo salvo: %s\n", currentFile)
					currentFile = "" // Reset para o próximo bloco
				}
			}
			continue
		}

		if inCodeBlock {
			currentCode.WriteString(line + "\n")
		}
	}

	return nil
}
func (fl *FeedbackLoop) captureDiff() string {
	// Verifica se é um repositório git
	if _, err := os.Stat(filepath.Join(fl.ProjectPath, ".git")); os.IsNotExist(err) {
		return "Git não inicializado. Não foi possível gerar diff."
	}

	// Executa git diff
	c := exec.Command("git", "diff")
	c.Dir = fl.ProjectPath
	diffOut, err := c.Output()
	if err != nil {
		return "Erro ao capturar diff: " + err.Error()
	}

	if len(diffOut) == 0 {
		return "Nenhuma alteração detectada via git diff."
	}

	return string(diffOut)
}
