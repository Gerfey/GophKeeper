package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
)

var (
	ErrUnauthorized         = errors.New("не авторизован")
	ErrServerError          = errors.New("ошибка сервера")
	ErrNetworkError         = errors.New("ошибка сети")
	ErrInvalidResponse      = errors.New("неверный ответ сервера")
	ErrMasterPasswordNotSet = errors.New("мастер-пароль не установлен")
)

const (
	masterPasswordFileName = ".gophkeeper_master"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

// NewClient создает новый клиент
func NewClient(baseURL string) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr,
		},
	}
}

// Register регистрирует нового пользователя
func (c *Client) Register(username, password string) (*models.UserResponse, string, error) {
	user := models.User{
		Username: username,
		Password: password,
	}

	resp, err := c.sendRequest("POST", "/api/auth/register", user, "")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var response struct {
		User  models.UserResponse `json:"user"`
		Token string              `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", ErrInvalidResponse
	}

	c.authToken = response.Token

	return &response.User, response.Token, nil
}

// Login авторизует пользователя
func (c *Client) Login(username, password string) (*models.UserResponse, string, error) {
	creds := models.UserCredentials{
		Username: username,
		Password: password,
	}

	resp, err := c.sendRequest("POST", "/api/auth/login", creds, "")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var response struct {
		User  models.UserResponse `json:"user"`
		Token string              `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", ErrInvalidResponse
	}

	c.authToken = response.Token

	return &response.User, response.Token, nil
}

// SetAuthToken устанавливает токен авторизации
func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

// GetAllData получает все данные пользователя
func (c *Client) GetAllData() ([]models.DataResponse, error) {
	resp, err := c.sendRequest("GET", "/api/data", nil, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	for i := range response {
		c.processDataContent(&response[i])
	}

	return response, nil
}

// GetData получает данные по ID
func (c *Client) GetData(id int64) (*models.DataResponse, error) {
	resp, err := c.sendRequest("GET", fmt.Sprintf("/api/data/%d", id), nil, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	c.processDataContent(&response)

	return &response, nil
}

// GetEncryptedData получает зашифрованные данные по ID
func (c *Client) GetEncryptedData(id int64) (*models.Data, error) {
	resp, err := c.sendRequest("GET", fmt.Sprintf("/api/data/%d/encrypted", id), nil, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения данных: %s", resp.Status)
	}

	var data models.Data
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, ErrInvalidResponse
	}

	return &data, nil
}

// DecryptData расшифровывает данные с использованием мастер-пароля
func (c *Client) DecryptData(data *models.Data, masterPassword string) (interface{}, error) {
	salt := []byte("gophkeeper-salt")
	key := crypto.GenerateKey([]byte(masterPassword), salt)

	decrypted, err := crypto.Decrypt(data.EncryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка расшифровки: %v", err)
	}

	var result interface{}
	switch data.Type {
	case models.LoginPassword:
		var loginData models.LoginPasswordData
		if err := json.Unmarshal(decrypted, &loginData); err != nil {
			return nil, fmt.Errorf("ошибка разбора данных: %v", err)
		}
		result = loginData
	case models.TextData:
		var textData models.TextDataContent
		if err := json.Unmarshal(decrypted, &textData); err != nil {
			return nil, fmt.Errorf("ошибка разбора данных: %v", err)
		}
		result = textData
	case models.CardData:
		var cardData models.CardDataContent
		if err := json.Unmarshal(decrypted, &cardData); err != nil {
			return nil, fmt.Errorf("ошибка разбора данных: %v", err)
		}
		result = cardData
	case models.BinaryData:
		result = decrypted
	default:
		return nil, fmt.Errorf("неизвестный тип данных: %s", data.Type)
	}

	return result, nil
}

// EncryptData шифрует данные с использованием мастер-пароля
func (c *Client) EncryptData(data interface{}, dataType models.DataType, masterPassword string) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации данных: %v", err)
	}

	salt := []byte("gophkeeper-salt")
	key := crypto.GenerateKey([]byte(masterPassword), salt)

	encryptedData, err := crypto.Encrypt(jsonData, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка шифрования: %v", err)
	}

	return encryptedData, nil
}

