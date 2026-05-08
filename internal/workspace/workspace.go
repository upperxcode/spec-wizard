package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type ProjectEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Workspace struct {
	Projects []ProjectEntry `json:"projects"`
	filePath string
	mu       sync.Mutex
}

func NewWorkspace(filePath string) *Workspace {
	w := &Workspace{
		filePath: filePath,
	}
	w.Load()
	return w
}

func (w *Workspace) Load() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := os.ReadFile(w.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.Projects = []ProjectEntry{}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &w.Projects)
}

func (w *Workspace) Save() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ordenar por nome antes de salvar
	sort.Slice(w.Projects, func(i, j int) bool {
		return w.Projects[i].Name < w.Projects[j].Name
	})

	data, err := json.MarshalIndent(w.Projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(w.filePath, data, 0644)
}

func (w *Workspace) AddProject(name, path string) error {
	// Validação: verificar se .spec-wizard/config.json existe no diretório
	configPath := filepath.Join(path, ".spec-wizard", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("projeto inválido: %s não encontrado", configPath)
	}

	w.mu.Lock()
	// Evitar duplicatas por path
	for i, p := range w.Projects {
		if p.Path == path {
			w.Projects[i].Name = name // Atualiza nome se o path já existe
			w.mu.Unlock()
			return w.Save()
		}
	}
	w.Projects = append(w.Projects, ProjectEntry{Name: name, Path: path})
	w.mu.Unlock()

	return w.Save()
}

func (w *Workspace) RemoveProject(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	newProjects := []ProjectEntry{}
	for _, p := range w.Projects {
		if p.Path != path {
			newProjects = append(newProjects, p)
		}
	}
	w.Projects = newProjects
	return w.Save()
}
