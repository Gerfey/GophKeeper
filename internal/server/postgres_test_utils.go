package server

import (
	"database/sql"

	"github.com/gerfey/gophkeeper/pkg/logger"
)

type PostgresRepositoryTest struct {
	*PostgresRepository
}

func NewPostgresRepositoryTest(db *sql.DB, logger logger.Logger) *PostgresRepository {
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}
