package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type AnalysisRequest struct {
	Data string `json:"data"` // Conteúdo do go.mod
}

type AnalysisResponse struct {
	ProjectName  string   `json:"project_name"`
	DetectedLibs []string `json:"detected_libs"`
	ProjectType  string   `json:"project_type"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	mux.HandleFunc("/options", handleOptions)
	mux.HandleFunc("/AnalyzeCodebase", handleAnalyze)
	mux.HandleFunc("/Format", handleFormat)

	port := "8083"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	fmt.Printf("🚀 Go Expert ativo na porta %s\n", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor: %v\n", err)
	}
}

type FormatRequest struct {
	Code string `json:"code"`
}

type FormatResponse struct {
	FormattedCode string `json:"formatted_code"`
	Error         string `json:"error,omitempty"`
}

func handleFormat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req FormatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("gofmt")
	cmd.Stdin = strings.NewReader(req.Code)
	output, err := cmd.CombinedOutput()

	resp := FormatResponse{
		FormattedCode: string(output),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	options := map[string]interface{}{
		"architectures": []map[string]interface{}{
			{"id": "clean_arch", "name": "Clean Architecture", "description": "Uso de camadas: cmd, internal/domain, internal/platform"},
			{"id": "flat_arch", "name": "Flat (Standard)", "description": "Pacote único ou estruturado por funcionalidade na raiz"},
		},
		"stack_templates": []map[string]interface{}{
			{
				"id":   "go_backend",
				"name": "Go Backend (Gin/Sqlx)",
				"libraries": []map[string]interface{}{
					{"name": "github.com/gin-gonic/gin", "mandatory": true, "usage_example": "r := gin.Default()\nr.GET(\"/api/v1/resource\", handler)\nr.Run()"},
					{"name": "github.com/jmoiron/sqlx", "mandatory": true, "usage_example": "var items []Item\nerr := db.Select(&items, \"SELECT * FROM items WHERE status = ?\", \"active\")"},
				},
			},
			{
				"id":   "go_cli_service",
				"name": "Go CLI Service (Cobra/Zap/Viper)",
				"libraries": []map[string]interface{}{
					{"name": "github.com/spf13/cobra", "mandatory": true, "usage_example": "rootCmd.Execute()"},
					{"name": "go.uber.org/zap", "mandatory": true, "usage_example": "logger, _ := zap.NewProduction()\nlogger.Info(\"serviço iniciado\")"},
					{"name": "github.com/spf13/viper", "mandatory": true, "usage_example": "viper.ReadInConfig()"},
					{"name": "github.com/stretchr/testify/assert", "mandatory": false, "usage_example": "assert.NoError(t, err)"},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(options); err != nil {
		fmt.Printf("❌ Erro ao codificar opções: %v\n", err)
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var envelope struct {
		Action string `json:"action"`
		Data   struct {
			Data        string `json:"data"`
			ProjectPath string `json:"project_path"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, "Erro ao decodificar JSON (Envelope)", http.StatusBadRequest)
		return
	}

	goModData := envelope.Data.Data
	projectPath := envelope.Data.ProjectPath

	if projectPath != "" && goModData == "" {
		content, err := os.ReadFile(projectPath + "/go.mod")
		if err == nil {
			goModData = string(content)
		}
	}

	projectName := "Unknown Go Project"
	libs := []string{}
	projectType := "Web Service / CLI"

	lines := strings.Split(goModData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			projectName = strings.TrimPrefix(line, "module ")
		}
		if strings.Contains(line, "github.com/") {
			parts := strings.Split(line, " ")
			if len(parts) > 0 {
				libs = append(libs, parts[0])
			}
		}
	}

	resp := AnalysisResponse{
		ProjectName:  projectName,
		DetectedLibs: libs,
		ProjectType:  projectType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
