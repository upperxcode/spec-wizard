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
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	http.HandleFunc("/AnalyzeCodebase", handleAnalyze)
	http.HandleFunc("/Format", handleFormat)
	
	port := "8083"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	
	fmt.Printf("🚀 Go Expert ativo na porta %s\n", port)
	http.ListenAndServe(":"+port, nil)
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

	// Tenta UCF primeiro (Universal Code Formatter) por ser mais rápido
	cmd := exec.Command("ucf")
	cmd.Stdin = strings.NewReader(req.Code)
	output, err := cmd.CombinedOutput()
	
	// Fallback para gofmt se ucf falhar ou não estiver instalado
	if err != nil {
		cmd = exec.Command("gofmt")
		cmd.Stdin = strings.NewReader(req.Code)
		output, err = cmd.CombinedOutput()
	}
	
	resp := FormatResponse{
		FormattedCode: string(output),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Envelope padrão do Servidor
	var envelope struct {
		Action string `json:"action"`
		Data   struct {
			Data        string `json:"data"`
			ProjectPath string `json:"project_path"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		// Fallback para decodificação direta se não houver envelope
		r.Body = os.Stdin // dummy reset (não funciona bem com decoder)
		http.Error(w, "Erro ao decodificar JSON (Envelope)", http.StatusBadRequest)
		return
	}

	goModData := envelope.Data.Data
	projectPath := envelope.Data.ProjectPath

	// Se recebemos o path, tentamos ler o go.mod localmente
	if projectPath != "" && goModData == "" {
		// Garante path absoluto se necessário
		content, err := os.ReadFile(projectPath + "/go.mod")
		if err == nil {
			goModData = string(content)
		}
	}

	// Análise simplificada do go.mod
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
