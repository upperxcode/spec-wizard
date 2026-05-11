package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/registry"
	"strings"
	"time"
)

// RunSensor executa a suite de testes focada nos arquivos alterados ou no projeto completo
func (o *Orchestrator) RunSensor(lang string, touchedFiles []string) (bool, string, error) {
	var cmd *exec.Cmd
	lang = strings.ToLower(lang)

	// Identifica os pacotes/diretórios afetados para testes focados
	affectedPkgs := make(map[string]bool)
	for _, f := range touchedFiles {
		dir := filepath.Dir(f)
		if dir != "." && dir != "" {
			affectedPkgs[dir] = true
		}
	}

	// 1. Tenta usar instrução de testes definida no plugin (MCP)
	plugin, _ := registry.FindExpertByLanguage(lang, o.Plugins)
	if plugin != nil && plugin.TestConfig != nil && plugin.TestConfig.Command != "" {
		logger.Info("📦 [RunSensor] Usando configuração de testes do Expert", "expert", plugin.ID)
		cmd = exec.Command("bash", "-c", plugin.TestConfig.Command)
	}

	if cmd == nil {
		switch lang {
		case "go":
			args := []string{"test", "-v"}
			if len(affectedPkgs) > 0 {
				for pkg := range affectedPkgs {
					args = append(args, "./"+pkg)
				}
			} else {
				args = append(args, "./...")
			}
			cmd = exec.Command("go", args...)
		case "flutter", "dart":
			exec.Command("flutter", "pub", "get").Run()
			// Flutter test não suporta múltiplos pacotes na mesma linha facilmente como o go test ./dir1 ./dir2
			// Por enquanto, se houver pacotes específicos, rodamos o comando base que o flutter lida bem com caminhos
			args := []string{"test"}
			if len(affectedPkgs) > 0 {
				for pkg := range affectedPkgs {
					args = append(args, pkg)
				}
			}
			cmd = exec.Command("flutter", args...)
		default:
			return false, "", fmt.Errorf("linguagem %s não suportada pelo RunSensor", lang)
		}
	}

	cmd.Dir = o.ProjectPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	logger.Info("📡 [RunSensor] Iniciando validação focada", "lang", lang, "pkgs", affectedPkgs)
	err := cmd.Run()
	output := out.String()

	if err != nil {
		logger.Warn("❌ [RunSensor] Testes falharam", "output_len", len(output))
		return false, output, nil
	}

	logger.Info("✅ [RunSensor] Testes do escopo passaram")
	return true, "Tests passed for affected packages! ✅\n" + output, nil
}

// UpdateTaskProgress atualiza o status, os logs de teste e o resultado da validação de uma tarefa no sprints.json
// ExecuteTaskHarness coordena o ciclo TDD automático para a próxima tarefa pendente
func (o *Orchestrator) ExecuteTaskHarness(ctx context.Context, config ProjectConfig) error {
	// 1. Carregar Roadmap para encontrar a tarefa atual
	sprintsPath := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")
	data, err := os.ReadFile(sprintsPath)
	if err != nil {
		return fmt.Errorf("roadmap não encontrado: %v", err)
	}

	var roadmap ProjectRoadmap
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return err
	}

	var sprintIdx, taskIdx int = -1, -1
	for i, s := range roadmap.Sprints {
		for j, t := range s.Tasks {
			if t.Status == "pending" || t.Status == "in_progress" {
				sprintIdx = i
				taskIdx = j
				break
			}
		}
		if sprintIdx != -1 {
			break
		}
	}

	if sprintIdx == -1 {
		return fmt.Errorf("nenhuma tarefa pendente encontrada no roadmap")
	}

	currentSprint := roadmap.Sprints[sprintIdx]
	currentTask := roadmap.Sprints[sprintIdx].Tasks[taskIdx]

	_, _, err = o.ExecuteTaskWithTDD(ctx, currentSprint.ID, currentTask.ID, config)
	return err
}

