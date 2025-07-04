package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
)

var (
	ErrInvalidResponse  = errors.New("неверный ответ сервера")
	ErrNotAuthenticated = errors.New("не авторизован")
	ErrAuthFailed       = errors.New("аутентификация не удалась")
	ErrDataNotFound     = errors.New("данные не найдены")
)

const (
	httpErrorCodeStart     = 400
	masterPasswordFileName = "master.key"
	configDir              = ".gophkeeper"
	clientTimeoutSeconds   = 10
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	logger     *slog.Logger
	salt       []byte
	configPath string
	username   string
}

func NewClient(baseURL string, insecureSkipVerify bool) *Client {
	httpClient := &http.Client{
		Timeout: clientTimeoutSeconds * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: createTLSConfig(insecureSkipVerify),
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

func createTLSConfig(insecureSkipVerify bool) *tls.Config {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if insecureSkipVerify {
		config.InsecureSkipVerify = true
	}

	return config
}

func (c *Client) Register(ctx context.Context, username, password string) error {
	creds := models.UserCredentials{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(ctx, "POST", "/api/auth/register", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response models.RegisterResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return errDecode
	}

	return nil
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	req := models.LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(ctx, "POST", "/api/auth/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrAuthFailed
	}

	var loginResp models.LoginResponse
	if errDecoder := json.NewDecoder(resp.Body).Decode(&loginResp); errDecoder != nil {
		return errDecoder
	}

	c.token = loginResp.Token
	c.username = username

	_ = c.initSalt()

	return nil
}

func (c *Client) SetAuthToken(token string) {
	c.token = token
}

func (c *Client) GetAuthToken() string {
	return c.token
}

func (c *Client) GetAllData() ([]models.DataResponse, error) {
	resp, err := c.sendRequest(context.Background(), http.MethodGet, "/api/data/", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []models.DataResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return nil, ErrInvalidResponse
	}

	for i := range response {
		if errProcessDataContent := c.processDataContent(&response[i]); errProcessDataContent != nil {
			return nil, errProcessDataContent
		}
	}

	return response, nil
}

func (c *Client) GetData(ctx context.Context, id int64) (*models.Data, error) {
	if c.token == "" {
		return nil, ErrNotAuthenticated
	}

	resp, err := c.sendRequest(ctx, "GET", fmt.Sprintf("/api/data/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrDataNotFound
	}

	var data models.Data
	if errDecode := json.NewDecoder(resp.Body).Decode(&data); errDecode != nil {
		return nil, ErrInvalidResponse
	}

	return &data, nil
}

func (c *Client) GetEncryptedData(id int64) (*models.Data, error) {
	resp, err := c.sendRequest(context.Background(), http.MethodGet, fmt.Sprintf("/api/data/%d/encrypted", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения данных: %s", resp.Status)
	}

	var data models.Data
	if errDecode := json.NewDecoder(resp.Body).Decode(&data); errDecode != nil {
		return nil, ErrInvalidResponse
	}

	return &data, nil
}

func (c *Client) DecryptData(data *models.Data, masterPassword string) (any, error) {
	key := c.deriveKeyFromPassword(masterPassword)

	decrypted, err := crypto.Decrypt(data.EncryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка расшифровки данных: %w", err)
	}

	var result any

	if data.Type == models.BinaryData {
		var binaryDataMap map[string]any
		if errUnmarshal := json.Unmarshal(decrypted, &binaryDataMap); errUnmarshal != nil {
			return nil, errors.New("ошибка десериализации бинарных данных")
		}

		fileName, ok := binaryDataMap["file_name"].(string)
		if !ok {
			return nil, errors.New("ошибка получения имени файла")
		}

		dataBase64, ok := binaryDataMap["data"].(string)
		if !ok {
			return nil, errors.New("ошибка получения данных файла")
		}

		fileData, errDecode := base64.StdEncoding.DecodeString(dataBase64)
		if errDecode != nil {
			return nil, errors.New("ошибка декодирования данных файла")
		}

		result = models.BinaryDataContent{
			FileName: fileName,
			Data:     fileData,
		}

		return result, nil
	}

	switch data.Type {
	case models.LoginPassword:
		var loginData models.LoginPasswordData
		if errUnmarshal := json.Unmarshal(decrypted, &loginData); errUnmarshal != nil {
			return nil, fmt.Errorf("ошибка десериализации данных: %w", errUnmarshal)
		}
		result = loginData
	case models.CardData:
		var cardData models.CardDataContent
		if errUnmarshal := json.Unmarshal(decrypted, &cardData); errUnmarshal != nil {
			return nil, fmt.Errorf("ошибка десериализации данных: %w", errUnmarshal)
		}
		result = cardData
	case models.TextData:
		var textData models.TextDataContent
		if errUnmarshal := json.Unmarshal(decrypted, &textData); errUnmarshal != nil {
			return nil, fmt.Errorf("ошибка десериализации данных: %w", errUnmarshal)
		}
		result = textData
	case models.BinaryData:
		var binaryData models.BinaryDataContent
		if errUnmarshal := json.Unmarshal(decrypted, &binaryData); errUnmarshal != nil {
			return nil, fmt.Errorf("ошибка десериализации данных: %w", errUnmarshal)
		}

		result = binaryData
	default:
		return nil, fmt.Errorf("неизвестный тип данных: %s", data.Type)
	}

	return result, nil
}

func (c *Client) EncryptData(data any, dataType models.DataType, masterPassword string) ([]byte, error) {
	var jsonData []byte
	var err error

	if dataType == models.BinaryData {
		if binaryData, ok := data.(models.BinaryDataContent); ok {
			binaryDataMap := map[string]any{
				"file_name": binaryData.FileName,
				"data":      base64.StdEncoding.EncodeToString(binaryData.Data),
			}
			jsonData, err = json.Marshal(binaryDataMap)
		} else {
			return nil, errors.New("ошибка приведения типа бинарных данных")
		}
	} else {
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации данных: %w", err)
	}

	key := c.deriveKeyFromPassword(masterPassword)

	encryptedData, err := crypto.Encrypt(jsonData, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка шифрования данных: %w", err)
	}

	return encryptedData, nil
}

func (c *Client) CreateData(
	ctx context.Context,
	name string,
	dataType models.DataType,
	encryptedData []byte,
) (int64, error) {
	if c.token == "" {
		return 0, ErrNotAuthenticated
	}

	req := models.DataRequest{
		Name:          name,
		Type:          dataType,
		EncryptedData: encryptedData,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	resp, err := c.sendRequest(ctx, "POST", "/api/data/", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response models.DataResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return 0, errDecode
	}

	return response.ID, nil
}

func (c *Client) UpdateData(
	ctx context.Context,
	id int64,
	name string,
	dataType models.DataType,
	encryptedData []byte,
) error {
	if c.token == "" {
		return ErrNotAuthenticated
	}

	req := models.DataRequest{
		Name:          name,
		Type:          dataType,
		EncryptedData: encryptedData,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(ctx, "PUT", fmt.Sprintf("/api/data/%d/", id), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response models.Response
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return errDecode
	}

	return nil
}

func (c *Client) DeleteData(ctx context.Context, id int64) error {
	if c.token == "" {
		return ErrNotAuthenticated
	}

	resp, err := c.sendRequest(ctx, "DELETE", fmt.Sprintf("/api/data/%d/", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response models.DeleteResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return errDecode
	}

	return nil
}

func (c *Client) SyncData(ctx context.Context, data []*models.Data) ([]*models.Data, error) {
	if c.token == "" {
		return nil, ErrNotAuthenticated
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := c.sendRequest(ctx, "POST", "/api/sync/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response models.SyncResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&response); errDecode != nil {
		return nil, errDecode
	}

	return response.Data, nil
}

func (c *Client) VerifyMasterPassword(password string) bool {
	if !c.HasMasterPassword() {
		return false
	}

	filePath := c.getMasterPasswordPath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	return crypto.VerifyPassword(password, string(data))
}

func (c *Client) GetMasterPassword() (string, error) {
	masterPasswordPath := c.getMasterPasswordPath()
	data, err := os.ReadFile(masterPasswordPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
	}

	return string(data), nil
}

func (c *Client) SetMasterPassword(password string) error {
	if c.username == "" {
		return errors.New("имя пользователя не задано")
	}

	salt := make([]byte, crypto.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("ошибка генерации соли: %w", err)
	}
	c.salt = salt

	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		return fmt.Errorf("ошибка хэширования пароля: %w", err)
	}

	passwordPath := filepath.Join(c.configPath, c.username+".pwd")
	if errCreateDir := os.MkdirAll(filepath.Dir(passwordPath), 0700); errCreateDir != nil {
		return fmt.Errorf("ошибка создания директории для пароля: %w", errCreateDir)
	}

	if errSavePass := os.WriteFile(passwordPath, []byte(hashedPassword), 0600); errSavePass != nil {
		return fmt.Errorf("ошибка сохранения пароля в файл: %w", errSavePass)
	}

	saltPath := c.getSaltPath()
	if errCreateSaltPath := os.MkdirAll(filepath.Dir(saltPath), 0700); errCreateSaltPath != nil {
		return fmt.Errorf("ошибка создания директории для соли: %w", errCreateSaltPath)
	}

	if errSaveSalt := os.WriteFile(saltPath, salt, 0600); errSaveSalt != nil {
		return fmt.Errorf("ошибка сохранения соли в файл: %w", errSaveSalt)
	}

	return nil
}

func (c *Client) HasMasterPassword() bool {
	_, err := os.Stat(c.getMasterPasswordPath())

	return err == nil
}

func (c *Client) GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ошибка получения домашней директории: %w", err)
	}

	return filepath.Join(homeDir, configDir), nil
}

func (c *Client) getMasterPasswordPath() string {
	configDir, err := c.GetConfigDir()
	if err != nil {
		return ""
	}

	if c.username == "" {
		return filepath.Join(configDir, masterPasswordFileName)
	}

	return filepath.Join(configDir, c.username+"_"+masterPasswordFileName)
}

func (c *Client) getSaltPath() string {
	configDir, err := c.GetConfigDir()
	if err != nil {
		return ""
	}

	if c.username == "" {
		return filepath.Join(configDir, "salt.key")
	}

	return filepath.Join(configDir, c.username+"_salt.key")
}

func (c *Client) initSalt() error {
	if c.username == "" {
		return errors.New("имя пользователя не задано")
	}

	saltPath := c.getSaltPath()

	if _, err := os.Stat(saltPath); os.IsNotExist(err) {
		salt := make([]byte, crypto.SaltLength)
		if _, errSalt := rand.Read(salt); errSalt != nil {
			return fmt.Errorf("ошибка генерации соли: %w", errSalt)
		}
		c.salt = salt

		if errCreateDir := os.MkdirAll(filepath.Dir(saltPath), 0700); errCreateDir != nil {
			return fmt.Errorf("ошибка создания директории для соли: %w", errCreateDir)
		}

		if errSaveSalt := os.WriteFile(saltPath, salt, 0600); errSaveSalt != nil {
			return fmt.Errorf("ошибка сохранения соли в файл: %w", errSaveSalt)
		}

		return nil
	}

	salt, err := os.ReadFile(saltPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла с солью: %w", err)
	}

	if len(salt) != crypto.SaltLength {
		return fmt.Errorf("неверный размер соли: %d, ожидается: %d", len(salt), crypto.SaltLength)
	}

	c.salt = salt

	return nil
}

func (c *Client) sendRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= httpErrorCodeStart {
		if errClose := resp.Body.Close(); errClose != nil {
			return nil, fmt.Errorf("ошибка при закрытии тела ответа: %w", errClose)
		}

		return nil, fmt.Errorf("ошибка HTTP: %d %s", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		var errResp models.ErrorResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&errResp); errDecode != nil {
			return nil, ErrNotAuthenticated
		}

		return nil, fmt.Errorf("%w: %s", ErrNotAuthenticated, errResp.Error)
	}

	return resp, nil
}

func (c *Client) processDataContent(data *models.DataResponse) error {
	if data.Content == nil {
		return nil
	}

	switch data.Type {
	case models.LoginPassword:
		var loginData models.LoginPasswordData
		if err := c.unmarshalContent(data.Content, &loginData); err != nil {
			return err
		}
		data.Content = loginData
	case models.TextData:
		var textData models.TextDataContent
		if err := c.unmarshalContent(data.Content, &textData); err != nil {
			return err
		}
		data.Content = textData
	case models.CardData:
		var cardData models.CardDataContent
		if err := c.unmarshalContent(data.Content, &cardData); err != nil {
			return err
		}
		data.Content = cardData
	case models.BinaryData:
		var binaryData models.BinaryDataContent
		if err := c.unmarshalContent(data.Content, &binaryData); err != nil {
			return err
		}
		data.Content = binaryData
	}

	return nil
}

func (c *Client) unmarshalContent(content any, data any) error {
	jsonData, err := json.Marshal(content)
	if err != nil {
		return err
	}

	if errUnmarshal := json.Unmarshal(jsonData, data); errUnmarshal != nil {
		return errUnmarshal
	}

	return nil
}

func (c *Client) deriveKeyFromPassword(password string) []byte {
	if len(c.salt) == 0 {
		tempSalt := make([]byte, crypto.SaltLength)
		copy(tempSalt, "temporary_salt")

		return crypto.GenerateKey([]byte(password), tempSalt)
	}

	key := crypto.GenerateKey([]byte(password), c.salt)

	return key
}
