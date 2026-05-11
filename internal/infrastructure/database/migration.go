package database

import (
	"fmt"
	"github.com/jmoiron/sqlx"
)

// Migration representa uma migração de banco de dados.
type Migration struct {
	ID   string
	SQL  string
}

// RunMigrations executa as migrações iniciais para garantir a estrutura do banco.
// No Spec Wizard, isso garante que as tabelas de sprints e tasks existam.
func RunMigrations(db *sqlx.DB) error {
	migrations := []Migration{
		{
			ID: "20240511_init_tasks",
			SQL: `CREATE TABLE IF NOT EXISTS tasks (
				id TEXT PRIMARY KEY,
				title TEXT,
				status TEXT,
				created_at DATETIME
			);`,
		},
	}

	for _, m := range migrations {
		fmt.Printf("🚀 Executando migração: %s\n", m.ID)
		if _, err := db.Exec(m.SQL); err != nil {
			return fmt.Errorf("falha na migração %s: %w", m.ID, err)
		}
	}

	return nil
}
