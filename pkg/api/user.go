package api

import (
	"net/http"
	"time"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gin-gonic/gin"
)

type UserService interface {
	CreateUser(user *models.User) (int64, error)
	GetUserByUsername(username string) (*models.User, error)
}

// register обрабатывает запрос на регистрацию пользователя
func (h *Handler) register(c *gin.Context) {
	var user models.User

	if err := c.BindJSON(&user); err != nil {
		h.logger.Error("Ошибка при парсинге запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})
		return
	}

	userID, err := h.userService.CreateUser(&user)
	if err != nil {
		h.logger.Error("Ошибка при создании пользователя: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при создании пользователя"})
		return
	}

	token, err := h.tokenManager.GenerateToken(userID, 24*time.Hour)
	if err != nil {
		h.logger.Error("Ошибка при генерации токена: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при генерации токена"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  user.ToUserResponse(),
		"token": token,
	})
}

// login обрабатывает запрос на аутентификацию пользователя
func (h *Handler) login(c *gin.Context) {
	var creds models.UserCredentials

	if err := c.BindJSON(&creds); err != nil {
		h.logger.Error("Ошибка при парсинге запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})
		return
	}

	user, err := h.userService.GetUserByUsername(creds.Username)
	if err != nil {
		h.logger.Error("Ошибка при получении пользователя: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "неверное имя пользователя или пароль"})
		return
	}

	token, err := h.tokenManager.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		h.logger.Error("Ошибка при генерации токена: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user.ToUserResponse(),
		"token": token,
	})
}
