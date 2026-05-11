package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Option struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type StackTemplate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Libraries []Library `json:"libraries"`
}

type Library struct {
	Name         string `json:"name"`
	Mandatory    bool   `json:"mandatory"`
	UsageExample string `json:"usage_example"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	mux.HandleFunc("/options", handleOptions)
	mux.HandleFunc("/AnalyzeCodebase", handleAnalyze)
	mux.HandleFunc("/GetPatterns", handleGetPatterns)

	port := "8081"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	fmt.Printf("🚀 Flutter Expert (Go) ativo na porta %s\n", port)
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor: %v\n", err)
	}
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	options := map[string]interface{}{
		"architectures": []Option{
			{ID: "clean_architecture", Name: "Clean Architecture", Description: "Domain, Data, Presentation layers"},
			{ID: "bloc", Name: "BLoC Architecture", Description: "Event-based state management"},
			{ID: "mvvm", Name: "MVVM", Description: "Model-View-ViewModel"},
		},
		"state_managements": []Option{
			{ID: "bloc", Name: "BLoC/Cubit"},
			{ID: "riverpod", Name: "Riverpod"},
			{ID: "provider", Name: "Provider"},
			{ID: "getx", Name: "GetX"},
		},
		"stack_templates": []StackTemplate{
			{
				ID:   "flutter_standard",
				Name: "Flutter (Standard - Dio/GetIt/Bloc)",
				Libraries: []Library{
					{Name: "dio", Mandatory: true, UsageExample: "final dio = Dio();\nawait dio.get('/api');"},
					{Name: "get_it", Mandatory: true, UsageExample: "GetIt.I.registerSingleton(Service());"},
					{Name: "flutter_bloc", Mandatory: true, UsageExample: "BlocProvider(create: (_) => MyBloc());"},
				},
			},
			{
				ID:   "flutter_enterprise",
				Name: "Flutter (Enterprise - Riverpod/Freezed)",
				Libraries: []Library{
					{Name: "flutter_riverpod", Mandatory: true, UsageExample: "final provider = StateProvider((ref) => 0);"},
					{Name: "freezed_annotation", Mandatory: true, UsageExample: "@freezed class User with _$User {...}"},
					{Name: "dio", Mandatory: true, UsageExample: "final response = await dio.get('/endpoint');"},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	patterns := map[string]interface{}{
		"patterns": []map[string]string{
			{"id": "bloc", "name": "BLoC", "category": "State Management"},
			{"id": "repository_pattern", "name": "Repository", "category": "Data"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var envelope struct {
		Data struct {
			ProjectPath string `json:"project_path"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	path := envelope.Data.ProjectPath
	pubspecPath := filepath.Join(path, "pubspec.yaml")
	
	projectName := filepath.Base(path)
	detectedLibs := []string{}

	if _, err := os.Stat(pubspecPath); err == nil {
		data, _ := os.ReadFile(pubspecPath)
		var pubspec struct {
			Name         string                 `yaml:"name"`
			Dependencies map[string]interface{} `yaml:"dependencies"`
		}
		if err := yaml.Unmarshal(data, &pubspec); err == nil {
			if pubspec.Name != "" {
				projectName = pubspec.Name
			}
			for lib := range pubspec.Dependencies {
				detectedLibs = append(detectedLibs, lib)
			}
		}
	}

	resp := map[string]interface{}{
		"project_name": projectName,
		"dependencies": detectedLibs,
		"project_type": "Flutter App",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