// ExecuteTaskWithTDD executa uma tarefa específica com loop de validação e auto-correção
func (o *Orchestrator) ExecuteTaskWithTDD(ctx context.Context, sprintID, taskID FlexibleID, config ProjectConfig) (string, bool, error) {
	// PROTEÇÃO: Recuperar de qualquer pânico inesperado e logar detalhes
	defer func() {
		if r := recover(); r != nil {
			logger.Error("🔥 [Harness] PÂNICO CRÍTICO detectado na execução da tarefa", fmt.Errorf("%v", r))
		}
	}()

	logger.Info("🚀 [Harness] Iniciando loop TDD para tarefa", "taskID", taskID)

	// 1. Carregar a tarefa para obter critérios de aceite
	var roadmap ProjectRoadmap
	foundRoadmap := false

	// Fallback 1: Memória da Configuração do Projeto
	if !foundRoadmap && config.Roadmap != nil {
		logger.Info("ℹ️ [Harness] sprints.json não encontrado, usando roadmap da memória (Config)")
		roadmapData, _ := json.Marshal(config.Roadmap)
		json.Unmarshal(roadmapData, &roadmap)
		foundRoadmap = true
	}

	sprintsPath := filepath.Join(o.ProjectPath, ".spec-wizard", "sprints.json")
	if data, err := os.ReadFile(sprintsPath); err == nil {
		if err := json.Unmarshal(data, &roadmap); err != nil {
			logger.Warn("⚠️ [Harness] Erro ao processar sprints.json", "error", err)
		} else {
			foundRoadmap = true
		}
	}

	// Fallback 2: Se o arquivo não existir, o dashboard manda a tarefa/sprint no corpo da requisição
	// Vamos tentar reconstruir o roadmap a partir disso se necessário
	if !foundRoadmap && config.CurrentSprint != nil && config.CurrentTask != nil {
		logger.Info("ℹ️ [Harness] sprints.json não encontrado, reconstruindo a partir da requisição UI")
		roadmap.Sprints = []Sprint{*config.CurrentSprint}
		foundRoadmap = true
	}

	if !foundRoadmap {
		logger.Warn("❌ [Harness] Impossível executar: Roadmap ausente em disco e na memória")
		return "", false, fmt.Errorf("roadmap não encontrado em %s. Por favor, gere o roadmap estratégico primeiro ou inicialize o projeto.", sprintsPath)
	}

	var task *Task
	var currentSprint *Sprint
	sID := string(sprintID)
	tID := string(taskID)

	logger.Info("🔍 [Harness] Iniciando busca da tarefa no roadmap", "sprintID", sID, "taskID", tID, "foundRoadmap", foundRoadmap, "sprintCount", len(roadmap.Sprints))

	for si := range roadmap.Sprints {
		if string(roadmap.Sprints[si].ID) == sID {
			currentSprint = &roadmap.Sprints[si]
			logger.Info("📂 [Harness] Sprint encontrado", "sprintIdx", si, "taskCount", len(currentSprint.Tasks))
			for ti := range currentSprint.Tasks {
				if string(currentSprint.Tasks[ti].ID) == tID {
					task = &currentSprint.Tasks[ti]
					logger.Info("🎯 [Harness] Tarefa encontrada!", "title", task.Title)
					break
				}
			}
		}
	}

	if task == nil {
		logger.Warn("❌ [Harness] Tarefa NÃO encontrada no roadmap", "taskID", tID, "sprintID", sID)
		return "", false, fmt.Errorf("tarefa %s não encontrada no sprint %s", tID, sID)
	}

	logger.Info("✅ [Harness] Preparando para iniciar execução", "taskTitle", task.Title)

	// 2. Marcar como in_progress inicialmente (Task Lock)
	o.UpdateTaskProgress(sprintID, taskID, "in_progress", "", "", nil, nil, nil)

	llmClient, err := o.GetLLMClient(config)
	if err != nil {
		return "", false, err
	}

	maxAttempts := 5
	var lastLogs string
	var lastResponse string

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("\n🔄 [TDD LOOP] Tentativa %d de %d para a tarefa: %s\n", attempt, maxAttempts, task.Title)
		logger.Info(fmt.Sprintf("🔄 [Harness] Tentativa %d de %d", attempt, maxAttempts))

		// Cada tentativa do TDD tem seu próprio timeout de 600s (fôlego renovado)
		// permitindo que o GoClaw execute seu loop interno de ferramentas com calma.
		attemptCtx, cancel := context.WithTimeout(ctx, 600*time.Second)

		// 2. Gerar Prompt (inclui logs de falha da tentativa anterior se houver)
		promptStr, err := o.GenerateHarnessPrompt(task, currentSprint, config, "")
		if err != nil {
			cancel()
			return "", false, err
		}

		// 3. Disparar execução da IA (O GoClaw gerencia o loop de ferramentas internamente)
		aiResponse, err := llmClient.AskWithContext(attemptCtx, promptStr, nil)
		cancel() // Libera o contexto assim que a chamada termina

		if err != nil {
			// Se o erro for timeout da tentativa, mas o global ainda estiver ok, logamos e tentamos de novo
			if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "deadline exceeded") {
				logger.Warn("⚠️ [Harness] Timeout atingido nesta tentativa (600s), mas tentaremos de novo se houver margem.", "attempt", attempt)
				continue
			}
			return "", false, fmt.Errorf("falha na execução da IA: %v", err)
		}
		lastResponse = aiResponse

		// 4. Aplicar mudanças no disco e capturar arquivos afetados
		// 4a. Via Markdown Parsing (padrão legado/fallback)
		touchedFiles, err := o.applyAIChanges(aiResponse)
		if err != nil {
			logger.Warn("⚠️ [Harness] Falha ao aplicar mudanças de markdown", "error", err)
		}

		// 4b. Via Ferramentas MCP (Rastreio Direto do Backend)
		// Isso captura arquivos criados por write_file, edit_file, etc.
		mcpFiles := o.Tools.GetTouchedFiles()
		if len(mcpFiles) > 0 {
			logger.Info("📂 [Harness] Arquivos detectados via ferramentas MCP", "count", len(mcpFiles))
			// Merge único
			fileMap := make(map[string]bool)
			for _, f := range touchedFiles {
				fileMap[f] = true
			}
			for _, f := range mcpFiles {
				fileMap[f] = true
			}

			touchedFiles = nil
			for f := range fileMap {
				touchedFiles = append(touchedFiles, f)
			}
		}

		// 5. Rodar Sensor de Validação Robusto (Custo Zero)
		plugin, _ := registry.FindExpertByLanguage(config.Language, o.Plugins)
		sensor := NewSensorManager(o.ProjectPath, config.Language, plugin)
		sensor.RunValidation(touchedFiles)

		success := !sensor.HasFailures()
		lastLogs = sensor.GetFailureReport()

		// 5b. Verificação de Inércia e Rigor (TDD Enforcement)
		if len(touchedFiles) == 0 {
			success = false
			lastLogs += "\n⚠️ [Harness] REJEITADO POR INÉRCIA: Você não realizou nenhuma alteração em arquivos.\n"
			logger.Warn("⚠️ [Harness] Nenhuma alteração detectada. Rejeitando execução.")
		}

		// Separar arquivos de implementação e testes
		var implementationFiles []string
		var testFiles []string
		for _, f := range touchedFiles {
			// Melhora a detecção de arquivos de teste para Go e Dart
			lowerF := strings.ToLower(f)
			if strings.Contains(lowerF, "_test.go") || strings.Contains(lowerF, "_test.dart") ||
				strings.Contains(lowerF, ".spec.") || strings.Contains(lowerF, "test/") {
				testFiles = append(testFiles, f)
			} else {
				implementationFiles = append(implementationFiles, f)
			}
		}

		testsFound := len(testFiles) > 0
		if success && !testsFound {
			success = false
			lastLogs += "\n⚠️ [Harness] REJEITADO POR FALTA DE RIGOR (TDD): Você esqueceu de criar os arquivos de teste unitário correspondentes. Todo código novo deve vir acompanhado de testes.\n"
			logger.Warn("⚠️ [Harness] Sucesso técnico mas rejeitado por testes ausentes.")
		}

		// 6. Persistência de logs e status parcial
		if success {
			logger.Info("✅ [Harness] Sucesso Total! Testes passaram e arquivos foram criados.")

			// Marcar todos os critérios como atendidos
			metCriteria := make([]AcceptanceCriterion, len(task.AcceptanceCriteria))
			for i, c := range task.AcceptanceCriteria {
				metCriteria[i] = c
				metCriteria[i].Status = "met"
			}

			o.UpdateTaskProgress(sprintID, taskID, "completed", "All tests passed and files linked automatically! ✅\n"+lastLogs, "success", metCriteria, implementationFiles, testFiles)
			return aiResponse, true, nil
		}

		// Se chegamos aqui, falhou. Atualizamos com o erro para o próximo ciclo de TDD ver
		o.UpdateTaskProgress(sprintID, taskID, "failure", lastLogs, "error", nil, implementationFiles, testFiles)

		// Se o sensor passou mas falta teste, forçamos falha para o retry
		if success && !testsFound {
			lastLogs += "\n⚠️ [Harness] REJEITADO POR FALTA DE RIGOR: Você esqueceu de criar os arquivos de teste unitário.\n"
			logger.Warn("⚠️ [Harness] Sucesso técnico mas rejeitado por testes ausentes.")
		} else {
			logger.Warn("❌ [Harness] Validação técnica falhou.")
		}

		o.UpdateTaskProgress(sprintID, taskID, "in_progress", lastLogs, "failure", nil, implementationFiles, testFiles)
		
		// CRÍTICO: Atualiza as referências locais para a próxima tentativa do prompt
		task.TestLogs = lastLogs
		task.TestStatus = "failure"
		// Se houve arquivos criados, eles se tornam referência para a correção
		if len(implementationFiles) > 0 {
			task.Files = uniqueStrings(append(task.Files, implementationFiles...))
		}
		if len(testFiles) > 0 {
			task.TestFiles = uniqueStrings(append(task.TestFiles, testFiles...))
		}

		logger.Warn("🔁 [Harness] Preparando próxima tentativa com feedback de erro", "error_len", len(lastLogs))
		
		// Pequena pausa entre tentativas para evitar rate limits se for muito rápido
		time.Sleep(2 * time.Second)
	}

	o.UpdateTaskProgress(sprintID, taskID, "failed", lastLogs, "failure", nil, nil, nil)
	return lastResponse, false, fmt.Errorf("tarefa não validada após %d tentativas. Verifique os logs no dashboard", maxAttempts)
}