// CreateEncryptedData создает новые зашифрованные данные
func (c *Client) CreateEncryptedData(req models.DataRequest, masterPassword string) (*models.DataResponse, error) {
	encryptedData, err := c.EncryptData(req.Content, req.Type, masterPassword)
	if err != nil {
		return nil, err
	}

	encryptedReq := models.DataRequest{
		Type:          req.Type,
		Name:          req.Name,
		Metadata:      req.Metadata,
		EncryptedData: encryptedData,
	}

	resp, err := c.sendRequest("POST", "/api/data", encryptedReq, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("ошибка создания данных: %s", resp.Status)
	}

	var response models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	c.processDataContent(&response)

	return &response, nil
}

// CreateData создает новые данные
func (c *Client) CreateData(req models.DataRequest) (*models.DataResponse, error) {
	resp, err := c.sendRequest("POST", "/api/data", req, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	c.processDataContent(&response)

	return &response, nil
}

// UpdateData обновляет данные
func (c *Client) UpdateData(id int64, req models.DataRequest) (*models.DataResponse, error) {
	resp, err := c.sendRequest("PUT", fmt.Sprintf("/api/data/%d", id), req, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	c.processDataContent(&response)

	return &response, nil
}

// DeleteData удаляет данные
func (c *Client) DeleteData(id int64) error {
	resp, err := c.sendRequest("DELETE", fmt.Sprintf("/api/data/%d", id), nil, c.authToken)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// SyncData синхронизирует данные с сервером
func (c *Client) SyncData(data []*models.Data) ([]models.DataResponse, error) {
	resp, err := c.sendRequest("POST", "/api/sync", data, c.authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []models.DataResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	for i := range response {
		c.processDataContent(&response[i])
	}

	return response, nil
}

// GetMasterPassword возвращает мастер-пароль пользователя из локального хранилища
func (c *Client) GetMasterPassword() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ошибка получения домашней директории: %v", err)
	}

	masterPasswordPath := filepath.Join(homeDir, masterPasswordFileName)

	if _, err := os.Stat(masterPasswordPath); os.IsNotExist(err) {
		return "", ErrMasterPasswordNotSet
	}

	data, err := os.ReadFile(masterPasswordPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения мастер-пароля: %v", err)
	}

	return string(data), nil
}

// SetMasterPassword устанавливает мастер-пароль пользователя в локальное хранилище
func (c *Client) SetMasterPassword(password string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("ошибка получения домашней директории: %v", err)
	}

	masterPasswordPath := filepath.Join(homeDir, masterPasswordFileName)

	err = os.WriteFile(masterPasswordPath, []byte(password), 0600)
	if err != nil {
		return fmt.Errorf("ошибка записи мастер-пароля: %v", err)
	}

	return nil
}

// HasMasterPassword проверяет, установлен ли мастер-пароль
func (c *Client) HasMasterPassword() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	masterPasswordPath := filepath.Join(homeDir, masterPasswordFileName)

	_, err = os.Stat(masterPasswordPath)
	return err == nil
}

// sendRequest отправляет запрос на сервер
func (c *Client) sendRequest(method, path string, body interface{}, token string) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrNetworkError
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return resp, nil
	case http.StatusUnauthorized:
		resp.Body.Close()
		return nil, ErrUnauthorized
	default:
		resp.Body.Close()
		return nil, ErrServerError
	}
}

// processDataContent обрабатывает содержимое данных в зависимости от типа
func (c *Client) processDataContent(data *models.DataResponse) {
	if data.Content != nil {
		return
	}

	switch data.Type {
	case models.LoginPassword:
		data.Content = models.LoginPasswordData{
			Login:    "Логин недоступен без расшифровки",
			Password: "Пароль недоступен без расшифровки",
		}
	case models.TextData:
		data.Content = models.TextDataContent{
			Content: "Содержимое недоступно без расшифровки",
		}
	case models.CardData:
		data.Content = models.CardDataContent{
			CardNumber: "Номер карты недоступен без расшифровки",
			CardHolder: "Владелец карты недоступен без расшифровки",
			ExpiryDate: "Срок действия недоступен без расшифровки",
			CVV:        "CVV недоступен без расшифровки",
		}
	case models.BinaryData:
		data.Content = []byte("Бинарные данные недоступны без расшифровки")
	}
}
