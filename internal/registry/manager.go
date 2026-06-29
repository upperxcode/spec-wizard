package registry

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"spec-wizard/internal/logger"
	"sync"
	"syscall"
	"time"
)

// RunningExpert encapsula um expert em execução
type RunningExpert struct {
	Plugin *ExpertPlugin
	Cmd    *exec.Cmd
}

// PluginManager gerencia o ciclo de vida dos processos MCP de linguagem e Experts de IA
type PluginManager struct {
	Running      map[string]*RunningExpert
	ResourcesDir string
	mu           sync.RWMutex
}

// GlobalManager é uma instância única para gerenciar os plugins ativos
var GlobalManager = &PluginManager{
	Running: make(map[string]*RunningExpert),
}

// FindFreePort busca uma porta disponível em um range
func (m *PluginManager) findFreePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			continue
		}
		l, err := net.ListenTCP("tcp", addr)
		if err == nil {
			l.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("nenhuma porta livre no range %d-%d", start, end)
}

// EnsureExpertRunning garante que o plugin específico está em execução
func (m *PluginManager) EnsureExpertRunning(plugin *ExpertPlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Verifica se já está rodando e saudável
	if running, ok := m.Running[plugin.ID]; ok {
		if m.isHealthy(plugin.ID, running.Plugin.Endpoint) {
			return nil
		}
		// Se não estiver saudável, remove da lista e tenta reiniciar
		m.stopExpertLocked(plugin.ID)
	}

	// 2. Aloca porta dinâmica (Range 8080-8200)
	port, err := m.findFreePort(8080, 8200)
	if err != nil {
		return err
	}

	// 3. Configura o endpoint dinamicamente
	plugin.Endpoint = fmt.Sprintf("http://localhost:%d", port)
	logger.Info("🚀 [Lifecycle] Alocando Expert", "id", plugin.ID, "port", port)

	// 4. Inicia o processo
	startCmd := fmt.Sprintf("%s %d", plugin.StartCommand, port)
	cmd := exec.Command("bash", "-c", startCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if m.ResourcesDir != "" {
		cmd.Dir = m.ResourcesDir
	}

	logPath := filepath.Join(m.ResourcesDir, "logs", fmt.Sprintf("expert_%s.log", plugin.ID))
	if m.ResourcesDir == "" {
		logPath = filepath.Join("logs", fmt.Sprintf("expert_%s.log", plugin.ID))
		os.MkdirAll("logs", 0755)
	}
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar expert %s: %v", plugin.ID, err)
	}

	m.Running[plugin.ID] = &RunningExpert{
		Plugin: plugin,
		Cmd:    cmd,
	}

	// 5. Aguarda saúde
	logger.Info("⏳ [Lifecycle] Aguardando expert subir", "id", plugin.ID, "endpoint", plugin.Endpoint)
	for i := 0; i < 30; i++ {
		resp, err := http.Get(plugin.Endpoint + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			logger.Info("✅ [Lifecycle] Expert ONLINE", "id", plugin.ID, "endpoint", plugin.Endpoint)
			resp.Body.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	m.stopExpertLocked(plugin.ID)
	return fmt.Errorf("timeout: expert %s não respondeu ao health check em %s após 30 tentativas", plugin.ID, plugin.Endpoint)
}

// StopExpert encerra um expert específico
func (m *PluginManager) StopExpert(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopExpertLocked(id)
}

func (m *PluginManager) stopExpertLocked(id string) {
	if running, ok := m.Running[id]; ok {
		if running.Cmd != nil && running.Cmd.Process != nil {
			logger.Info("🛑 [Lifecycle] Encerrando Expert", "id", id, "pid", running.Cmd.Process.Pid)
			running.Cmd.Process.Signal(os.Interrupt)
			pgid, err := syscall.Getpgid(running.Cmd.Process.Pid)
			if err == nil {
				syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				running.Cmd.Process.Kill()
			}
			running.Cmd.Wait()
		}
		delete(m.Running, id)
	}
}

// StopAll encerra todos os plugins ativos
func (m *PluginManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id := range m.Running {
		m.stopExpertLocked(id)
	}
}

// StopCurrent mantido para compatibilidade se necessário, mas agora redireciona
func (m *PluginManager) StopCurrent() {
	m.StopAll()
}

// IsHealthy verifica se o expert está saudável
func (m *PluginManager) IsHealthy(id string) bool {
	m.mu.RLock()
	running, ok := m.Running[id]
	endpoint := ""
	if ok && running.Plugin != nil {
		endpoint = running.Plugin.Endpoint
	}
	m.mu.RUnlock()

	if endpoint == "" {
		return false
	}

	client := http.Client{
		Timeout: 800 * time.Millisecond,
	}
	url := endpoint + "/health"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsRunning verifica se o processo do expert existe
func (m *PluginManager) IsRunning(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	running, ok := m.Running[id]
	return ok && running.Cmd != nil && running.Cmd.Process != nil
}

func (m *PluginManager) isHealthy(id, endpoint string) bool {
	client := http.Client{
		Timeout: 800 * time.Millisecond,
	}
	url := endpoint + "/health"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
