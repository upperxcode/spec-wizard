package repository

import (
	"errors"
	"spec-wizard/pkg/model"
)

var (
	ErrNotFound = errors.New("recurso não encontrado")
	ErrInternal = errors.New("erro interno de repositório")
)

// Repository define a interface base para persistência de dados do sistema.
type Repository interface {
	HealthCheck() error
	GetTaskByID(id string) (*model.Task, error)
	SaveTask(task *model.Task) error
}
