package main

import (
	"fmt"
	"os"
)

func main() {
	val := os.Getenv("OPENROUTER_API_KEY")
	fmt.Printf("OPENROUTER_API_KEY = '%s', len = %d\n", val, len(val))
}
