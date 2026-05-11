package utils

import (
	"fmt"
	"os"
	"strings"
)

// EditFile realiza uma substituição cirúrgica em um arquivo.
// Retorna erro se o bloco 'search' não for encontrado ou se for ambíguo (múltiplas ocorrências).
func EditFile(path, search, replace string) error {
	// 1. Lê o conteúdo original e as permissões
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("erro ao acessar ficheiro: %v", err)
	}
	perm := info.Mode()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("erro ao ler ficheiro: %v", err)
	}

	content := string(data)

	// 2. Validação de Ocorrências
	count := strings.Count(content, search)

	if count == 0 {
		return fmt.Errorf("bloco de pesquisa não encontrado no ficheiro '%s'.\n\nERRO DE PRECISÃO: O conteúdo de 'search' deve ser EXATAMENTE igual ao que está no ficheiro, incluindo espaços, tabs e quebras de linha. Dica: Use a ferramenta 'search_code' para copiar o bloco exato antes de tentar editar.", path)
	}

	if count > 1 {
		return fmt.Errorf("múltiplas ocorrências (%d) do bloco de pesquisa encontradas no ficheiro '%s'.\n\nERRO DE AMBIGUIDADE: Para evitar edições acidentais, o bloco 'search' deve ser ÚNICO. Dica: Adicione mais linhas de contexto (linhas acima ou abaixo) ao seu parâmetro 'search' para torná-lo exclusivo.", count, path)
	}

	// 3. Substituição Segura
	newContent := strings.Replace(content, search, replace, 1)

	// 4. Escrita Atômica (usando arquivo temporário para evitar truncamento em caso de crash)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), perm); err != nil {
		return fmt.Errorf("erro ao criar ficheiro temporário: %v", err)
	}

	// Move o temporário para o destino original (operação atômica no Linux)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Limpa se falhar
		return fmt.Errorf("erro ao finalizar escrita do ficheiro: %v", err)
	}

	return nil
}
