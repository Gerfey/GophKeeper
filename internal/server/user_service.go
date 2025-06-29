package server

import (
	"context"
	"errors"
	"time"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

var (
	ErrUserAlreadyExists = errors.New("пользователь уже существует")
	ErrInvalidPassword   = errors.New("неверный пароль")
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (int64, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
}

type UserService struct {
	repo   UserRepository
	logger logger.Logger
}

func NewUserService(repo UserRepository, logger logger.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, username string, password string) (int64, error) {
	existingUser, err := s.repo.GetByUsername(ctx, username)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return 0, err
	}
	if existingUser != nil {
		return 0, ErrUserAlreadyExists
	}

	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	user := &models.User{
		Username:  username,
		Password:  hashedPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return s.repo.CreateUser(ctx, user)
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *UserService) VerifyUser(ctx context.Context, creds *models.UserCredentials) (*models.User, error) {
	user, err := s.repo.GetByUsername(ctx, creds.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidPassword
		}

		return nil, err
	}

	valid := crypto.VerifyPassword(creds.Password, user.Password)
	if !valid {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

func (s *UserService) CheckCredentials(ctx context.Context, username string, password string) (*models.User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	valid := crypto.VerifyPassword(password, user.Password)
	if !valid {
		return nil, ErrInvalidPassword
	}

	return user, nil
}
