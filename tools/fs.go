package adatools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath garante que o caminho solicitado está dentro da raiz do workspace
func (r *Registry) validatePath(path string) (string, error) {
	root := r.Root()
	if root == "" {
		return "", fmt.Errorf("raiz do workspace não configurada")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("falha ao resolver raiz: %v", err)
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Join(absRoot, path)
	}

	// Adiciona separador para evitar bypass de path (ex: "/meu_app_secreto" passando no teste de "/meu_app")
	rootWithSep := absRoot
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}

	if absPath != absRoot && !strings.HasPrefix(absPath, rootWithSep) {
		return "", fmt.Errorf("acesso negado: caminho fora do workspace")
	}

	return absPath, nil
}

// RegisterFSTools registra as ferramentas de arquivos no registro fornecido
func RegisterFSTools(r *Registry) {
	// 1. ler_arquivo
	r.Register("read_file", "Lê o conteúdo de um arquivo específico dentro do Workspace.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "O caminho relativo do arquivo para leitura." }
		},
		"required": ["path"]
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		path, _ := args["path"].(string)
		fullPath, err := r.validatePath(path)
		if err != nil {
			return "", err
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("erro ao ler arquivo: %v", err)
		}

		return string(data), nil
	})

	// 2. listar_diretorio
	r.Register("list_dir", "Lista os arquivos e pastas de um diretório.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Caminho do diretório a ser listado.", "default": "." }
		}
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		path := "."
		if p, ok := args["path"].(string); ok {
			path = p
		}

		fullPath, err := r.validatePath(path)
		if err != nil {
			return "", err
		}

		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return "", fmt.Errorf("erro ao listar diretório: %v", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Conteúdo de %s:\n", path))
		for _, entry := range entries {
			typeStr := "[FILE]"
			if entry.IsDir() {
				typeStr = "[DIR ]"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", typeStr, entry.Name()))
		}

		return sb.String(), nil
	})

	// 3. escrever_arquivo
	r.Register("write_file", "Cria ou sobrescreve um arquivo com o conteúdo fornecido.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Caminho do arquivo." },
			"content": { "type": "string", "description": "Conteúdo a ser gravado." }
		},
		"required": ["path", "content"]
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)

		fullPath, err := r.validatePath(path)
		if err != nil {
			return "", err
		}

		// Garante que o diretório pai existe
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return "", fmt.Errorf("erro ao criar diretórios: %v", err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("erro ao gravar arquivo: %v", err)
		}

		return fmt.Sprintf("Arquivo %s gravado com sucesso (%d bytes).", path, len(content)), nil
	})

	// 4. editar_arquivo (SEARCH/REPLACE)
	r.Register("edit_file", "Realiza edições precisas em um arquivo local usando blocos de substituição.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Caminho relativo do arquivo." },
			"search": { "type": "string", "description": "O conteúdo exato a ser localizado no arquivo." },
			"replace": { "type": "string", "description": "O novo conteúdo para substituir o trecho localizado." }
		},
		"required": ["path", "search", "replace"]
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		path, _ := args["path"].(string)
		search, _ := args["search"].(string)
		replace, _ := args["replace"].(string)

		fullPath, err := r.validatePath(path)
		if err != nil {
			return "", err
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("erro ao ler arquivo para edição: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, search) {
			return "", fmt.Errorf("trecho de busca não encontrado no arquivo %s. Certifique-se de que o trecho 'search' seja idêntico ao conteúdo do arquivo, incluindo espaços e quebras de linha", path)
		}

		// Conta ocorrências para evitar edições ambíguas se necessário (opcional)
		count := strings.Count(content, search)
		if count > 1 {
			return "", fmt.Errorf("o trecho de busca foi encontrado %d vezes. Seja mais específico para evitar edições incorretas", count)
		}

		newContent := strings.Replace(content, search, replace, 1)
		if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
			return "", fmt.Errorf("erro ao salvar arquivo editado: %v", err)
		}

		return fmt.Sprintf("Arquivo %s editado com sucesso. Substituído 1 bloco.", path), nil
	})
}
