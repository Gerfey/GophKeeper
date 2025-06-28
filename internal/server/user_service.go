package server

import (
	"errors"
	"time"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

var (
	ErrUserAlreadyExists = errors.New("пользователь уже существует")
	ErrUserNotFound      = errors.New("пользователь не найден")
	ErrInvalidPassword   = errors.New("неверный пароль")
)

type UserRepository interface {
	Create(user *models.User) (int64, error)
	GetByUsername(username string) (*models.User, error)
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

// CreateUser создает нового пользователя
func (s *UserService) CreateUser(user *models.User) (int64, error) {
	_, err := s.repo.GetByUsername(user.Username)
	if err == nil {
		return 0, ErrUserAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return 0, err
	}

	hashedPassword, err := crypto.HashPassword(user.Password)
	if err != nil {
		return 0, err
	}

	user.Password = hashedPassword

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	return s.repo.Create(user)
}

// GetUserByUsername возвращает пользователя по имени пользователя
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	return s.repo.GetByUsername(username)
}

// VerifyUser проверяет учетные данные пользователя
func (s *UserService) VerifyUser(creds *models.UserCredentials) (*models.User, error) {
	user, err := s.repo.GetByUsername(creds.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	valid, err := crypto.VerifyPassword(creds.Password, user.Password)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, ErrInvalidPassword
	}

	return user, nil
}
