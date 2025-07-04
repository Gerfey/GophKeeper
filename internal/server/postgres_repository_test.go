package server_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/internal/server"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

func TestRepo_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	now := time.Now()
	user := &models.User{
		Username:  "testuser",
		Password:  "hashedpassword",
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectQuery("INSERT INTO users").
		WithArgs(user.Username, user.Password, user.CreatedAt, user.UpdatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	id, err := repo.CreateUser(t.Context(), user)
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestRepo_GetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	username := "testuser"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "created_at", "updated_at"}).
		AddRow(1, username, "hashedpassword", now, now)

	mock.ExpectQuery("SELECT (.+) FROM users").
		WithArgs(username).
		WillReturnRows(rows)

	user, err := repo.GetByUsername(t.Context(), username)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "hashedpassword", user.Password)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestRepo_CreateData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	now := time.Now()
	data := &models.Data{
		UserID:        1,
		Type:          models.LoginPassword,
		Name:          "test",
		EncryptedData: []byte("encrypted"),
		Metadata:      "meta",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mock.ExpectQuery(`INSERT INTO data`).
		WithArgs(data.UserID, data.Type, data.Name, data.EncryptedData, data.Metadata, data.CreatedAt, data.UpdatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	id, err := repo.CreateData(t.Context(), data)
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestRepo_GetAllData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "data_type", "name", "encrypted_data", "metadata", "created_at", "updated_at"}).
		AddRow(1, 1, models.LoginPassword, "test", []byte("encrypted"), "meta", now, now)

	mock.ExpectQuery(`SELECT id, user_id, data_type, name, encrypted_data, metadata, created_at, updated_at FROM data`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	data, err := repo.GetAll(t.Context(), 1)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, int64(1), data[0].ID)
	assert.Equal(t, int64(1), data[0].UserID)
	assert.Equal(t, models.LoginPassword, data[0].Type)
	assert.Equal(t, "test", data[0].Name)
	assert.Equal(t, []byte("encrypted"), data[0].EncryptedData)
	assert.Equal(t, "meta", data[0].Metadata)
}

func TestRepo_GetDataByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "data_type", "name", "encrypted_data", "metadata", "created_at", "updated_at"}).
		AddRow(1, 1, models.LoginPassword, "test", []byte("encrypted"), "meta", now, now)

	mock.ExpectQuery(`SELECT id, user_id, data_type, name, encrypted_data, metadata, created_at, updated_at FROM data`).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(rows)

	data, err := repo.GetByID(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), data.ID)
	assert.Equal(t, int64(1), data.UserID)
	assert.Equal(t, models.LoginPassword, data.Type)
	assert.Equal(t, "test", data.Name)
	assert.Equal(t, []byte("encrypted"), data.EncryptedData)
	assert.Equal(t, "meta", data.Metadata)
}

func TestRepo_UpdateData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	now := time.Now()
	data := &models.Data{
		ID:            1,
		UserID:        1,
		Type:          models.LoginPassword,
		Name:          "test",
		EncryptedData: []byte("encrypted"),
		Metadata:      "meta",
		UpdatedAt:     now,
	}

	mock.ExpectExec(`UPDATE data`).
		WithArgs(data.Type, data.Name, data.EncryptedData, data.Metadata, sqlmock.AnyArg(), data.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(t.Context(), data)
	require.NoError(t, err)
}

func TestRepo_DeleteData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLogger(ctrl)
	repo := server.NewPostgresRepositoryTest(db, mockLogger)

	dataID := int64(1)
	userID := int64(1)

	mock.ExpectExec("DELETE FROM data").
		WithArgs(dataID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(t.Context(), dataID, userID)
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}
