package server

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

var (
	ErrDataAccessDenied = errors.New("доступ к данным запрещен")
	ErrEncryptionFailed = errors.New("ошибка шифрования данных")
)

type DataRepository interface {
	CreateData(ctx context.Context, data *models.Data) (int64, error)
	GetByID(ctx context.Context, id, userID int64) (*models.Data, error)
	GetAll(ctx context.Context, userID int64) ([]*models.Data, error)
	Update(ctx context.Context, data *models.Data) error
	Delete(ctx context.Context, id, userID int64) error
}

type DataService struct {
	repo   DataRepository
	logger logger.Logger
}

func NewDataService(repo DataRepository, logger logger.Logger) *DataService {
	return &DataService{
		repo:   repo,
		logger: logger,
	}
}

func (s *DataService) CreateData(ctx context.Context, data *models.Data) (int64, error) {
	err := s.prepareData(data)
	if err != nil {
		return 0, ErrEncryptionFailed
	}

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(ctx, data)
}

func (s *DataService) CreateDataWithEncrypted(ctx context.Context, data *models.Data) (int64, error) {
	if len(data.EncryptedData) == 0 {
		return 0, ErrEncryptionFailed
	}

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(ctx, data)
}

func (s *DataService) GetAllData(ctx context.Context, userID int64) ([]*models.Data, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *DataService) GetDataByID(ctx context.Context, id, userID int64) (*models.Data, error) {
	data, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if data.UserID != userID {
		return nil, ErrDataAccessDenied
	}

	return data, nil
}

func (s *DataService) UpdateData(ctx context.Context, data *models.Data) error {
	existingData, err := s.repo.GetByID(ctx, data.ID, data.UserID)
	if err != nil {
		return err
	}

	if existingData.UserID != data.UserID {
		return ErrDataAccessDenied
	}

	err = s.prepareData(data)
	if err != nil {
		return ErrEncryptionFailed
	}

	if data.Type == "" {
		data.Type = existingData.Type
	}

	if data.Name == "" {
		data.Name = existingData.Name
	}

	data.UpdatedAt = time.Now()

	return s.repo.Update(ctx, data)
}

func (s *DataService) DeleteData(ctx context.Context, id, userID int64) error {
	existingData, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}

	if existingData.UserID != userID {
		return ErrDataAccessDenied
	}

	return s.repo.Delete(ctx, id, userID)
}

func (s *DataService) SyncData(ctx context.Context, userID int64, clientData []*models.Data) ([]*models.Data, error) {
	serverData, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}

	serverDataMap := make(map[int64]*models.Data)
	for _, data := range serverData {
		serverDataMap[data.ID] = data
	}

	clientDataMap := make(map[int64]*models.Data)
	for _, data := range clientData {
		if data.ID > 0 {
			clientDataMap[data.ID] = data
		}
	}

	var result []*models.Data

	result = s.processClientData(ctx, clientData, serverDataMap, result)

	result = s.addMissingServerData(serverData, clientDataMap, result)

	return result, nil
}

func (s *DataService) processClientData(
	ctx context.Context,
	clientData []*models.Data,
	serverDataMap map[int64]*models.Data,
	result []*models.Data,
) []*models.Data {
	for _, data := range clientData {
		if data.ID < 0 {
			result = append(result, data)

			continue
		}

		if data.ID == 0 {
			result = s.processNewClientData(ctx, data, result)

			continue
		}

		result = s.processExistingClientData(ctx, data, serverDataMap, result)
	}

	return result
}

func (s *DataService) processNewClientData(
	ctx context.Context,
	data *models.Data,
	result []*models.Data,
) []*models.Data {
	id, err := s.CreateDataWithEncrypted(ctx, data)
	if err != nil {
		return result
	}

	data.ID = id

	return append(result, data)
}

func (s *DataService) processExistingClientData(
	ctx context.Context,
	data *models.Data,
	serverDataMap map[int64]*models.Data,
	result []*models.Data,
) []*models.Data {
	serverData, exists := serverDataMap[data.ID]
	if !exists {
		return s.handleNonExistentServerData(ctx, data, result)
	}

	return s.handleDataConflict(ctx, data, serverData, result)
}

func (s *DataService) handleNonExistentServerData(
	ctx context.Context,
	data *models.Data,
	result []*models.Data,
) []*models.Data {
	data.ID = 0
	id, err := s.CreateDataWithEncrypted(ctx, data)
	if err != nil {
		return result
	}

	data.ID = id

	return append(result, data)
}

func (s *DataService) handleDataConflict(
	ctx context.Context,
	data *models.Data,
	serverData *models.Data,
	result []*models.Data,
) []*models.Data {
	if !data.UpdatedAt.After(serverData.UpdatedAt) {
		return append(result, serverData)
	}

	err := s.repo.Update(ctx, data)
	if err != nil {
		return result
	}

	return append(result, data)
}

func (s *DataService) addMissingServerData(
	serverData []*models.Data,
	clientDataMap map[int64]*models.Data,
	result []*models.Data,
) []*models.Data {
	for _, data := range serverData {
		if _, ok := clientDataMap[data.ID]; !ok {
			result = append(result, data)
		}
	}

	return result
}

func (s *DataService) prepareData(data *models.Data) error {
	if data.EncryptedData != nil {
		return nil
	}

	var jsonData []byte
	var err error

	switch data.Type {
	case models.LoginPassword, models.TextData, models.CardData:
		jsonData, err = json.Marshal(data.Content)
		if err != nil {
			return err
		}
	case models.BinaryData:
		if content, ok := data.Content.([]byte); ok {
			jsonData = content
		} else {
			return errors.New("неверный формат бинарных данных")
		}
	default:
		return errors.New("неизвестный тип данных")
	}

	data.EncryptedData = jsonData
	data.Content = nil

	return nil
}
