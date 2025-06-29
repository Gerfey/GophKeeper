package server

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/gerfey/gophkeeper/pkg/logger"
)

var (
	ErrDataAccessDenied = errors.New("доступ к данным запрещен")
	ErrEncryptionFailed = errors.New("ошибка шифрования данных")
)

type DataRepository interface {
	CreateData(data *models.Data) (int64, error)
	GetByID(id, userID int64) (*models.Data, error)
	GetAll(userID int64) ([]*models.Data, error)
	Update(data *models.Data) error
	Delete(id, userID int64) error
}

type DataService struct {
	repo          DataRepository
	logger        logger.Logger
	encryptionKey []byte
}

func NewDataService(repo DataRepository, logger logger.Logger, encryptionKey []byte) *DataService {
	return &DataService{
		repo:          repo,
		logger:        logger,
		encryptionKey: encryptionKey,
	}
}

func (s *DataService) CreateData(data *models.Data) (int64, error) {
	err := s.encryptData(data)
	if err != nil {
		return 0, ErrEncryptionFailed
	}

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(data)
}

func (s *DataService) CreateDataWithEncrypted(data *models.Data) (int64, error) {
	if len(data.EncryptedData) == 0 {
		return 0, ErrEncryptionFailed
	}

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(data)
}

func (s *DataService) GetAllData(userID int64) ([]*models.Data, error) {
	return s.repo.GetAll(userID)
}

func (s *DataService) GetDataByID(id, userID int64) (*models.Data, error) {
	data, err := s.repo.GetByID(id, userID)
	if err != nil {
		return nil, err
	}

	if data.UserID != userID {
		return nil, ErrDataAccessDenied
	}

	return data, nil
}

func (s *DataService) UpdateData(data *models.Data) error {
	existingData, err := s.repo.GetByID(data.ID, data.UserID)
	if err != nil {
		return err
	}

	if existingData.UserID != data.UserID {
		return ErrDataAccessDenied
	}

	err = s.encryptData(data)
	if err != nil {
		return ErrEncryptionFailed
	}

	data.UpdatedAt = time.Now()

	return s.repo.Update(data)
}

func (s *DataService) DeleteData(id, userID int64) error {
	existingData, err := s.repo.GetByID(id, userID)
	if err != nil {
		return err
	}

	if existingData.UserID != userID {
		return ErrDataAccessDenied
	}

	return s.repo.Delete(id, userID)
}

func (s *DataService) SyncData(userID int64, clientData []*models.Data) ([]*models.Data, error) {
	serverData, err := s.repo.GetAll(userID)
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

	result = s.processClientData(clientData, serverDataMap, result)

	result = s.addMissingServerData(serverData, clientDataMap, result)

	return result, nil
}

func (s *DataService) processClientData(
	clientData []*models.Data,
	serverDataMap map[int64]*models.Data,
	result []*models.Data,
) []*models.Data {
	for _, data := range clientData {
		if data.ID == 0 {
			id, err := s.CreateDataWithEncrypted(data)
			if err != nil {
				continue
			}
			data.ID = id
			result = append(result, data)

			continue
		}

		//nolint:nestif
		if serverData, ok := serverDataMap[data.ID]; ok {
			if data.UpdatedAt.After(serverData.UpdatedAt) {
				err := s.repo.Update(data)
				if err != nil {
					continue
				}
				result = append(result, data)
			} else {
				result = append(result, serverData)
			}
		} else {
			data.ID = 0
			id, err := s.CreateDataWithEncrypted(data)
			if err != nil {
				continue
			}
			data.ID = id
			result = append(result, data)
		}
	}

	return result
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

func (s *DataService) encryptData(data *models.Data) error {
	if data.EncryptedData != nil {
		// Если данные уже зашифрованы, ничего не делаем
		return nil
	}

	var jsonData []byte
	var err error

	switch data.Type {
	case models.LoginPassword, models.TextData, models.CardData:
		jsonData, err = json.Marshal(data.Content)
		if err != nil {
			return errors.New("ошибка маршалинга данных")
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

	encryptedData, err := crypto.Encrypt(jsonData, s.encryptionKey)
	if err != nil {
		return errors.New("ошибка шифрования данных")
	}

	data.EncryptedData = encryptedData

	return nil
}
