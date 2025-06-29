package api

import (
	"net/http"
	"strings"

	"github.com/gerfey/gophkeeper/internal/auth"
	"github.com/gerfey/gophkeeper/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	tokenManager auth.TokenManager
	userService  UserService
	dataService  DataService
	logger       logger.Logger
}

func NewHandler(
	tokenManager auth.TokenManager,
	userService UserService,
	dataService DataService,
	logger logger.Logger,
) *Handler {
	return &Handler{
		tokenManager: tokenManager,
		userService:  userService,
		dataService:  dataService,
		logger:       logger,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(h.loggerMiddleware())

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.register)
			auth.POST("/login", h.login)
		}

		data := api.Group("/data", h.authMiddleware())
		{
			data.POST("/", h.createData)
			data.GET("/", h.getAllData)
			data.GET("/:id", h.getData)
			data.GET("/:id/encrypted", h.getEncryptedData)
			data.PUT("/:id", h.updateData)
			data.DELETE("/:id", h.deleteData)
		}

		sync := api.Group("/sync", h.authMiddleware())
		{
			sync.POST("/", h.syncData)
		}
	}

	return router
}

func (h *Handler) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.logger.Infof("Request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		h.logger.Infof("Response: %d", c.Writer.Status())
	}
}

func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "отсутствует токен авторизации"})

			return
		}

		const bearerPrefix = "Bearer "
		if len(header) > len(bearerPrefix) && strings.HasPrefix(header, bearerPrefix) {
			header = header[len(bearerPrefix):]
		}

		claims, err := h.tokenManager.ValidateToken(header)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func getUserID(c *gin.Context) (int64, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}

	id, ok := userID.(int64)

	return id, ok
}
