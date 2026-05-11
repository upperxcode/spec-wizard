package model

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestTaskValidation(t *testing.T) {
	validate := validator.New()

	// Cenário: Task válida
	validTask := Task{
		ID:          "1",
		Title:       "Test Task",
		Description: "A valid description",
		Status:      "pending",
	}
	err := validate.Struct(validTask)
	assert.NoError(t, err)

	// Cenário: Task inválida (faltando Title)
	invalidTask := Task{
		ID:          "1",
		Description: "Missing title",
		Status:      "pending",
	}
	err = validate.Struct(invalidTask)
	assert.Error(t, err)
}
