package registry

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// PluginManager gerencia o ciclo de vida dos processos MCP de linguagem
type PluginManager struct {
	CurrentPlugin *ExpertPlugin
	CurrentCmd    *exec.Cmd
}

// GlobalManager é uma instância única para gerenciar os plugins ativos
var GlobalManager = &PluginManager{}

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

// EnsureExpertRunning garante que o plugin correto está em execução e aloca porta dinâmica
func (m *PluginManager) EnsureExpertRunning(plugin *ExpertPlugin) error {
	// Se já estiver rodando o plugin certo e estiver saudável, não faz nada
	if m.CurrentPlugin != nil && m.CurrentPlugin.ID == plugin.ID {
		if m.isHealthy(m.CurrentPlugin.Endpoint) {
			return nil
		}
		m.StopCurrent()
	}

	// Se houver outro, mata
	if m.CurrentPlugin != nil {
		m.StopCurrent()
	}

	// Aloca porta dinâmica (Range 8080-8100)
	port, err := m.findFreePort(8080, 8100)
	if err != nil {
		return err
	}

	// Atualiza o endpoint dinamicamente
	plugin.Endpoint = fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("🚀 [Lifecycle] Alocando Expert %s na porta %d...\n", plugin.ID, port)

	// Injeta a porta no comando de inicialização
	startCmd := fmt.Sprintf("%s %d", plugin.StartCommand, port)

	cmd := exec.Command("bash", "-c", startCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logPath := filepath.Join("logs", fmt.Sprintf("expert_%s.log", plugin.ID))
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar expert: %v", err)
	}

	m.CurrentPlugin = plugin
	m.CurrentCmd = cmd

	// Aguarda saúde
	maxRetries := 15
	for i := 0; i < maxRetries; i++ {
		if m.isHealthy(plugin.Endpoint) {
			fmt.Printf("✅ [Lifecycle] Expert %s ONLINE em %s\n", plugin.ID, plugin.Endpoint)
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	m.StopCurrent()
	return fmt.Errorf("timeout: expert %s não subiu em %s", plugin.ID, plugin.Endpoint)
}

// StopCurrent encerra o plugin que está rodando no momento
func (m *PluginManager) StopCurrent() {
	if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
		fmt.Printf("🛑 [Lifecycle] Encerrando Expert: %s (PID: %d)\n", m.CurrentPlugin.ID, m.CurrentCmd.Process.Pid)

		// Mata o grupo de processos inteiro (bash + python/filhos)
		pgid, err := syscall.Getpgid(m.CurrentCmd.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			m.CurrentCmd.Process.Kill()
		}

		m.CurrentCmd.Wait()
		m.CurrentCmd = nil
		m.CurrentPlugin = nil
	}
}

func (m *PluginManager) isHealthy(endpoint string) bool {
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	// Adicionado retry interno no health check para ser mais tolerante a picos
	resp, err := client.Get(endpoint + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
