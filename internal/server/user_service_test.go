package server_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/internal/server"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

func TestUserService_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockUserRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewUserService(mockRepo, mockLogger)

	username := "testuser"
	password := "password123"

	mockRepo.EXPECT().
		GetByUsername(gomock.Any(), username).
		Return(nil, server.ErrUserNotFound)

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, u *models.User) (int64, error) {
			assert.Equal(t, username, u.Username)
			assert.NotEqual(t, password, u.Password)

			return 1, nil
		})

	id, err := service.CreateUser(t.Context(), username, password)

	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestUserService_CreateUser_UserExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockUserRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewUserService(mockRepo, mockLogger)

	username := "testuser"
	password := "password123"

	existingUser := &models.User{
		ID:       1,
		Username: username,
		Password: "hashedpassword",
	}

	mockRepo.EXPECT().
		GetByUsername(gomock.Any(), username).
		Return(existingUser, nil)

	_, err := service.CreateUser(t.Context(), username, password)

	assert.ErrorIs(t, err, server.ErrUserAlreadyExists)
}

func TestUserService_CheckCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockUserRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewUserService(mockRepo, mockLogger)

	username := "testuser"
	password := "password123"

	hashedPassword, err := crypto.HashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		ID:       1,
		Username: username,
		Password: hashedPassword,
	}

	mockRepo.EXPECT().
		GetByUsername(gomock.Any(), username).
		Return(user, nil)

	resultUser, err := service.CheckCredentials(t.Context(), username, password)

	require.NoError(t, err)
	assert.Equal(t, user, resultUser)
}

func TestUserService_CheckCredentials_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockUserRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewUserService(mockRepo, mockLogger)

	username := "testuser"
	password := "password123"

	mockRepo.EXPECT().
		GetByUsername(gomock.Any(), username).
		Return(nil, server.ErrUserNotFound)

	_, err := service.CheckCredentials(t.Context(), username, password)

	assert.ErrorIs(t, err, server.ErrUserNotFound)
}

func TestUserService_CheckCredentials_InvalidPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockUserRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewUserService(mockRepo, mockLogger)

	username := "testuser"
	password := "password123"
	wrongPassword := "wrongpassword"

	hashedPassword, err := crypto.HashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		ID:       1,
		Username: username,
		Password: hashedPassword,
	}

	mockRepo.EXPECT().
		GetByUsername(gomock.Any(), username).
		Return(user, nil)

	_, err = service.CheckCredentials(t.Context(), username, wrongPassword)

	assert.ErrorIs(t, err, server.ErrInvalidPassword)
}
