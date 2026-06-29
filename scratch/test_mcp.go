package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func main() {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "wz_init",
			"arguments": map[string]interface{}{
				"language": "pt-BR",
				"name":     "Test Project",
				"stack":    "Flutter",
				"path":     "/tmp/test-wizard-final",
			},
		},
	}

	input, _ := json.Marshal(req)
	
	cmd := exec.Command("/home/john/.local/bin/mcp-wizard")
	cmd.Stdin = bytes.NewReader(input)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
	}
	
	fmt.Printf("Output: %s\n", string(output))
}
