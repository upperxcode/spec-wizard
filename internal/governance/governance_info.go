package governance

import (
	"fmt"
	"strings"
)

// GetTaskInfoInternal retorna um relatório Markdown detalhado da tarefa
func GetTaskInfoInternal(projectPath string, idOrDisplayID string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("banco não inicializado")
	}

	var t struct {
		ID          int
		DisplayID   int
		Title       string
		Description string
		Status      string
		TestLogs    string
		TestStatus  string
	}

	// Tenta buscar por display_id primeiro, depois por ID interno no contexto do projeto atual
	query := `
		SELECT t.id, t.display_id, t.title, t.description, t.status, t.test_logs, t.test_status 
		FROM tasks t
		JOIN sprints s ON t.sprint_id = s.id
		JOIN projects p ON s.project_id = p.id
		WHERE p.path = ? AND (t.display_id = ? OR t.id = ?)
	`
	err := db.QueryRow(query, projectPath, idOrDisplayID, idOrDisplayID).Scan(&t.ID, &t.DisplayID, &t.Title, &t.Description, &t.Status, &t.TestLogs, &t.TestStatus)
	if err != nil {
		return "", fmt.Errorf("tarefa não encontrada para este projeto: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 📋 Tarefa %d: %s\n\n", t.DisplayID, t.Title))
	sb.WriteString(fmt.Sprintf("**ID Interno:** `%d` | **Status:** `%s`\n\n", t.ID, t.Status))
	sb.WriteString(fmt.Sprintf("## 📝 Descrição\n%s\n\n", t.Description))

	// Busca Critérios
	sb.WriteString("## ✅ Critérios de Aceite\n")
	rows, _ := db.Query("SELECT description, status FROM acceptance_criteria WHERE task_id = ?", t.ID)
	for rows != nil && rows.Next() {
		var desc, status string
		rows.Scan(&desc, &status)
		icon := "⚪"
		if status == "completed" || status == "Completed" {
			icon = "✅"
		}
		sb.WriteString(fmt.Sprintf("- %s %s\n", icon, desc))
	}
	if rows != nil {
		rows.Close()
	}

	// Busca Arquivos
	sb.WriteString("\n## 📂 Arquivos Vinculados\n")
	rows, _ = db.Query(`SELECT f.path, tf.file_type FROM files f 
		JOIN task_files tf ON f.id = tf.file_id WHERE tf.task_id = ?`, t.ID)
	for rows != nil && rows.Next() {
		var path, fType string
		rows.Scan(&path, &fType)
		sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", path, fType))
	}
	if rows != nil {
		rows.Close()
	}

	// Logs de Teste
	if t.TestLogs != "" {
		sb.WriteString("\n## 🧪 Logs de Teste\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", t.TestLogs))
	}

	return sb.String(), nil
}
