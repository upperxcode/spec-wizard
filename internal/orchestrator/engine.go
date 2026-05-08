package orchestrator

import (
	"fmt"
	"os"
	"path/filepath" // Essencial para manipulação de caminhos
)

// StartDiscovery realiza o escaneamento e dispara a inteligência se encontrar um gatilho
func (o *Orchestrator) StartDiscovery() {
	fmt.Printf("🔍 Analisando o caminho: %s\n", o.ProjectPath)

	files, err := os.ReadDir(o.ProjectPath)
	if err != nil {
		fmt.Printf("❌ Erro ao ler diretório: %v\n", err)
		return
	}

	for _, file := range files {
		for _, plugin := range o.Plugins {
			for _, trigger := range plugin.Triggers {
				if file.Name() == trigger {
					fmt.Printf("🎯 Plugin '%s' reconheceu o projeto via: %s\n", plugin.ID, trigger)

					// 1. Cria a infraestrutura básica (.spec-wizard/project_config.json) [cite: 250, 431]
					o.InitializeProject(plugin.Language, plugin)

					// 2. Lê o conteúdo do arquivo de assinatura (ex: pubspec.yaml) [cite: 368, 612]
					targetFilePath := filepath.Join(o.ProjectPath, file.Name())
					rawContent, err := os.ReadFile(targetFilePath)
					if err != nil {
						fmt.Printf("❌ Erro ao ler arquivo de gatilho: %v\n", err)
						return
					}

					// 3. Dispara a síntese de inteligência (Expert + LM Studio) [cite: 199, 599, 613]
					o.OrchestrateIntelligence(plugin, string(rawContent))

					return
				}
			}
		}
	}
	fmt.Println("⚠️ Nenhum arquivo de gatilho (ex: pubspec.yaml) foi encontrado.")
}
