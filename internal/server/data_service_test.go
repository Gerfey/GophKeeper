package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/internal/server"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

func TestDataService_CreateData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	data := &models.Data{
		UserID: 1,
		Type:   models.TextData,
		Name:   "Test Data",
	}

	mockRepo.EXPECT().
		CreateData(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) (int64, error) {
			assert.Equal(t, data.UserID, d.UserID)
			assert.Equal(t, data.Type, d.Type)
			assert.Equal(t, data.Name, d.Name)
			assert.False(t, d.CreatedAt.IsZero())
			assert.False(t, d.UpdatedAt.IsZero())

			return 1, nil
		})

	id, err := service.CreateData(t.Context(), data)

	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestDataService_GetDataByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	dataID := int64(1)
	userID := int64(1)

	expectedData := &models.Data{
		ID:        dataID,
		UserID:    userID,
		Type:      models.TextData,
		Name:      "Test Data",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetByID(gomock.Any(), dataID, userID).
		Return(expectedData, nil)

	data, err := service.GetDataByID(t.Context(), dataID, userID)

	require.NoError(t, err)
	assert.Equal(t, expectedData, data)
}

func TestDataService_GetAllData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	userID := int64(1)

	expectedData := []*models.Data{
		{
			ID:        1,
			UserID:    userID,
			Type:      models.TextData,
			Name:      "Test Data 1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			UserID:    userID,
			Type:      models.BinaryData,
			Name:      "Test Data 2",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo.EXPECT().
		GetAll(gomock.Any(), userID).
		Return(expectedData, nil)

	data, err := service.GetAllData(t.Context(), userID)

	require.NoError(t, err)
	assert.Equal(t, expectedData, data)
}

func TestDataService_UpdateData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	data := &models.Data{
		ID:        1,
		UserID:    1,
		Type:      models.TextData,
		Name:      "Updated Data",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetByID(gomock.Any(), data.ID, data.UserID).
		Return(&models.Data{
			ID:        data.ID,
			UserID:    data.UserID,
			Type:      models.TextData,
			Name:      "Original Data",
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		}, nil)

	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) error {
			assert.Equal(t, data.ID, d.ID)
			assert.Equal(t, data.UserID, d.UserID)
			assert.Equal(t, data.Type, d.Type)
			assert.Equal(t, data.Name, d.Name)

			return nil
		})

	err := service.UpdateData(t.Context(), data)

	require.NoError(t, err)
}

func TestDataService_DeleteData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	dataID := int64(1)
	userID := int64(1)

	mockRepo.EXPECT().
		GetByID(gomock.Any(), dataID, userID).
		Return(&models.Data{
			ID:     dataID,
			UserID: userID,
			Type:   models.TextData,
			Name:   "Test Data",
		}, nil)

	mockRepo.EXPECT().
		Delete(gomock.Any(), dataID, userID).
		Return(nil)

	err := service.DeleteData(t.Context(), dataID, userID)

	require.NoError(t, err)
}

func TestDataService_CreateDataWithEncrypted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	data := &models.Data{
		UserID:        1,
		Type:          models.TextData,
		Name:          "Encrypted Data",
		EncryptedData: []byte("already-encrypted-data"),
	}

	mockRepo.EXPECT().
		CreateData(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) (int64, error) {
			assert.Equal(t, data.UserID, d.UserID)
			assert.Equal(t, data.Type, d.Type)
			assert.Equal(t, data.Name, d.Name)
			assert.Equal(t, data.EncryptedData, d.EncryptedData)
			assert.False(t, d.CreatedAt.IsZero())
			assert.False(t, d.UpdatedAt.IsZero())

			return 1, nil
		})

	id, err := service.CreateDataWithEncrypted(t.Context(), data)
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)

	emptyData := &models.Data{
		UserID:        1,
		Type:          models.TextData,
		Name:          "Empty Data",
		EncryptedData: nil,
	}

	id, err = service.CreateDataWithEncrypted(t.Context(), emptyData)
	require.Error(t, err)
	assert.Equal(t, server.ErrEncryptionFailed, err)
	assert.Equal(t, int64(0), id)
}

