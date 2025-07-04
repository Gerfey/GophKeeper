package server

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

const (
	pingTimeout   = 5 * time.Second
	schemaTimeout = 10 * time.Second
)

var (
	ErrUserNotFound = errors.New("пользователь не найден")
	ErrDataNotFound = errors.New("данные не найдены")
)

type PostgresRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewPostgresRepository(dsn string, logger logger.Logger) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if pingErr := db.PingContext(ctx); pingErr != nil {
		return nil, pingErr
	}

	return &PostgresRepository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) InitSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), schemaTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS data (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			data_type VARCHAR(20) NOT NULL,
			name VARCHAR(100) NOT NULL,
			encrypted_data BYTEA NOT NULL,
			metadata TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)

	return err
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.Username, user.Password, user.CreatedAt, user.UpdatedAt).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) CreateData(ctx context.Context, data *models.Data) (int64, error) {
	query := `
		INSERT INTO data (user_id, data_type, name, encrypted_data, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query, data.UserID, data.Type, data.Name, data.EncryptedData, data.Metadata, data.CreatedAt, data.UpdatedAt).
		Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *PostgresRepository) GetAll(ctx context.Context, userID int64) ([]*models.Data, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, data_type, name, encrypted_data, metadata, created_at, updated_at
		FROM data
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Data

	for rows.Next() {
		var data models.Data
		scanErr := rows.Scan(
			&data.ID,
			&data.UserID,
			&data.Type,
			&data.Name,
			&data.EncryptedData,
			&data.Metadata,
			&data.CreatedAt,
			&data.UpdatedAt,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, &data)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}

	return result, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, userID int64) (*models.Data, error) {
	query := `
		SELECT id, user_id, data_type, name, encrypted_data, metadata, created_at, updated_at
		FROM data
		WHERE id = $1 AND user_id = $2
	`

	var data models.Data
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&data.ID,
		&data.UserID,
		&data.Type,
		&data.Name,
		&data.EncryptedData,
		&data.Metadata,
		&data.CreatedAt,
		&data.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDataNotFound
		}

		return nil, err
	}

	return &data, nil
}

func (r *PostgresRepository) Update(ctx context.Context, data *models.Data) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE data
		SET data_type = $1, name = $2, encrypted_data = $3, metadata = $4, updated_at = $5
		WHERE id = $6
	`, data.Type, data.Name, data.EncryptedData, data.Metadata, time.Now(), data.ID)

	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id, userID int64) error {
	query := `DELETE FROM data WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrDataNotFound
	}

	return nil
}
