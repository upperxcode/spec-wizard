package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".spec-wizard", "governance.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	columns := []string{
		"language", "expert", "architecture", "domain", "prd",
		"functional_requirements", "non_functional_requirements",
		"data_strategy", "state_management", "api_contract",
		"customization", "instructions", "user_language",
	}

	for _, col := range columns {
		query := fmt.Sprintf("ALTER TABLE projects ADD COLUMN %s TEXT", col)
		_, err := db.Exec(query)
		if err != nil {
			fmt.Printf("⚠️ Column %s already exists or error: %v\n", col, err)
		} else {
			fmt.Printf("✅ Column %s added.\n", col)
		}
	}
	fmt.Println("🚀 Schema update complete.")
}