func TestDataService_PrepareData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	data1 := &models.Data{
		Type:          models.TextData,
		EncryptedData: []byte("already-encrypted"),
	}

	err := service.PrepareDataTest(data1)
	require.NoError(t, err)
	assert.Equal(t, []byte("already-encrypted"), data1.EncryptedData)

	loginData := &models.LoginPasswordData{
		Login:    "user",
		Password: "pass",
	}

	data2 := &models.Data{
		Type:    models.LoginPassword,
		Content: loginData,
	}

	err = service.PrepareDataTest(data2)
	require.NoError(t, err)
	assert.NotNil(t, data2.EncryptedData)
	assert.Nil(t, data2.Content)

	textContent := &models.TextDataContent{
		Content: "secret text",
	}

	data3 := &models.Data{
		Type:    models.TextData,
		Content: textContent,
	}

	err = service.PrepareDataTest(data3)
	require.NoError(t, err)
	assert.NotNil(t, data3.EncryptedData)
	assert.Nil(t, data3.Content)

	binaryContent := []byte("binary data")

	data4 := &models.Data{
		Type:    models.BinaryData,
		Content: binaryContent,
	}

	err = service.PrepareDataTest(data4)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, data4.EncryptedData)
	assert.Nil(t, data4.Content)

	data5 := &models.Data{
		Type:    models.BinaryData,
		Content: "not binary",
	}

	err = service.PrepareDataTest(data5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "неверный формат бинарных данных")

	data6 := &models.Data{
		Type:    "unknown",
		Content: "content",
	}

	err = service.PrepareDataTest(data6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "неизвестный тип данных")
}

func TestDataService_SyncData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := server.NewMockDataRepository(ctrl)
	mockLogger := logger.NewMockLogger(ctrl)

	service := server.NewDataService(mockRepo, mockLogger)

	userID := int64(1)

	serverData := []*models.Data{
		{
			ID:            1,
			UserID:        userID,
			Type:          models.TextData,
			Name:          "Server Data 1",
			EncryptedData: []byte("server-data-1"),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            2,
			UserID:        userID,
			Type:          models.LoginPassword,
			Name:          "Server Data 2",
			EncryptedData: []byte("server-data-2"),
			UpdatedAt:     time.Now().Add(-time.Hour),
		},
		{
			ID:            3,
			UserID:        userID,
			Type:          models.BinaryData,
			Name:          "Server Data 3",
			EncryptedData: []byte("server-data-3"),
			UpdatedAt:     time.Now(),
		},
	}

	clientData := []*models.Data{
		{
			ID:            2,
			UserID:        userID,
			Type:          models.LoginPassword,
			Name:          "Updated Client Data 2",
			EncryptedData: []byte("updated-client-data-2"),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            0,
			UserID:        userID,
			Type:          models.CardData,
			Name:          "New Client Data",
			EncryptedData: []byte("new-client-data"),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            4,
			UserID:        userID,
			Type:          models.TextData,
			Name:          "Missing Server Data",
			EncryptedData: []byte("missing-server-data"),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            -1,
			UserID:        userID,
			Type:          models.TextData,
			Name:          "Client Only Data",
			EncryptedData: []byte("client-only-data"),
			UpdatedAt:     time.Now(),
		},
	}

	mockRepo.EXPECT().
		GetAll(gomock.Any(), userID).
		Return(serverData, nil)

	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) error {
			assert.Equal(t, int64(2), d.ID)
			assert.Equal(t, "Updated Client Data 2", d.Name)

			return nil
		})

	mockRepo.EXPECT().
		CreateData(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) (int64, error) {
			assert.Equal(t, int64(0), d.ID)
			assert.Equal(t, "New Client Data", d.Name)

			return 5, nil
		})

	mockRepo.EXPECT().
		CreateData(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d *models.Data) (int64, error) {
			assert.Equal(t, int64(0), d.ID)
			assert.Equal(t, "Missing Server Data", d.Name)

			return 6, nil
		})

	result, err := service.SyncData(t.Context(), userID, clientData)

	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Len(t, result, 6)

	foundServer1 := false
	foundServer3 := false
	foundUpdated2 := false
	foundNew5 := false
	foundNew6 := false
	foundClientOnly := false

	for _, d := range result {
		switch d.ID {
		case 1:
			foundServer1 = true
			assert.Equal(t, "Server Data 1", d.Name)
		case 3:
			foundServer3 = true
			assert.Equal(t, "Server Data 3", d.Name)
		case 2:
			foundUpdated2 = true
			assert.Equal(t, "Updated Client Data 2", d.Name)
		case 5:
			foundNew5 = true
			assert.Equal(t, "New Client Data", d.Name)
		case 6:
			foundNew6 = true
			assert.Equal(t, "Missing Server Data", d.Name)
		case -1:
			foundClientOnly = true
			assert.Equal(t, "Client Only Data", d.Name)
		}
	}

	assert.True(t, foundServer1, "Не найдены данные сервера с ID=1")
	assert.True(t, foundServer3, "Не найдены данные сервера с ID=3")
	assert.True(t, foundUpdated2, "Не найдены обновленные данные с ID=2")
	assert.True(t, foundNew5, "Не найдены новые данные с ID=5 (бывший ID=0)")
	assert.True(t, foundNew6, "Не найдены данные с ID=6 (бывший ID=4)")
	assert.True(t, foundClientOnly, "Не найдены данные только для клиента с ID=-1")
}
