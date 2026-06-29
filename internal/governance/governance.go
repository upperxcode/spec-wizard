package governance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"spec-wizard/internal/logger"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// Roadmap struct for API responses
type Roadmap struct {
	Name                      string `json:"name"`
	ProjectName               string `json:"project_name"`
	Stack                     string `json:"stack"`
	Language                  string `json:"language"`
	Architecture              string `json:"architecture"`
	Domain                    string `json:"domain"`
	FunctionalRequirements    string `json:"functionalRequirements"`
	NonFunctionalRequirements string `json:"nonFunctionalRequirements"`
	DataStrategy              string `json:"dataStrategy"`
	StateManagement           string `json:"stateManagement"`
	ApiContract               string `json:"apiContract"`
	Customization             string `json:"customization"`
	Instructions              string `json:"instructions"`
	Sprints []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Goal  string `json:"goal"`
		Tasks []struct {
			ID                 string `json:"id"`
			Title              string `json:"title"`
			Description        string `json:"description"`
			Status             string `json:"status"`
			Bugs               string `json:"bugs"`
			Notes              string `json:"notes"`
			TestLogs           string `json:"test_logs"`
			TestStatus         string `json:"test_status"`
			AcceptanceCriteria []struct {
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"acceptance_criteria"`
			Files     interface{} `json:"files"`
			TestFiles interface{} `json:"test_files"`
		} `json:"tasks"`
	} `json:"sprints"`
}

// DB Structs
type DBProject struct {
	ID                        int    `db:"id" json:"id"`
	Name                      string `db:"name" json:"name"`
	Stack                     string `db:"stack" json:"stack"`
	Path                      string `db:"path" json:"path"`
	Language                  string `db:"language" json:"language"`
	Architecture              string `db:"architecture" json:"architecture"`
	Domain                    string `db:"domain" json:"domain"`
	PRD                       string `db:"prd" json:"prd"`
	FunctionalRequirements    string `db:"functional_requirements" json:"functional_requirements"`
	NonFunctionalRequirements string `db:"non_functional_requirements" json:"non_functional_requirements"`
	DataStrategy              string `db:"data_strategy" json:"data_strategy"`
	StateManagement           string `db:"state_management" json:"state_management"`
	ApiContract               string `db:"api_contract" json:"api_contract"`
	Customization             string `db:"customization" json:"customization"`
	Instructions              string `db:"instructions" json:"instructions"`
	UserLanguage              string `db:"user_language" json:"user_language"`
}

type DBSprint struct {
	ID        string `db:"id"`
	ProjectID int    `db:"project_id"`
	Title     string `db:"title"`
	Goal      string `db:"goal"`
	Status    string `db:"status"`
}

type DBTask struct {
	ID            string `db:"id"`
	SprintID      string `db:"sprint_id"`
	Title         string `db:"title"`
	Description   string `db:"description"`
	Status        string `db:"status"`
	TechnicalSpec string `db:"technical_spec"`
	Bugs          string `db:"bugs"`
	Notes         string `db:"notes"`
}

type FileMap struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

// GetDB retorna a instância do banco de dados (singleton-ish para o pacote)
func GetDB() *sql.DB {
	return db
}

// InitDB inicializa o banco de dados global no diretório ~/.spec-wizard
func InitDB(projectPath string) error {
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".spec-wizard")
	dbPath := filepath.Join(dbDir, "governance.db")
	os.MkdirAll(dbDir, 0755)

	if db != nil {
		db.Close()
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("falha ao abrir banco sqlite global: %v", err)
	}

	// 🏛️ ESQUEMA MULTI-PROJETO BLINDADO (Isolation Level: Project)
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		stack TEXT,
		path TEXT UNIQUE,
		language TEXT,
		expert TEXT,
		architecture TEXT,
		domain TEXT,
		prd TEXT,
		functional_requirements TEXT,
		non_functional_requirements TEXT,
		data_strategy TEXT,
		state_management TEXT,
		api_contract TEXT,
		customization TEXT,
		instructions TEXT,
		user_language TEXT
	);

	CREATE TABLE IF NOT EXISTS sprints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		display_id INTEGER,
		title TEXT,
		goal TEXT,
		status TEXT,
		UNIQUE(project_id, display_id),
		FOREIGN KEY(project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		display_id INTEGER,
		sprint_id INTEGER,
		title TEXT,
		description TEXT,
		status TEXT,
		bugs TEXT,
		notes TEXT,
		test_logs TEXT,
		test_status TEXT,
		UNIQUE(project_id, sprint_id, display_id),
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(sprint_id) REFERENCES sprints(id)
	);

	CREATE TABLE IF NOT EXISTS task_details (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		full_context TEXT,
		technical_requirements TEXT,
		implementation_notes TEXT,
		UNIQUE(project_id, task_id),
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(task_id) REFERENCES tasks(id)
	);

	CREATE TABLE IF NOT EXISTS acceptance_criteria (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		description TEXT,
		status TEXT DEFAULT 'pending',
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(task_id) REFERENCES tasks(id)
	);

	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		path TEXT,
		content_hash TEXT,
		last_modified DATETIME,
		UNIQUE(project_id, path),
		FOREIGN KEY(project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS task_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		file_id INTEGER,
		file_type TEXT, -- 'source', 'test', 'doc'
		action TEXT,    -- 'created', 'modified', 'deleted'
		UNIQUE(project_id, task_id, file_id),
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(task_id) REFERENCES tasks(id),
		FOREIGN KEY(file_id) REFERENCES files(id)
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		command TEXT,
		result TEXT,
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(task_id) REFERENCES tasks(id)
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("erro ao aplicar schema blindado: %v", err)
	}

	// 🔄 MIGRATIONS INCREMENTAIS — adiciona colunas em bancos existentes sem recriar
	migrations := []string{
		"ALTER TABLE projects ADD COLUMN expert TEXT",
		"ALTER TABLE projects ADD COLUMN prd TEXT",
	}
	for _, m := range migrations {
		db.Exec(m) // Ignora erro "column already exists" intencionalmente
	}

	return nil
}

// GetOrCreateProject recupera ou cria um projeto no banco global baseado no path
func GetOrCreateProject(projectPath string) (int, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return 0, err
	}

	var id int
	err = db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO projects (name, path) VALUES (?, ?)", filepath.Base(absPath), absPath)
		if err != nil {
			return 0, err
		}
		lastID, _ := res.LastInsertId()
		return int(lastID), nil
	}
	return id, err
}

func SetProjectGoal(projectPath, goal string) string {
	if db == nil {
		if err := InitDB(projectPath); err != nil {
			return fmt.Sprintf("❌ Erro ao inicializar banco: %v", err)
		}
	}

	absPath, _ := filepath.Abs(projectPath)
	name := filepath.Base(absPath)

	// Garante que o projeto existe
	pID, err := GetOrCreateProject(absPath)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao acessar projeto: %v", err)
	}

	_, err = db.Exec(`
		UPDATE projects 
		SET functional_requirements = ?, instructions = ?, name = ?, user_language = ?
		WHERE id = ?`, goal, goal, name, "pt-BR", pID)
	
	if err != nil {
		return fmt.Sprintf("❌ Erro ao salvar objetivo: %v", err)
	}

	return fmt.Sprintf("✅ Objetivo do projeto definido com sucesso!\nCaminho: %s\nObjetivo: %s", absPath, goal)
}

// HandleSync lê o arquivo local e atualiza o banco
func HandleSync(projectPath string) string {
	targetPath := filepath.Join(projectPath, ".spec-wizard", "sprints.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao ler sprints.json: %v. Tente rodar 'wz init' primeiro.", err)
	}

	if err := UpdateFromRoadmapJSON(projectPath, data); err != nil {
		return fmt.Sprintf("❌ Erro ao sincronizar banco: %v", err)
	}

	return "✅ Banco de dados sincronizado com .spec-wizard/sprints.json"
}

// UpdateFromRoadmapJSON processa um objeto Roadmap vindo da API com proteção de status
// GetProjectProfile retorna os metadados técnicos do projeto em JSON
func GetProjectProfile(projectPath string) string {
	absPath, _ := filepath.Abs(projectPath)
	var p DBProject
	err := db.QueryRow(`SELECT name, stack, language, architecture, domain, 
		functional_requirements, non_functional_requirements, data_strategy, 
		state_management, api_contract, customization, instructions 
		FROM projects WHERE path = ?`, absPath).Scan(
		&p.Name, &p.Stack, &p.Language, &p.Architecture, &p.Domain,
		&p.FunctionalRequirements, &p.NonFunctionalRequirements, &p.DataStrategy,
		&p.StateManagement, &p.ApiContract, &p.Customization, &p.Instructions,
	)
	if err != nil {
		return fmt.Sprintf("{\"error\": \"Projeto não encontrado ou sem perfil: %v\"}", err)
	}

	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data)
}

// UpdateProjectProfile atualiza os metadados técnicos do projeto
func UpdateProjectProfile(projectPath string, data []byte) string {
	// Usa mapa para atualizar apenas os campos presentes no JSON (update parcial)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Sprintf("❌ Erro ao processar JSON de perfil: %v", err)
	}

	if len(raw) == 0 {
		return "⚠️ Nenhum campo fornecido para atualizar."
	}

	pID, err := GetOrCreateProject(projectPath)
	if err != nil {
		return "❌ Erro ao identificar projeto: " + err.Error()
	}

	// Campos permitidos para atualização
	allowed := map[string]bool{
		"name": true, "stack": true, "language": true, "tech": true, "expert": true,
		"architecture": true, "domain": true, "prd": true,
		"functional_requirements": true, "non_functional_requirements": true,
		"data_strategy": true, "state_management": true,
		"api_contract": true, "customization": true, "instructions": true,
		"user_language": true,
	}

	var setClauses []string
	var args []interface{}
	for k, v := range raw {
		if !allowed[k] {
			continue
		}
		columnName := k
		if k == "tech" {
			columnName = "language"
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", columnName))
		args = append(args, v)
	}

	if len(setClauses) == 0 {
		return "⚠️ Nenhum campo válido para atualizar. Campos permitidos: name, stack, tech, expert, architecture, domain, prd, functional_requirements, non_functional_requirements."
	}

	args = append(args, pID)
	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	if _, err = db.Exec(query, args...); err != nil {
		return "❌ Erro ao atualizar perfil no banco: " + err.Error()
	}

	// Confirma o que foi salvo com os comandos para a IA
	saved := make([]string, 0, len(raw))
	for k := range raw {
		if allowed[k] {
			saved = append(saved, k)
		}
	}
	return fmt.Sprintf("✅ Profile updated successfully! Fields saved: %s\n\nTo verify: run `wz profile get`", strings.Join(saved, ", "))
}

func UpdateFromRoadmapJSON(projectPath string, data []byte) error {
	if db == nil {
		return fmt.Errorf("banco não inicializado")
	}

	var roadmap Roadmap
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return fmt.Errorf("falha ao parsear roadmap: %v", err)
	}

	pID, err := GetOrCreateProject(projectPath)
	if err != nil {
		return fmt.Errorf("falha ao registrar projeto: %v", err)
	}

	// Atualiza metadados do projeto a partir do roadmap
	db.Exec(`UPDATE projects SET name=?, stack=?, language=?, architecture=?, domain=?, 
		functional_requirements=?, non_functional_requirements=?, data_strategy=?, 
		state_management=?, api_contract=?, customization=?, instructions=? WHERE id=?`,
		roadmap.ProjectName, roadmap.Stack, roadmap.Language, roadmap.Architecture, roadmap.Domain,
		roadmap.FunctionalRequirements, roadmap.NonFunctionalRequirements, roadmap.DataStrategy,
		roadmap.StateManagement, roadmap.ApiContract, roadmap.Customization, roadmap.Instructions, pID)

	for _, s := range roadmap.Sprints {
		var sDisplayID int
		fmt.Sscanf(s.ID, "%d", &sDisplayID)

		// Busca ou cria a Sprint no contexto do projeto
		var realSprintID int64
		err := db.QueryRow("SELECT id FROM sprints WHERE project_id = ? AND display_id = ?", pID, sDisplayID).Scan(&realSprintID)
		if err == sql.ErrNoRows {
			res, err := db.Exec(`INSERT INTO sprints (project_id, display_id, title, goal, status) 
				VALUES (?, ?, ?, ?, ?)`,
				pID, sDisplayID, s.Title, s.Goal, "pending")
			if err != nil {
				logger.Warn("⚠️ Erro ao criar sprint", "id", sDisplayID, "error", err)
				continue
			}
			realSprintID, _ = res.LastInsertId()
		} else {
			db.Exec("UPDATE sprints SET title=?, goal=? WHERE id=?", s.Title, s.Goal, realSprintID)
		}

		for _, t := range s.Tasks {
			var tDisplayID int
			fmt.Sscanf(t.ID, "%d", &tDisplayID)

			// Busca ou cria a Task no contexto da Sprint
			var realTaskID int64
			err = db.QueryRow("SELECT id FROM tasks WHERE sprint_id = ? AND display_id = ?", realSprintID, tDisplayID).Scan(&realTaskID)
			
			if err == sql.ErrNoRows {
				res, err := db.Exec(`INSERT INTO tasks (project_id, display_id, sprint_id, title, description, status) 
					VALUES (?, ?, ?, ?, ?, ?)`,
					pID, tDisplayID, realSprintID, t.Title, t.Description, t.Status)
				if err != nil {
					logger.Warn("⚠️ Erro ao criar task", "id", tDisplayID, "error", err)
					continue
				}
				realTaskID, _ = res.LastInsertId()
			} else {
				// Atualiza se necessário, mas protege status se já estiver completed
				var currentStatus string
				db.QueryRow("SELECT status FROM tasks WHERE id=?", realTaskID).Scan(&currentStatus)
				if currentStatus != "completed" {
					db.Exec("UPDATE tasks SET title=?, description=?, status=? WHERE id=?", 
						t.Title, t.Description, t.Status, realTaskID)
				}
			}

			// Atualiza detalhes extras (bugs, notes, etc.)
			_, err = db.Exec(`UPDATE tasks SET bugs=?, notes=?, test_logs=?, test_status=? WHERE id=?`,
				t.Bugs, t.Notes, t.TestLogs, t.TestStatus, realTaskID)
			
			if err != nil {
				logger.Warn("⚠️ Erro ao atualizar detalhes da task", "id", tDisplayID, "error", err)
			}

			// 🛡️ PROTEÇÃO DE CRITÉRIOS: Só atualiza se a IA enviar dados. Nunca apaga.
			if len(t.AcceptanceCriteria) > 0 {
				for _, ac := range t.AcceptanceCriteria {
					db.Exec(`INSERT INTO acceptance_criteria (task_id, description, status) VALUES (?, ?, ?)
						ON CONFLICT(task_id, description) DO UPDATE SET 
							status = CASE WHEN excluded.status != '' THEN excluded.status ELSE acceptance_criteria.status END`,
						realTaskID, ac.Description, ac.Status)
				}
			}

			// 🛡️ PROTEÇÃO DE ARQUIVOS: Sempre aditivo (Merge).
			if t.Files != nil {
				processFiles(int(realTaskID), t.Files, "scope")
			}
			if t.TestFiles != nil {
				processFiles(int(realTaskID), t.TestFiles, "test")
			}
			
			SyncTaskMarkdown(projectPath, int(realTaskID))
		}
	}

	GenerateProjectMap(projectPath)
	return nil
}


// SyncTaskMarkdown gera o .md da tarefa a partir do banco

// LinkFileToTask vincula um arquivo a uma tarefa (Merge Aditivo)
func LinkFileToTask(projectPath string, taskID int, filePath, fileType, action string) error {
	if db == nil {
		return fmt.Errorf("banco não inicializado")
	}

	// 1. Registra o arquivo na tabela global se não existir
	_, err := db.Exec("INSERT INTO files (path, last_modified) VALUES (?, CURRENT_TIMESTAMP) ON CONFLICT(path) DO UPDATE SET last_modified=excluded.last_modified", filePath)
	if err != nil {
		return err
	}

	var fileID int64
	err = db.QueryRow("SELECT id FROM files WHERE path = ?", filePath).Scan(&fileID)
	if err != nil {
		return err
	}

	// 2. Vincula à tarefa
	_, err = db.Exec(`INSERT INTO task_files (task_id, file_id, file_type, action) VALUES (?, ?, ?, ?)
		ON CONFLICT(task_id, file_id) DO UPDATE SET action=excluded.action`,
		taskID, fileID, fileType, action)
	
	return err
}

// UpdateTaskStatus atualiza o status via display_id (Interface Humana)
func UpdateTaskStatus(projectPath string, displayID int, status, testLogs string) error {
	if db == nil {
		return fmt.Errorf("banco não inicializado")
	}

	taskID, err := getTaskIDByDisplayID(projectPath, fmt.Sprintf("%d", displayID))
	if err != nil {
		return err
	}

	return updateTaskStatusByID(taskID, status, testLogs)
}

// updateTaskStatusByID atualiza o status via ID real (Interface Interna)
func updateTaskStatusByID(taskID int, status, testLogs string) error {
	if db == nil { return nil }
	
	// 🛡️ PROTEÇÃO: Exigir logs para status Completed
	if status == "Completed" && testLogs == "" {
		var currentLogs string
		db.QueryRow("SELECT test_logs FROM tasks WHERE id = ?", taskID).Scan(&currentLogs)
		if currentLogs == "" {
			return fmt.Errorf("impossível marcar como Completed sem evidência de testes")
		}
	}

	_, err := db.Exec(`UPDATE tasks SET status = ?, test_logs = CASE WHEN ? != '' THEN ? ELSE test_logs END WHERE id = ?`,
		status, testLogs, testLogs, taskID)
	
	return err
}

// GetTaskInfoWrapper para o MCP Server que lida com múltiplos retornos
func GetRoadmap(projectPath string) string {
	if db == nil {
		if err := InitDB(projectPath); err != nil {
			return fmt.Sprintf("❌ Erro: Banco não inicializado e falha ao abrir: %v", err)
		}
	}

	pID, err := GetOrCreateProject(projectPath)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao identificar projeto: %v", err)
	}

	rows, err := db.Query(`
		SELECT s.id, s.title, s.status, t.id, t.title, t.status
		FROM sprints s
		LEFT JOIN tasks t ON s.id = t.sprint_id
		WHERE s.project_id = ?
		ORDER BY s.id, t.id`, pID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao buscar roadmap: %v", err)
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 Roadmap do Projeto: %s\n\n", filepath.Base(projectPath)))

	currentSprintID := -1
	for rows.Next() {
		var sID int
		var sTitle, sStatus string
		var tIDPtr *int
		var tTitlePtr, tStatusPtr *string

		if err := rows.Scan(&sID, &sTitle, &sStatus, &tIDPtr, &tTitlePtr, &tStatusPtr); err != nil {
			continue
		}

		if sID != currentSprintID {
			sb.WriteString(fmt.Sprintf("\n📦 Sprint %d: %s [%s]\n", sID, sTitle, sStatus))
			currentSprintID = sID
		}

		if tIDPtr != nil {
			sb.WriteString(fmt.Sprintf("  - [%s] Task %d: %s\n", *tStatusPtr, *tIDPtr, *tTitlePtr))
		}
	}

	res := sb.String()
	if currentSprintID == -1 {
		return "📭 Nenhum roadmap encontrado para este projeto. Use 'wz info' para definir os objetivos e depois 'wz init' para gerar o planejamento."
	}

	return res
}

func GetTaskInfo(projectPath, id string) string {
	res, err := GetTaskInfoInternal(projectPath, id)
	if err != nil {
		return fmt.Sprintf("❌ Erro: %v", err)
	}
	return res
}

func SyncTaskMarkdown(projectPath string, taskID int) {
	if db == nil {
		return
	}
	var title, desc, status, bugs, notes string
	err := db.QueryRow(`
		SELECT title, description, status, 
		       COALESCE(bugs, ''), 
		       COALESCE(notes, '') 
		FROM tasks WHERE id = ?`, taskID).Scan(
		&title, &desc, &status, &bugs, &notes)
	if err != nil {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 🎯 TASK SPEC: %s\n", title))
	sb.WriteString(fmt.Sprintf("*(ID: %d)*\n\n", taskID))
	sb.WriteString("## 📋 Visão Geral\n")
	sb.WriteString(desc + "\n\n")
	sb.WriteString(fmt.Sprintf("## 🚀 Status: %s\n\n", status))

	sb.WriteString("## ✅ Critérios de Aceite\n")
	rows, _ := db.Query("SELECT description, status FROM acceptance_criteria WHERE task_id = ?", taskID)
	if rows != nil {
		for rows.Next() {
			var cDesc, cStatus string
			rows.Scan(&cDesc, &cStatus)
			icon := "⚪"
			if cStatus == "completed" {
				icon = "✅"
			}
			sb.WriteString(fmt.Sprintf("- %s %s\n", icon, cDesc))
		}
		rows.Close()
	}
	sb.WriteString("\n")

	sb.WriteString("## 📂 Arquivos Vinculados\n")
	sb.WriteString("| Arquivo | Tipo | Ação | Descrição |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- |\n")
	fRows, _ := db.Query(`
		SELECT f.path, tf.file_type, tf.action, f.description 
		FROM files f 
		JOIN task_files tf ON f.id = tf.file_id 
		WHERE tf.task_id = ?`, taskID)
	if fRows != nil {
		for fRows.Next() {
			var fPath, fType, fAction, fDesc string
			fRows.Scan(&fPath, &fType, &fAction, &fDesc)
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", fPath, fType, fAction, fDesc))
		}
		fRows.Close()
	}
	sb.WriteString("\n")

	if bugs != "" {
		sb.WriteString("## 🐞 Bugs Reportados\n")
		sb.WriteString("> [!CAUTION]\n")
		sb.WriteString("> " + bugs + "\n\n")
	}

	if notes != "" {
		sb.WriteString("## 📝 Notas\n")
		sb.WriteString(notes + "\n")
	}

	targetDir := filepath.Join(projectPath, ".spec-wizard/tasks")
	os.MkdirAll(targetDir, 0755)
	
	fileName := fmt.Sprintf("task-%d.md", taskID)
	os.WriteFile(filepath.Join(targetDir, fileName), []byte(sb.String()), 0644)
}

// GenerateProjectMap gera o MAP.md a partir do banco
func GenerateProjectMap(projectPath string) {
	if db == nil {
		return
	}
	var sb strings.Builder
	sb.WriteString("# 🗺️ Mapa de Arquivos do Projeto\n\n")
	sb.WriteString("Este arquivo é gerado automaticamente pelo **Spec Wizard**. Ele mapeia cada arquivo físico à sua responsabilidade e tarefa de origem.\n\n")

	var pID int
	absPath, _ := filepath.Abs(projectPath)
	db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&pID)

	sprintRows, _ := db.Query("SELECT id, title FROM sprints WHERE project_id = ?", pID)
	if sprintRows != nil {
		defer sprintRows.Close()
		for sprintRows.Next() {
			var sID int
			var sTitle string
			sprintRows.Scan(&sID, &sTitle)
			sb.WriteString(fmt.Sprintf("## 🏃 Sprint: %s\n", sTitle))

			taskRows, _ := db.Query("SELECT id, display_id, title, description FROM tasks WHERE sprint_id = ?", sID)
			if taskRows != nil {
				for taskRows.Next() {
					var tID int
					var tDisplayID int
					var tTitle, tDesc string
					taskRows.Scan(&tID, &tDisplayID, &tTitle, &tDesc)
					sb.WriteString(fmt.Sprintf("### 🎯 Tarefa %d: %s\n", tDisplayID, tTitle))
					sb.WriteString(tDesc + "\n\n")

					sb.WriteString("| Arquivo | Descrição |\n")
					sb.WriteString("| :--- | :--- |\n")
					
					fileRows, _ := db.Query(`
						SELECT f.path, f.description 
						FROM files f 
						JOIN task_files tf ON f.id = tf.file_id 
						WHERE tf.task_id = ?`, tID)
					
					hasFiles := false
					if fileRows != nil {
						for fileRows.Next() {
							var fPath, fDesc string
							fileRows.Scan(&fPath, &fDesc)
							sb.WriteString(fmt.Sprintf("| `%s` | %s |\n", fPath, fDesc))
							hasFiles = true
						}
						fileRows.Close()
					}
					if !hasFiles {
						sb.WriteString("| - | Nenhum arquivo mapeado ainda |\n")
					}
					sb.WriteString("\n")
				}
				taskRows.Close()
			}
		}
	}

	mapPath := filepath.Join(projectPath, "docs/MAP.md")
	os.MkdirAll(filepath.Dir(mapPath), 0755)
	os.WriteFile(mapPath, []byte(sb.String()), 0644)
}

// SyncDBToJSON sincroniza o estado do banco global de volta para o arquivo local do projeto.
func SyncDBToJSON(projectPath string) error {
	if db == nil {
		if err := InitDB(projectPath); err != nil {
			return err
		}
	}

	pID, err := GetOrCreateProject(projectPath)
	if err != nil {
		return err
	}

	var roadmap Roadmap
	// Busca dados básicos do projeto com proteção contra NULL
	err = db.QueryRow(`SELECT name, COALESCE(language, ''), COALESCE(architecture, ''), 
	                          COALESCE(data_strategy, ''), COALESCE(state_management, ''), 
	                          COALESCE(instructions, '') 
	                   FROM projects WHERE id = ?`, pID).Scan(
		&roadmap.ProjectName, &roadmap.Language, &roadmap.Architecture, 
		&roadmap.DataStrategy, &roadmap.StateManagement, &roadmap.Instructions)
	
	if err != nil {
		return err
	}

	// Busca Sprints
	sRows, err := db.Query("SELECT id, title, goal FROM sprints WHERE project_id = ? ORDER BY id", pID)
	if err != nil {
		return err
	}
	defer sRows.Close()

	for sRows.Next() {
		var s struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Goal  string `json:"goal"`
			Tasks []struct {
				ID                 string `json:"id"`
				Title              string `json:"title"`
				Description        string `json:"description"`
				Status             string `json:"status"`
				Bugs               string `json:"bugs"`
				Notes              string `json:"notes"`
				TestLogs           string `json:"test_logs"`
				TestStatus         string `json:"test_status"`
				AcceptanceCriteria []struct {
					Description string `json:"description"`
					Status      string `json:"status"`
				} `json:"acceptance_criteria"`
				Files     interface{} `json:"files"`
				TestFiles interface{} `json:"test_files"`
			} `json:"tasks"`
		}
		var sIDInt int
		sRows.Scan(&sIDInt, &s.Title, &s.Goal)
		s.ID = fmt.Sprintf("%d", sIDInt)

		// Busca Tasks desta Sprint
		tRows, err := db.Query(`SELECT id, display_id, title, description, status, 
		                               COALESCE(bugs, ''), COALESCE(notes, ''), 
		                               COALESCE(test_logs, ''), COALESCE(test_status, '') 
		                        FROM tasks WHERE sprint_id = ? ORDER BY display_id`, sIDInt)
		if err == nil {
			for tRows.Next() {
				var t struct {
					ID                 string `json:"id"`
					Title              string `json:"title"`
					Description        string `json:"description"`
					Status             string `json:"status"`
					Bugs               string `json:"bugs"`
					Notes              string `json:"notes"`
					TestLogs           string `json:"test_logs"`
					TestStatus         string `json:"test_status"`
					AcceptanceCriteria []struct {
						Description string `json:"description"`
						Status      string `json:"status"`
					} `json:"acceptance_criteria"`
					Files     interface{} `json:"files"`
					TestFiles interface{} `json:"test_files"`
				}
				var realID, dispID int
				tRows.Scan(&realID, &dispID, &t.Title, &t.Description, &t.Status, &t.Bugs, &t.Notes, &t.TestLogs, &t.TestStatus)
				t.ID = fmt.Sprintf("%d", dispID)

				// Busca Critérios de Aceite
				acRows, _ := db.Query("SELECT description, status FROM acceptance_criteria WHERE task_id = ?", realID)
				if acRows != nil {
					for acRows.Next() {
						var ac struct {
							Description string `json:"description"`
							Status      string `json:"status"`
						}
						acRows.Scan(&ac.Description, &ac.Status)
						t.AcceptanceCriteria = append(t.AcceptanceCriteria, ac)
					}
					acRows.Close()
				}

				// Busca Arquivos (Scope)
				var scopeFiles []string
				fRows, _ := db.Query(`SELECT f.path FROM files f JOIN task_files tf ON f.id = tf.file_id 
				                      WHERE tf.task_id = ? AND tf.file_type = 'scope'`, realID)
				if fRows != nil {
					for fRows.Next() {
						var p string
						fRows.Scan(&p)
						scopeFiles = append(scopeFiles, p)
					}
					fRows.Close()
				}
				t.Files = scopeFiles

				// Busca Arquivos (Test)
				var testFiles []string
				tfRows, _ := db.Query(`SELECT f.path FROM files f JOIN task_files tf ON f.id = tf.file_id 
				                       WHERE tf.task_id = ? AND tf.file_type = 'test'`, realID)
				if tfRows != nil {
					for tfRows.Next() {
						var p string
						tfRows.Scan(&p)
						testFiles = append(testFiles, p)
					}
					tfRows.Close()
				}
				t.TestFiles = testFiles

				s.Tasks = append(s.Tasks, t)
			}
			tRows.Close()
		}
		roadmap.Sprints = append(roadmap.Sprints, s)
	}

	// Salva o roadmap (sprints.json)
	jsonData, _ := json.MarshalIndent(roadmap, "", "  ")
	targetPath := filepath.Join(projectPath, ".spec-wizard", "sprints.json")
	os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err := os.WriteFile(targetPath, jsonData, 0644); err != nil {
		return err
	}

	// Gera o config.json compatível com o Orchestrator (ProjectConfig)
	configData := map[string]interface{}{
		"projectName":  roadmap.ProjectName,
		"language":     roadmap.Language,
		"architecture": roadmap.Architecture,
		"patterns":     []string{roadmap.Architecture}, // Usa Architecture como padrão
		"dataStrategy": roadmap.DataStrategy,
		"stateManagement": roadmap.StateManagement,
		"instructions": roadmap.Instructions,
	}
	
	configJSON, _ := json.MarshalIndent(configData, "", "  ")
	configPath := filepath.Join(projectPath, ".spec-wizard", "config.json")
	return os.WriteFile(configPath, configJSON, 0644)
}

// HandleUpdateTask unificado
func HandleUpdateTask(projectPath string, args []string) string {
	if len(args) < 1 {
		return "❌ Erro: ID da tarefa não fornecido."
	}
	displayIDStr := args[0]
	status := extractArg(args, "--status")
	bugs := extractArg(args, "--bugs")
	notes := extractArg(args, "--notes")
	mapArg := extractArg(args, "--map")

	if db == nil {
		return "❌ Erro: Banco de dados não inicializado."
	}

	// Resolve o displayID para ID real
	taskID, err := getTaskIDByDisplayID(projectPath, displayIDStr)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao localizar tarefa: %v", err)
	}

	query := "UPDATE tasks SET "
	params := []interface{}{}
	updates := []string{}

	if status != "" {
		updates = append(updates, "status = ?")
		params = append(params, status)
	}
	if bugs != "" {
		updates = append(updates, "bugs = ?")
		params = append(params, bugs)
	}
	if notes != "" {
		updates = append(updates, "notes = ?")
		params = append(params, notes)
	}

	if len(updates) > 0 {
		query += strings.Join(updates, ", ") + " WHERE id = ?"
		params = append(params, taskID)
		_, err := db.Exec(query, params...)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao atualizar tarefa no banco: %v", err)
		}
	}

	if mapArg != "" {
		handleFilesUpdate(taskID, mapArg, "scope")
	}

	testMapArg := extractArg(args, "--test-map")
	if testMapArg != "" {
		handleFilesUpdate(taskID, testMapArg, "test")
	}

	// Reatividade Completa
	SyncTaskMarkdown(projectPath, taskID)
	GenerateProjectMap(projectPath)

	return fmt.Sprintf("✅ Tarefa %s atualizada no banco global.", displayIDStr)
}

func handleFilesUpdate(taskID int, mapJSON, fileType string) {
	var files []FileMap
	err := json.Unmarshal([]byte(mapJSON), &files)
	if err == nil {
		for _, f := range files {
			db.Exec("INSERT OR IGNORE INTO files (path, description) VALUES (?, ?)", f.Path, f.Description)
			db.Exec("UPDATE files SET description = ? WHERE path = ?", f.Description, f.Path)
			var fID int
			db.QueryRow("SELECT id FROM files WHERE path = ?", f.Path).Scan(&fID)
			db.Exec("INSERT OR IGNORE INTO task_files (task_id, file_id, file_type, action) VALUES (?, ?, ?, ?)",
				taskID, fID, fileType, "modify")
		}
	}
}

// SaveProject salva ou atualiza a especificação completa do projeto no banco
func SaveProject(p DBProject) error {
	if db == nil {
		return fmt.Errorf("banco não inicializado")
	}

	_, err := db.Exec(`INSERT INTO projects (
		name, stack, path, language, architecture, domain, 
		functional_requirements, non_functional_requirements, 
		data_strategy, state_management, api_contract, 
		customization, instructions, user_language
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		name = excluded.name,
		stack = excluded.stack,
		language = excluded.language,
		architecture = excluded.architecture,
		domain = excluded.domain,
		functional_requirements = excluded.functional_requirements,
		non_functional_requirements = excluded.non_functional_requirements,
		data_strategy = excluded.data_strategy,
		state_management = excluded.state_management,
		api_contract = excluded.api_contract,
		customization = excluded.customization,
		instructions = excluded.instructions,
		user_language = excluded.user_language`,
		p.Name, p.Stack, p.Path, p.Language, p.Architecture, p.Domain,
		p.FunctionalRequirements, p.NonFunctionalRequirements,
		p.DataStrategy, p.StateManagement, p.ApiContract,
		p.Customization, p.Instructions, p.UserLanguage)
	
	return err
}

func processFiles(taskID int, files interface{}, fileType string) {
	if files == nil { return }
	
	var list []interface{}
	switch v := files.(type) {
	case []interface{}:
		list = v
	case []string:
		for _, s := range v {
			list = append(list, s)
		}
	case string:
		// Suporta JSON string vindo de alguns fluxos da API
		if err := json.Unmarshal([]byte(v), &list); err != nil {
			list = append(list, v)
		}
	default:
		return
	}

	for _, item := range list {
		var path, desc string
		
		switch v := item.(type) {
		case string:
			path = v
		case map[string]interface{}:
			if p, ok := v["path"].(string); ok { path = p }
			if d, ok := v["description"].(string); ok { desc = d }
		}
		
		if path != "" {
			// 1. Garante que o arquivo existe na tabela de arquivos
			db.Exec("INSERT INTO files (path, description) VALUES (?, ?) ON CONFLICT(path) DO UPDATE SET description = CASE WHEN excluded.description != '' THEN excluded.description ELSE files.description END", 
				path, desc)
			
			var fID int
			db.QueryRow("SELECT id FROM files WHERE path = ?", path).Scan(&fID)
			
			// 2. Cria o vínculo com a tarefa se não existir (ADITIVO)
			// Não deletamos vínculos anteriores, apenas adicionamos novos ou atualizamos a ação.
			db.Exec(`INSERT INTO task_files (task_id, file_id, file_type, action) 
				VALUES (?, ?, ?, ?) 
				ON CONFLICT(task_id, file_id) DO UPDATE SET 
					file_type = excluded.file_type,
					action = CASE WHEN excluded.action != '' THEN excluded.action ELSE task_files.action END`,
				taskID, fID, fileType, "modify")
		}
	}
}

func extractArg(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// AddAuditLog registra uma ação de governança no banco
func AddAuditLog(command, result string, taskID int) {
	if db == nil { return }
	db.Exec("INSERT INTO audit_logs (command, result, task_id) VALUES (?, ?, ?)", command, result, taskID)
}

// getTaskIDByDisplayID resolve um display_id (1, 2...) no contexto do projeto atual
func getTaskIDByDisplayID(projectPath string, displayID string) (int, error) {
	absPath, _ := filepath.Abs(projectPath)
	var taskID int
	query := `
		SELECT id 
		FROM tasks 
		WHERE project_id = (SELECT id FROM projects WHERE path = ?) 
		AND (display_id = ? OR id = ?)
	`
	err := db.QueryRow(query, absPath, displayID, displayID).Scan(&taskID)
	if err != nil {
		return 0, fmt.Errorf("tarefa não encontrada para o projeto atual: %v", err)
	}
	return taskID, nil
}

// HandleAuditTask executa a auditoria técnica de uma tarefa, rodando seus testes
func HandleAuditTask(projectPath string, args []string) string {
	if len(args) < 1 {
		return "❌ Erro: ID da tarefa não fornecido (ex: wz audit 1)"
	}
	displayID := args[0]

	// 1. Resolve o ID real e o ProjectID
	absPath, _ := filepath.Abs(projectPath)
	var pID int
	db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&pID)

	taskID, err := getTaskIDByDisplayID(projectPath, displayID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao encontrar tarefa: %v", err)
	}

	// 2. Busca arquivos de teste vinculados (Isolado por Projeto)
	rows, err := db.Query(`
		SELECT f.path 
		FROM task_files tf 
		JOIN files f ON tf.file_id = f.id 
		WHERE tf.project_id = ? AND tf.task_id = ? AND tf.file_type = 'test'`, pID, taskID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao buscar arquivos de teste: %v", err)
	}
	defer rows.Close()

	var testFiles []string
	for rows.Next() {
		var path string
		rows.Scan(&path)
		testFiles = append(testFiles, path)
	}

	if len(testFiles) == 0 {
		return fmt.Sprintf("⚠️ Aviso: Nenhuns arquivos de teste vinculados à Task %s. Vincule um teste primeiro.", displayID)
	}

	logger.Info("🔍 Iniciando Auditoria para Task", "id", displayID)

	// 3. Executa os testes
	// Como é um projeto Go, tentamos rodar o teste específico
	testCmdArgs := []string{"test", "-v"}
	testCmdArgs = append(testCmdArgs, testFiles...)
	
	cmd := exec.Command("go", testCmdArgs...)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	logOutput := string(output)

	// 4. Determina o novo status
	newStatus := "Completed"
	statusMsg := "✅ Auditoria Aprovada!"
	if err != nil {
		newStatus = "Failed"
		statusMsg = "❌ Auditoria Falhou! Verifique os logs."
	}

	// 5. Atualiza o banco com o log real
	updateErr := updateTaskStatusByID(taskID, newStatus, logOutput)
	if updateErr != nil {
		return fmt.Sprintf("❌ Erro ao atualizar status pós-auditoria: %v", updateErr)
	}

	return fmt.Sprintf("%s\n\n📄 LOG DE EXECUÇÃO:\n%s", statusMsg, logOutput)
}

// HandleDBAction centraliza operações de CRUD para o CLI/IA
func HandleDBAction(projectPath string, action string, args []string) string {
	if db == nil {
		return "❌ Erro: Banco de dados não inicializado."
	}
	if len(args) < 1 {
		return "❌ Erro: Entidade não fornecida (ex: task, project, sprint)."
	}

	entity := strings.ToLower(args[0])
	
	switch action {
	case "read":
		if len(args) < 2 { return "❌ Erro: ID não fornecido." }
		return dbRead(projectPath, entity, args[1])
	case "insert":
		if len(args) < 2 { return "❌ Erro: Dados JSON não fornecidos." }
		return dbInsert(projectPath, entity, args[1])
	case "remove":
		if len(args) < 2 { return "❌ Erro: ID não fornecido." }
		return dbRemove(projectPath, entity, args[1])
	case "select":
		// Select genérico via WHERE
		where := ""
		if len(args) > 1 { where = strings.Join(args[1:], " ") }
		return dbSelect(projectPath, entity, where)
	default:
		return "❌ Erro: Ação desconhecida."
	}
}

func dbRead(projectPath, entity, id string) string {
	var query string
	var row *sql.Row

	switch entity {
	case "task":
		taskID, err := getTaskIDByDisplayID(projectPath, id)
		if err != nil { return fmt.Sprintf("❌ Erro: %v", err) }
		query = "SELECT id, display_id, sprint_id, title, description, status, COALESCE(bugs,''), COALESCE(notes,''), COALESCE(test_logs,''), COALESCE(test_status,'') FROM tasks WHERE id = ?"
		row = db.QueryRow(query, taskID)
		var tID, dID, sID int
		var title, desc, status, bugs, notes, logs, tStat string
		if err := row.Scan(&tID, &dID, &sID, &title, &desc, &status, &bugs, &notes, &logs, &tStat); err != nil {
			return fmt.Sprintf("❌ Erro ao ler task: %v", err)
		}
		res, _ := json.MarshalIndent(map[string]interface{}{
			"id": tID, "display_id": dID, "sprint_id": sID, "title": title, 
			"description": desc, "status": status, "bugs": bugs, "notes": notes, 
			"test_logs": logs, "test_status": tStat,
		}, "", "  ")
		return string(res)

	case "project":
		absPath, _ := filepath.Abs(projectPath)
		query = "SELECT * FROM projects WHERE path = ?"
		// Simplificado para retornar JSON direto do banco se possível ou via Map
		var p DBProject
		err := db.QueryRow("SELECT id, name, stack, path, language, architecture, domain, functional_requirements, non_functional_requirements, data_strategy, state_management, api_contract, customization, instructions, user_language FROM projects WHERE path = ?", absPath).Scan(
			&p.ID, &p.Name, &p.Stack, &p.Path, &p.Language, &p.Architecture, &p.Domain, &p.FunctionalRequirements, &p.NonFunctionalRequirements, &p.DataStrategy, &p.StateManagement, &p.ApiContract, &p.Customization, &p.Instructions, &p.UserLanguage)
		if err != nil { return fmt.Sprintf("❌ Erro ao ler projeto: %v", err) }
		res, _ := json.MarshalIndent(p, "", "  ")
		return string(res)

	default:
		return "⚠️ Leitura para esta entidade ainda não implementada."
	}
}

func dbInsert(projectPath, entity, jsonData string) string {
	absPath, _ := filepath.Abs(projectPath)
	var projectID int
	err := db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&projectID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao localizar projeto: %v", err)
	}

	switch entity {
	case "task":
		var t struct {
			SprintID    int    `json:"sprint_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Status      string `json:"status"`
		}
		if err := json.Unmarshal([]byte(jsonData), &t); err != nil {
			return fmt.Sprintf("❌ Erro no JSON: %v", err)
		}

		var dID int
		db.QueryRow("SELECT COALESCE(MAX(display_id), 0) + 1 FROM tasks WHERE project_id = ? AND sprint_id = ?", projectID, t.SprintID).Scan(&dID)

		_, err := db.Exec("INSERT INTO tasks (project_id, display_id, sprint_id, title, description, status) VALUES (?, ?, ?, ?, ?, ?)",
			projectID, dID, t.SprintID, t.Title, t.Description, t.Status)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao inserir task: %v", err)
		}
		return fmt.Sprintf("✅ Task inserida! Display ID: %d", dID)

	case "acceptance_criteria":
		var c struct {
			TaskID      interface{} `json:"task_id"`
			Description string      `json:"description"`
			Status      string      `json:"status"`
		}
		json.Unmarshal([]byte(jsonData), &c)

		// Resolve TaskID (pode ser display_id ou ID real)
		tID, _ := getTaskIDByAnyID(projectID, c.TaskID)

		_, err := db.Exec("INSERT INTO acceptance_criteria (project_id, task_id, description, status) VALUES (?, ?, ?, ?)",
			projectID, tID, c.Description, c.Status)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao inserir critério: %v", err)
		}
		return "✅ Critério de aceite registrado."

	case "files":
		var f struct {
			Path string `json:"path"`
		}
		json.Unmarshal([]byte(jsonData), &f)
		_, err := db.Exec("INSERT OR IGNORE INTO files (project_id, path) VALUES (?, ?)", projectID, f.Path)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao registrar arquivo: %v", err)
		}
		var fileID int
		db.QueryRow("SELECT id FROM files WHERE project_id = ? AND path = ?", projectID, f.Path).Scan(&fileID)
		return fmt.Sprintf("✅ Arquivo registrado. ID: %d", fileID)

	case "task_files":
		var tf struct {
			TaskID   interface{} `json:"task_id"`
			FileID   int         `json:"file_id"`
			FileType string      `json:"file_type"`
			Action   string      `json:"action"`
		}
		json.Unmarshal([]byte(jsonData), &tf)
		tID, _ := getTaskIDByAnyID(projectID, tf.TaskID)
		_, err := db.Exec("INSERT INTO task_files (project_id, task_id, file_id, file_type, action) VALUES (?, ?, ?, ?, ?)",
			projectID, tID, tf.FileID, tf.FileType, tf.Action)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao vincular arquivo: %v", err)
		}
		return "✅ Arquivo vinculado à tarefa."

	default:
		return fmt.Sprintf("⚠️ Entidade '%s' não suportada para inserção via MCP.", entity)
	}
}

