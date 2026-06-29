package governance

import (
	"encoding/json"
	"testing"
)

func TestMigrateUnmarshalResilience(t *testing.T) {
	jsonData := `{
		"name": "Test Project",
		"sprints": [
			{
				"id": "s1",
				"tasks": [
					{
						"id": "t1",
						"files": ["file1.go", "file2.go"],
						"test_files": ["test1.go"]
					},
					{
						"id": "t2",
						"files": [
							{"path": "file3.go", "description": "desc3", "action": "create"}
						]
					}
				]
			}
		]
	}`

	var roadmap struct {
		Sprints []struct {
			Tasks []struct {
				ID        string      `json:"id"`
				Files     interface{} `json:"files"`
				TestFiles interface{} `json:"test_files"`
			} `json:"tasks"`
		} `json:"sprints"`
	}

	if err := json.Unmarshal([]byte(jsonData), &roadmap); err != nil {
		t.Fatalf("Unmarshal falhou: %v", err)
	}

	if len(roadmap.Sprints) != 1 {
		t.Fatalf("Esperava 1 sprint, obteve %d", len(roadmap.Sprints))
	}

	task1 := roadmap.Sprints[0].Tasks[0]
	if _, ok := task1.Files.([]interface{}); !ok {
		t.Errorf("Task 1 Files deveria ser []interface{} (array de strings no JSON)")
	}

	task2 := roadmap.Sprints[0].Tasks[1]
	if _, ok := task2.Files.([]interface{}); !ok {
		t.Errorf("Task 2 Files deveria ser []interface{} (array de objetos no JSON)")
	}

	t.Log("✅ Unmarshal resiliente validado com sucesso!")
}
