package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// SearchCode realiza uma busca cirúrgica em um arquivo, retornando blocos de contexto.
// Suporta tanto Regex quanto busca literal.
func SearchCode(path, query string, contextLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("ficheiro não encontrado: %s", path)
	}
	defer file.Close()

	if contextLines < 0 {
		contextLines = 5
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("erro ao ler ficheiro: %v", err)
	}

	// Tenta compilar como Regex, se falhar usa busca literal de substring
	re, err := regexp.Compile(query)
	isRegex := err == nil

	matchedIndices := make(map[int]bool)
	for i, line := range lines {
		if isRegex {
			if re.MatchString(line) {
				matchedIndices[i] = true
			}
		} else {
			if strings.Contains(line, query) {
				matchedIndices[i] = true
			}
		}
	}

	if len(matchedIndices) == 0 {
		return "Nenhuma correspondência encontrada para: " + query, nil
	}

	// Coleta todas as linhas que devem ser exibidas (match + contexto)
	displayIndicesMap := make(map[int]bool)
	for idx := range matchedIndices {
		start := idx - contextLines
		if start < 0 {
			start = 0
		}
		end := idx + contextLines
		if end >= len(lines) {
			end = len(lines) - 1
		}

		for i := start; i <= end; i++ {
			displayIndicesMap[i] = true
		}
	}

	// Ordena os índices para exibição sequencial
	var displayIndices []int
	for idx := range displayIndicesMap {
		displayIndices = append(displayIndices, idx)
	}
	sort.Ints(displayIndices)

	var sb strings.Builder
	lastIdx := -2
	for _, idx := range displayIndices {
		// Adiciona separador visual se houver salto entre blocos de contexto
		if lastIdx != -2 && idx > lastIdx+1 {
			sb.WriteString("\n---\n")
		}

		prefix := "  "
		if matchedIndices[idx] {
			prefix = "> " // Marca a linha que deu match
		}

		sb.WriteString(fmt.Sprintf("%d:%s%s\n", idx+1, prefix, lines[idx]))
		lastIdx = idx
	}

	return sb.String(), nil
}
