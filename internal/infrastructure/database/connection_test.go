package database

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestCheckHealth(t *testing.T) {
	// Mock do banco de dados com monitoramento de pings habilitado
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("erro ao criar mock: %s", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Cenário 1: Sucesso
	mock.ExpectPing()
	err = CheckHealth(sqlxDB)
	assert.NoError(t, err)

	// Cenário 2: Erro no Ping
	mock.ExpectPing().WillReturnError(fmt.Errorf("db offline"))
	err = CheckHealth(sqlxDB)
	assert.Error(t, err)
}