// applyAIChanges extrai blocos de código markdown e salva no filesystem (Retorna lista de arquivos alterados)
func (o *Orchestrator) applyAIChanges(aiResponse string) ([]string, error) {
	lines := strings.Split(aiResponse, "\n")
	var currentFile string
	var currentCode strings.Builder
	inCodeBlock := false
	touchedFiles := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(strings.ToUpper(trimmed), "FILE:") || strings.Contains(strings.ToUpper(trimmed), "PATH:") {
			parts := strings.Split(trimmed, ":")
			if len(parts) > 1 {
				currentFile = strings.TrimSpace(parts[1])
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
					fullPath := filepath.Join(o.ProjectPath, currentFile)
					os.MkdirAll(filepath.Dir(fullPath), 0755)
					os.WriteFile(fullPath, []byte(currentCode.String()), 0644)
					logger.Debug("💾 [Harness] Arquivo salvo via markdown parsing", "file", currentFile)
					touchedFiles[currentFile] = true
					currentFile = ""
				}
			}
			continue
		}

		if inCodeBlock {
			currentCode.WriteString(line + "\n")
		}
	}

	var result []string
	for f := range touchedFiles {
		result = append(result, f)
	}
	return result, nil
}

func (o *Orchestrator) captureDiff() string {
	if _, err := os.Stat(filepath.Join(o.ProjectPath, ".git")); os.IsNotExist(err) {
		return ""
	}
	c := exec.Command("git", "diff")
	c.Dir = o.ProjectPath
	diffOut, _ := c.Output()
	return string(diffOut)
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
