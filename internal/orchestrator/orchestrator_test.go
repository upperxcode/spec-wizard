package orchestrator

import (
	"strings"
	"testing"
)

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Markdown block",
			input:    "```json\n{\"id\": 1}\n```",
			expected: "{\"id\": 1}",
		},
		{
			name:     "Markdown block with trailing text",
			input:    "```json\n{\"id\": 1}\n```Some other text",
			expected: "{\"id\": 1}",
		},
		{
			name:     "Pure JSON",
			input:    "{\"id\": 1}",
			expected: "{\"id\": 1}",
		},
		{
			name:     "Empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanJSONResponse(tt.input)
			if got != tt.expected {
				t.Errorf("CleanJSONResponse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"custom string", "custom", true},
		{"unidentified domain", "Domínio não identificado", true},
		{"valid string", "flutter", false},
		{"empty slice", []string{}, true},
		{"valid slice", []string{"a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmpty(tt.input); got != tt.want {
				t.Errorf("isEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateFileTree(t *testing.T) {
	// Mock orchestrator
	o := &Orchestrator{
		ProjectPath: ".", // Use current directory for test
	}

	// Test without exclusions
	tree := o.generateFileTree(nil)
	if tree == "" {
		t.Error("generateFileTree() returned empty string")
	}

	// Check if it contains some expected elements
	if !strings.Contains(tree, "types.go") {
		t.Error("generateFileTree() output missing types.go")
	}

	// Test with exclusions
	treeExclude := o.generateFileTree([]string{"types.go"})
	if strings.Contains(treeExclude, "types.go") {
		t.Error("generateFileTree() should have excluded types.go")
	}
}
