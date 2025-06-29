package api

import (
	"net/http"
	"strconv"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gin-gonic/gin"
)

type DataService interface {
	CreateData(data *models.Data) (int64, error)
	GetAllData(userID int64) ([]*models.Data, error)
	GetDataByID(id, userID int64) (*models.Data, error)
	UpdateData(data *models.Data) error
	DeleteData(id, userID int64) error
	SyncData(userID int64, data []*models.Data) ([]*models.Data, error)
	CreateDataWithEncrypted(data *models.Data) (int64, error)
}

func (h *Handler) createData(c *gin.Context) {
	var req models.DataRequest

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	if err := c.BindJSON(&req); err != nil {
		h.logger.Errorf("Ошибка при парсинге запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})

		return
	}

	data := &models.Data{
		UserID:   userID,
		Type:     req.Type,
		Name:     req.Name,
		Metadata: req.Metadata,
	}

	switch {
	case len(req.EncryptedData) > 0:
		data.EncryptedData = req.EncryptedData

		dataID, err := h.dataService.CreateDataWithEncrypted(data)
		if err != nil {
			h.logger.Errorf("Ошибка при создании данных: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при создании данных"})

			return
		}

		data.ID = dataID

		c.JSON(http.StatusCreated, data.ToDataResponse())
	case req.Content != nil:
		dataID, err := h.dataService.CreateData(data)
		if err != nil {
			h.logger.Errorf("Ошибка при создании данных: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при создании данных"})

			return
		}

		data.ID = dataID

		c.JSON(http.StatusCreated, data.ToDataResponse())
	default:
		h.logger.Errorf("Ошибка: отсутствуют данные для создания")
		c.JSON(http.StatusBadRequest, gin.H{"error": "отсутствуют данные для создания"})
	}
}

func (h *Handler) getAllData(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	data, err := h.dataService.GetAllData(userID)
	if err != nil {
		h.logger.Errorf("Ошибка при получении данных: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при получении данных"})

		return
	}

	response := make([]models.DataResponse, 0, len(data))
	for _, d := range data {
		response = append(response, d.ToDataResponse())
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) getData(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	dataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Errorf("Ошибка при парсинге ID данных: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID данных"})

		return
	}

	data, err := h.dataService.GetDataByID(dataID, userID)
	if err != nil {
		h.logger.Errorf("Ошибка при получении данных: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "данные не найдены"})

		return
	}

	c.JSON(http.StatusOK, data.ToDataResponse())
}

func (h *Handler) getEncryptedData(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	dataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Errorf("Ошибка при парсинге ID данных: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID данных"})

		return
	}

	data, err := h.dataService.GetDataByID(dataID, userID)
	if err != nil {
		h.logger.Errorf("Ошибка при получении данных: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "данные не найдены"})

		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) updateData(c *gin.Context) {
	var req models.DataRequest

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	dataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Errorf("Ошибка при парсинге ID данных: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID данных"})

		return
	}

	if bindErr := c.BindJSON(&req); bindErr != nil {
		h.logger.Errorf("Ошибка при парсинге запроса: %v", bindErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})

		return
	}

	_, err = h.dataService.GetDataByID(dataID, userID)
	if err != nil {
		h.logger.Errorf("Ошибка при получении данных: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "данные не найдены"})

		return
	}

	data := &models.Data{
		ID:       dataID,
		UserID:   userID,
		Type:     req.Type,
		Name:     req.Name,
		Metadata: req.Metadata,
	}

	err = h.dataService.UpdateData(data)
	if err != nil {
		h.logger.Errorf("Ошибка при обновлении данных: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при обновлении данных"})

		return
	}

	c.JSON(http.StatusOK, data.ToDataResponse())
}

func (h *Handler) deleteData(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	dataID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Errorf("Ошибка при парсинге ID данных: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID данных"})

		return
	}

	err = h.dataService.DeleteData(dataID, userID)
	if err != nil {
		h.logger.Errorf("Ошибка при удалении данных: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при удалении данных"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "данные успешно удалены"})
}

func (h *Handler) syncData(c *gin.Context) {
	var clientData []*models.Data

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})

		return
	}

	if err := c.BindJSON(&clientData); err != nil {
		h.logger.Errorf("Ошибка при парсинге запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})

		return
	}

	syncedData, err := h.dataService.SyncData(userID, clientData)
	if err != nil {
		h.logger.Errorf("Ошибка при синхронизации данных: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при синхронизации данных"})

		return
	}

	response := make([]models.DataResponse, 0, len(syncedData))
	for _, d := range syncedData {
		response = append(response, d.ToDataResponse())
	}

	c.JSON(http.StatusOK, response)
}
