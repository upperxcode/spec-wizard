package model

import "time"

// Task representa uma unidade de trabalho no Spec Wizard.
// Este modelo é exportado para ser usado por diferentes camadas.
type Task struct {
	ID                 string    `json:"id" validate:"required"`
	Title              string    `json:"title" validate:"required"`
	Description        string    `json:"description" validate:"required"`
	Status             string    `json:"status" validate:"required"`
	Priority           string    `json:"priority,omitempty"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
