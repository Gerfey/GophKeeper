package api

import (
	"net/http"
	"time"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gin-gonic/gin"
)

const (
	DefaultTokenDuration = 24 * time.Hour
)

type UserService interface {
	CreateUser(username string, password string) (int64, error)
	GetUserByUsername(username string) (*models.User, error)
	CheckCredentials(username string, password string) (*models.User, error)
}

func (h *Handler) register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	userID, err := h.userService.CreateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	token, err := h.tokenManager.GenerateToken(userID, DefaultTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

func (h *Handler) login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	user, err := h.userService.CheckCredentials(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "неверное имя пользователя или пароль"})

		return
	}

	token, err := h.tokenManager.GenerateToken(user.ID, DefaultTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
