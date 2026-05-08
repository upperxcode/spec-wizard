package adatools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// RegisterShellTools registra ferramentas de execução de comandos
func RegisterShellTools(r *Registry) {
	r.Register("shell_execute", "Executa um comando no terminal dentro do diretório do projeto. Ideal para builds, testes e git.", `{
		"type": "object",
		"properties": {
			"command": { "type": "string", "description": "O comando shell completo para execução." }
		},
		"required": ["command"]
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		command, _ := args["command"].(string)
		if command == "" {
			return "", fmt.Errorf("comando vazio")
		}

		// 1. Validação de Segurança (Blocklist básica)
		dangerous := []string{"sudo ", "rm -", "mkfs", "dd ", "shutdown", "reboot"}
		cmdLower := strings.ToLower(command)
		for _, bad := range dangerous {
			if strings.Contains(cmdLower, bad) {
				return "", fmt.Errorf("comando bloqueado por segurança: uso de '%s' não permitido", strings.TrimSpace(bad))
			}
		}

		// 2. Timeout de Segurança (evita travamentos com servidores ou processos infinitos)
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(timeoutCtx, "cmd", "/C", command)
		} else {
			cmd = exec.CommandContext(timeoutCtx, "bash", "-c", command)
		}

		// Define o diretório de trabalho como a raiz do workspace
		cmd.Dir = r.Root()

		output, err := cmd.CombinedOutput()

		// 3. Limite de Saída (Evita estouro de janela de contexto do LLM)
		outStr := string(output)
		if len(outStr) > 3000 {
			outStr = outStr[:3000] + "\n... [Saída truncada por limite de segurança do contexto]"
		}

		if err != nil {
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return outStr, fmt.Errorf("tempo limite de 30s excedido. Comando foi interrompido.\nSaída parcial: %s", outStr)
			}
			return outStr, fmt.Errorf("erro na execução: %v\nSaída: %s", err, outStr)
		}

		return outStr, nil
	})
}