// Auxiliar para resolver TaskID de forma flexível
func getTaskIDByAnyID(projectID int, input interface{}) (int, error) {
	strID := fmt.Sprint(input)
	var id int
	// Tenta por ID real primeiro
	err := db.QueryRow("SELECT id FROM tasks WHERE project_id = ? AND id = ?", projectID, strID).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Tenta por Display ID
	err = db.QueryRow("SELECT id FROM tasks WHERE project_id = ? AND display_id = ?", projectID, strID).Scan(&id)
	return id, err
}

func dbRemove(projectPath, entity, id string) string {
	absPath, _ := filepath.Abs(projectPath)
	var projectID int
	db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&projectID)

	switch entity {
	case "task":
		taskID, _ := getTaskIDByAnyID(projectID, id)
		db.Exec("DELETE FROM acceptance_criteria WHERE project_id = ? AND task_id = ?", projectID, taskID)
		db.Exec("DELETE FROM task_files WHERE project_id = ? AND task_id = ?", projectID, taskID)
		_, err := db.Exec("DELETE FROM tasks WHERE project_id = ? AND id = ?", projectID, taskID)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao remover: %v", err)
		}
		return fmt.Sprintf("✅ Task %s removida.", id)
	default:
		return "⚠️ Remoção não implementada para esta entidade."
	}
}

func dbSelect(projectPath, entity, where string) string {
	absPath, _ := filepath.Abs(projectPath)
	var projectID int
	db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&projectID)

	// Injeta project_id obrigatoriamente
	finalWhere := fmt.Sprintf("project_id = %d", projectID)
	if where != "" {
		finalWhere += " AND (" + where + ")"
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", entity, finalWhere)
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Sprintf("❌ Erro na query: %v", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		rows.Scan(columnPointers...)

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if val != nil && *val != nil {
				// Converte []byte (comum no sqlite para strings) para string
				if b, ok := (*val).([]byte); ok {
					m[colName] = string(b)
				} else {
					m[colName] = *val
				}
			} else {
				m[colName] = nil
			}
		}
		results = append(results, m)
	}

	res, _ := json.MarshalIndent(results, "", "  ")
	return string(res)
}
