package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Snapshot struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	SprintID  string    `json:"sprint_id"`
	TaskID    string    `json:"task_id"`
	Data      string    `json:"data"` // JSON bruto do roadmap
	Timestamp time.Time `json:"timestamp"`
}

type SnapshotManager struct {
	db *sql.DB
}

func NewSnapshotManager(dbPath string) (*SnapshotManager, error) {
	// Habilitar cache compartilhado e modo de concorrência
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}

	// Configurações de pool para SQLite
	db.SetMaxOpenConns(1)

	// Criar tabela se não existir
	query := `
	CREATE TABLE IF NOT EXISTS snapshots (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		sprint_id TEXT,
		task_id TEXT,
		data TEXT,
		timestamp DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_project_task ON snapshots(project_id, task_id);
	`
	if _, err := db.Exec(query); err != nil {
		return nil, err
	}

	return &SnapshotManager{db: db}, nil
}

// SaveSnapshot salva uma nova iteração e mantém apenas as 10 últimas
func (m *SnapshotManager) SaveSnapshot(projectID, sprintID, taskID string, data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	id := uuid.New().String()
	timestamp := time.Now()

	_, err = m.db.Exec(
		"INSERT INTO snapshots (id, project_id, sprint_id, task_id, data, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		id, projectID, sprintID, taskID, string(jsonData), timestamp,
	)
	if err != nil {
		return "", err
	}

	// Limpeza: Mantém apenas as 10 últimas versões para esta tarefa
	cleanupQuery := `
	DELETE FROM snapshots 
	WHERE id NOT IN (
		SELECT id FROM snapshots 
		WHERE project_id = ? AND task_id = ?
		ORDER BY timestamp DESC 
		LIMIT 10
	) AND project_id = ? AND task_id = ?
	`
	m.db.Exec(cleanupQuery, projectID, taskID, projectID, taskID)

	return id, nil
}

// GetLatestSnapshot recupera a versão mais recente
func (m *SnapshotManager) GetLatestSnapshot(projectID, taskID string) (string, error) {
	var data string
	err := m.db.QueryRow(
		"SELECT data FROM snapshots WHERE project_id = ? AND task_id = ? ORDER BY timestamp DESC LIMIT 10",
		projectID, taskID,
	).Scan(&data)

	if err == sql.ErrNoRows {
		return "", nil
	}
	return data, err
}
