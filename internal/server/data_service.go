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
	ErrDataNotFound     = errors.New("данные не найдены")
	ErrDataAccessDenied = errors.New("доступ к данным запрещен")
	ErrEncryptionFailed = errors.New("ошибка шифрования данных")
	ErrDecryptionFailed = errors.New("ошибка расшифровки данных")
)

type DataRepository interface {
	CreateData(data *models.Data) (int64, error)
	GetAll(userID int64) ([]*models.Data, error)
	GetByID(id int64) (*models.Data, error)
	Update(data *models.Data) error
	Delete(id int64) error
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

// CreateData создает новые данные
func (s *DataService) CreateData(data *models.Data) (int64, error) {
	encryptedData, err := s.encryptData(data)
	if err != nil {
		return 0, ErrEncryptionFailed
	}

	data.EncryptedData = encryptedData

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(data)
}

// CreateDataWithEncrypted создает новые данные с уже зашифрованным содержимым
func (s *DataService) CreateDataWithEncrypted(data *models.Data) (int64, error) {
	if data.EncryptedData == nil || len(data.EncryptedData) == 0 {
		return 0, ErrEncryptionFailed
	}

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	return s.repo.CreateData(data)
}

// GetAllData возвращает все данные пользователя
func (s *DataService) GetAllData(userID int64) ([]*models.Data, error) {
	return s.repo.GetAll(userID)
}

// GetDataByID возвращает данные по ID
func (s *DataService) GetDataByID(id, userID int64) (*models.Data, error) {
	data, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if data.UserID != userID {
		return nil, ErrDataAccessDenied
	}

	return data, nil
}

// UpdateData обновляет данные
func (s *DataService) UpdateData(data *models.Data) error {
	existingData, err := s.repo.GetByID(data.ID)
	if err != nil {
		return err
	}

	if existingData.UserID != data.UserID {
		return ErrDataAccessDenied
	}

	encryptedData, err := s.encryptData(data)
	if err != nil {
		return ErrEncryptionFailed
	}

	data.EncryptedData = encryptedData

	data.UpdatedAt = time.Now()

	return s.repo.Update(data)
}

// DeleteData удаляет данные
func (s *DataService) DeleteData(id, userID int64) error {
	existingData, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if existingData.UserID != userID {
		return ErrDataAccessDenied
	}

	return s.repo.Delete(id)
}

// SyncData синхронизирует данные между клиентом и сервером
func (s *DataService) SyncData(userID int64, clientData []*models.Data) ([]*models.Data, error) {
	serverData, err := s.repo.GetAll(userID)
	if err != nil {
		return nil, err
	}

	serverDataMap := make(map[int64]*models.Data)
	for _, data := range serverData {
		serverDataMap[data.ID] = data
	}

	for _, data := range clientData {
		if serverData, ok := serverDataMap[data.ID]; ok {
			if data.UpdatedAt.After(serverData.UpdatedAt) {
				data.UserID = userID
				err := s.UpdateData(data)
				if err != nil {
					return nil, err
				}
			}
		} else {
			data.UserID = userID
			_, err := s.CreateData(data)
			if err != nil {
				return nil, err
			}
		}
	}

	return s.repo.GetAll(userID)
}

// encryptData шифрует данные
func (s *DataService) encryptData(data *models.Data) ([]byte, error) {
	var content interface{}

	switch data.Type {
	case models.LoginPassword:
		content = &models.LoginPasswordData{}
	case models.TextData:
		content = &models.TextDataContent{}
	case models.CardData:
		content = &models.CardDataContent{}
	case models.BinaryData:
		// Для бинарных данных не нужна специальная структура
		return crypto.Encrypt(data.EncryptedData, s.encryptionKey)
	default:
		return nil, errors.New("неизвестный тип данных")
	}

	jsonData, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	return crypto.Encrypt(jsonData, s.encryptionKey)
}

// decryptData расшифровывает данные
func (s *DataService) decryptData(data *models.Data, result interface{}) error {
	decrypted, err := crypto.Decrypt(data.EncryptedData, s.encryptionKey)
	if err != nil {
		return ErrDecryptionFailed
	}

	if data.Type == models.BinaryData {
		if resultBytes, ok := result.(*[]byte); ok {
			*resultBytes = decrypted
			return nil
		}
		return errors.New("неверный тип результата для бинарных данных")
	}

	return json.Unmarshal(decrypted, result)
}
