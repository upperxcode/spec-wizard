package adatools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/utils"
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

		r.RecordTouch(path)
		return fmt.Sprintf("Arquivo %s gravado com sucesso (%d bytes).", path, len(content)), nil
	})

	// 4. editar_arquivo (SEARCH/REPLACE)
	r.Register("edit_file", "Realiza edições precisas num ficheiro local substituindo um bloco de código exato por outro. O bloco 'search' deve ser ÚNICO no ficheiro e respeitar a indentação exata. Use isto para pequenas alterações em vez de write_file.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "O caminho relativo do ficheiro." },
			"search": { "type": "string", "description": "O bloco de código exato a ser localizado (incluindo espaços e quebras de linha)." },
			"replace": { "type": "string", "description": "O novo código que irá substituir o bloco localizado." }
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

		if err := utils.EditFile(fullPath, search, replace); err != nil {
			// O erro retornado pelo motor já é pedagógico e amigável para a IA
			return "", err
		}

		r.RecordTouch(path)
		return fmt.Sprintf("Ficheiro %s atualizado com sucesso.", path), nil
	})

	// 5. pesquisar_codigo
	r.Register("search_code", "Pesquisa por uma string exata ou regex num ficheiro e retorna as linhas correspondentes com o contexto em redor. Use esta ferramenta em vez de read_file para ficheiros grandes.", `{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "O caminho relativo do ficheiro." },
			"query": { "type": "string", "description": "O texto ou expressão regular a procurar." },
			"context_lines": { "type": "integer", "description": "Número de linhas antes e depois do match para incluir como contexto. (Padrão: 5)", "default": 5 }
		},
		"required": ["path", "query"]
	}`, func(ctx context.Context, args map[string]any) (string, error) {
		path, _ := args["path"].(string)
		query, _ := args["query"].(string)

		contextLines := 5
		if cl, ok := args["context_lines"].(float64); ok {
			contextLines = int(cl)
		}

		fullPath, err := r.validatePath(path)
		if err != nil {
			return "", err
		}

		result, err := utils.SearchCode(fullPath, query, contextLines)
		if err != nil {
			return "", fmt.Errorf("erro na pesquisa: %v", err)
		}

		return result, nil
	})
}
