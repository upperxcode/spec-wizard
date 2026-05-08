package adatools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_PathValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ada-test-*")
	if err != nil {
		t.Fatalf("falha ao criar pasta temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	reg := NewRegistry(tempDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Caminho relativo dentro", "file.txt", false},
		{"Subpasta", "sub/file.txt", false},
		{"Caminho absoluto dentro", filepath.Join(tempDir, "abs.txt"), false},
		{"Tentativa de sair (..)", "../outside.txt", true},
		{"Raiz do sistema", "/etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reg.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%s) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestRegistry_Execute(t *testing.T) {
	reg := NewRegistry(".")

	reg.Register("hello", "Diz olá", `{}`, func(ctx context.Context, args map[string]any) (string, error) {
		name, _ := args["name"].(string)
		if name == "" {
			name = "Mundo"
		}
		return "Olá, " + name, nil
	})

	ctx := context.Background()

	t.Run("Execução simples", func(t *testing.T) {
		res, err := reg.Execute(ctx, "hello", map[string]any{"name": "Ada"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res != "Olá, Ada" {
			t.Errorf("resultado inesperado: %s", res)
		}
	})

	t.Run("Ferramenta inexistente", func(t *testing.T) {
		_, err := reg.Execute(ctx, "missing", nil)
		if err == nil {
			t.Error("esperava erro para ferramenta inexistente")
		}
	})
}
