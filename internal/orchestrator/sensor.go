package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"spec-wizard/internal/registry"
	"strings"
)

// SensorResult representa o resultado da validação de um sensor
type SensorResult struct {
	SensorName string `json:"sensor_name"`
	Status     string `json:"status"` // "pass", "fail", "warning"
	Message    string `json:"message"`
	ErrorLog   string `json:"error_log,omitempty"`
	RawOutput  string `json:"raw_output,omitempty"`
}

// SensorManager valida o código gerado pela IA
type SensorManager struct {
	ProjectPath   string
	Language      string
	SensorResults []SensorResult
	Plugin        *registry.ExpertPlugin
}

func NewSensorManager(projectPath, language string, plugin *registry.ExpertPlugin) *SensorManager {
	return &SensorManager{
		ProjectPath:   projectPath,
		Language:      language,
		SensorResults: make([]SensorResult, 0),
		Plugin:        plugin,
	}
}

// RunValidation executa os sensores apropriados para a linguagem
func (sm *SensorManager) RunValidation(touchedFiles []string) error {
	fmt.Printf("🔍 Executando sensores de validação para %s (Focado em %d arquivos)...\n", sm.Language, len(touchedFiles))

	// Identifica pacotes afetados
	affectedPkgs := make(map[string]bool)
	for _, f := range touchedFiles {
		dir := filepath.Dir(f)
		if dir != "." && dir != "" {
			affectedPkgs[dir] = true
		}
	}

	// Pre-validação: Sanidade de dependências para Go
	if strings.ToLower(sm.Language) == "go" {
		sm.runSensor("go_mod_tidy", "go", []string{"mod", "tidy"})
	}

	// 1. Tenta usar instrução de testes definida no plugin (MCP)
	if sm.Plugin != nil && sm.Plugin.TestConfig != nil && sm.Plugin.TestConfig.Command != "" {
		fmt.Printf("📦 Usando configuração de testes do Expert %s\n", sm.Plugin.ID)
		sm.runSensor("expert_test_command", "bash", []string{"-c", sm.Plugin.TestConfig.Command})
		return nil
	}

	// 2. Fallback para sensores nativos/hardcoded
	switch strings.ToLower(sm.Language) {
	case "flutter":
		sm.runFlutterSensors(affectedPkgs)
	case "typescript", "node":
		sm.runTypeScriptSensors()
	case "go":
		sm.runGoSensors(affectedPkgs)
	default:
		return fmt.Errorf("linguagem não suportada: %s", sm.Language)
	}

	return nil
}

// runFlutterSensors executa validações específicas para Flutter
func (sm *SensorManager) runFlutterSensors(affectedPkgs map[string]bool) {
	// 1. flutter analyze - Análise estática
	sm.runSensor("flutter_analyze", "flutter", []string{"analyze"})

	// 2. flutter test - Testes unitários focados ou globais
	testArgs := []string{"test"}
	if len(affectedPkgs) > 0 {
		for pkg := range affectedPkgs {
			testArgs = append(testArgs, pkg)
		}
	}
	sm.runSensor("flutter_test", "flutter", testArgs)

	// 3. flutter pub get - Validação de dependências
	sm.runSensor("flutter_pub_get", "flutter", []string{"pub", "get"})

	// 4. dart format check - Validação de formatação
	sm.runSensor("dart_format", "dart", []string{"format", "--set-exit-if-changed", "."})
}

// runTypeScriptSensors executa validações específicas para TypeScript
func (sm *SensorManager) runTypeScriptSensors() {
	// 1. npm/yarn lint
	sm.runSensor("eslint", "npm", []string{"run", "lint"})

	// 2. npm/yarn test
	sm.runSensor("unit_tests", "npm", []string{"run", "test"})

	// 3. TypeScript compilation
	sm.runSensor("tsc_check", "npx", []string{"tsc", "--noEmit"})
}

// runGoSensors executa validações específicas para Go
func (sm *SensorManager) runGoSensors(affectedPkgs map[string]bool) {
	// 0. go mod tidy
	sm.runSensor("go_mod_tidy", "go", []string{"mod", "tidy"})

	// 1. go fmt
	sm.runSensor("go_fmt", "go", []string{"fmt", "./..."})

	// 2. go vet
	sm.runSensor("go_vet", "go", []string{"vet", "./..."})

	// 3. go test - Testes focados ou globais
	testArgs := []string{"test", "-v"}
	if len(affectedPkgs) > 0 {
		for pkg := range affectedPkgs {
			testArgs = append(testArgs, "./"+pkg)
		}
	} else {
		testArgs = append(testArgs, "./...")
	}
	sm.runSensor("go_test", "go", testArgs)

	// 4. go build
	sm.runSensor("go_build", "go", []string{"build", "./..."})
}

// runSensor executa um sensor específico
func (sm *SensorManager) runSensor(sensorName, command string, args []string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = sm.ProjectPath

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	result := SensorResult{
		SensorName: sensorName,
		RawOutput:  outputStr,
	}

	if err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Sensor falhou: %v", err)
		result.ErrorLog = outputStr
		fmt.Printf("❌ %s FALHOU\n", sensorName)
	} else {
		result.Status = "pass"
		result.Message = fmt.Sprintf("Validação passou para %s", sensorName)
		fmt.Printf("✅ %s PASSOU\n", sensorName)
	}

	sm.SensorResults = append(sm.SensorResults, result)
}

// HasFailures retorna true se algum sensor falhou
func (sm *SensorManager) HasFailures() bool {
	for _, result := range sm.SensorResults {
		if result.Status == "fail" {
			return true
		}
	}
	return false
}

// GetFailureReport retorna relatório detalhado das falhas
func (sm *SensorManager) GetFailureReport() string {
	hasFailures := false
	for _, r := range sm.SensorResults {
		if r.Status == "fail" {
			hasFailures = true
			break
		}
	}
	if !hasFailures {
		return ""
	}

	var report strings.Builder
	report.WriteString("# 🚨 Relatório de Falhas de Validação\n\n")

	for _, result := range sm.SensorResults {
		if result.Status == "fail" {
			report.WriteString(fmt.Sprintf("## ❌ %s\n", result.SensorName))
			report.WriteString(fmt.Sprintf("**Erro:** %s\n\n", result.Message))
			report.WriteString("```\n")
			report.WriteString(result.ErrorLog)
			report.WriteString("\n```\n\n")
		}
	}

	return report.String()
}

// SaveSensorLog persiste os resultados dos sensores para auditoria
func (sm *SensorManager) SaveSensorLog(projectPath string) error {
	logsDir := filepath.Join(projectPath, ".spec-wizard", "sensor-logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	var logContent strings.Builder
	logContent.WriteString("# Sensor Validation Report\n\n")

	for _, result := range sm.SensorResults {
		logContent.WriteString(fmt.Sprintf("## %s: %s\n", result.SensorName, result.Status))
		logContent.WriteString(fmt.Sprintf("%s\n\n", result.Message))

		if result.ErrorLog != "" {
			logContent.WriteString("### Error Output\n```\n")
			logContent.WriteString(result.ErrorLog)
			logContent.WriteString("\n```\n\n")
		}
	}

	logFile := filepath.Join(logsDir, "latest-validation.md")
	return os.WriteFile(logFile, []byte(logContent.String()), 0644)
}
