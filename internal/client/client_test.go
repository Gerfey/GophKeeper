package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/gerfey/gophkeeper/internal/crypto"
	"github.com/gerfey/gophkeeper/internal/models"
)

func TestEncryptDecrypt(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gophkeeper_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	c.SetConfigPath(tempDir)
	c.SetUsername("testuser")

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	c.SetSalt(salt)

	saltFilePath := filepath.Join(tempDir, "testuser.salt")
	err = os.WriteFile(saltFilePath, salt, 0600)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		dataType models.DataType
		content  interface{}
	}{
		{
			name:     "LoginPassword",
			dataType: models.LoginPassword,
			content: models.LoginPasswordData{
				Login:    "testuser",
				Password: "testpassword",
			},
		},
		{
			name:     "TextData",
			dataType: models.TextData,
			content: models.TextDataContent{
				Content: "Test text data",
				Text:    "Test text data",
			},
		},
		{
			name:     "BinaryData",
			dataType: models.BinaryData,
			content: models.BinaryDataContent{
				FileName: "test.bin",
				Data:     []byte("Test binary data"),
			},
		},
		{
			name:     "CardData",
			dataType: models.CardData,
			content: models.CardDataContent{
				CardNumber: "1234567890123456",
				CardHolder: "Test User",
				ExpiryDate: "12/25",
				CVV:        "123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := &models.Data{
				Type:    tc.dataType,
				Name:    "test_" + string(tc.dataType),
				Content: tc.content,
			}

			masterPassword := "test_master_password"
			encryptedData, err := c.EncryptData(tc.content, tc.dataType, masterPassword)
			require.NoError(t, err)
			assert.NotEmpty(t, encryptedData)

			data.EncryptedData = encryptedData

			decryptedContent, err := c.DecryptData(data, masterPassword)
			require.NoError(t, err)
			assert.NotNil(t, decryptedContent)

			switch tc.dataType {
			case models.LoginPassword:
				original := tc.content.(models.LoginPasswordData)
				decrypted := decryptedContent.(models.LoginPasswordData)
				assert.Equal(t, original.Login, decrypted.Login)
				assert.Equal(t, original.Password, decrypted.Password)
			case models.TextData:
				original := tc.content.(models.TextDataContent)
				decrypted := decryptedContent.(models.TextDataContent)
				assert.Equal(t, original.Content, decrypted.Content)
				assert.Equal(t, original.Text, decrypted.Text)
			case models.BinaryData:
				original := tc.content.(models.BinaryDataContent)
				decrypted := decryptedContent.(models.BinaryDataContent)
				assert.Equal(t, original.FileName, decrypted.FileName)
				assert.Equal(t, original.Data, decrypted.Data)
			case models.CardData:
				original := tc.content.(models.CardDataContent)
				decrypted := decryptedContent.(models.CardDataContent)
				assert.Equal(t, original.CardNumber, decrypted.CardNumber)
				assert.Equal(t, original.CardHolder, decrypted.CardHolder)
				assert.Equal(t, original.ExpiryDate, decrypted.ExpiryDate)
				assert.Equal(t, original.CVV, decrypted.CVV)
			}
		})
	}
}

func TestVerifyMasterPassword(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gophkeeper_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c := &Client{}

	c.SetConfigPath(tempDir)
	c.SetUsername("testuser")

	password := "test_master_password"
	hashedPassword, err := crypto.HashPassword(password)
	require.NoError(t, err)

	err = os.MkdirAll(tempDir, 0700)
	require.NoError(t, err)

	passwordFilePath := filepath.Join(tempDir, "testuser.pwd")
	err = os.WriteFile(passwordFilePath, []byte(hashedPassword), 0600)
	require.NoError(t, err)

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	saltFilePath := filepath.Join(tempDir, "testuser.salt")
	err = os.WriteFile(saltFilePath, salt, 0600)
	require.NoError(t, err)

	c.SetSalt(salt)

	_, err = os.Stat(passwordFilePath)
	assert.NoError(t, err)

	data, err := os.ReadFile(passwordFilePath)
	require.NoError(t, err)
	isValid := crypto.VerifyPassword(password, string(data))
	assert.True(t, isValid)
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	jsonResponse := `{"token":"test_token"}`
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
		Header:     make(http.Header),
	}

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "/api/auth/login", req.URL.Path)

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var loginReq models.LoginRequest
			err = json.Unmarshal(body, &loginReq)
			require.NoError(t, err)

			assert.Equal(t, "testuser", loginReq.Username)
			assert.Equal(t, "password", loginReq.Password)

			return mockResp, nil
		})

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	tempDir, err := os.MkdirTemp("", "gophkeeper_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c.SetConfigPath(tempDir)

	err = c.Login(t.Context(), "testuser", "password")

	require.NoError(t, err)
	assert.Equal(t, "test_token", c.GetToken())
	assert.Equal(t, "testuser", c.GetUsername())
}

func TestLoginError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		Return(nil, assert.AnError)

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	err := c.Login(t.Context(), "testuser", "password")

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestLoginAuthFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		Return(nil, fmt.Errorf("ошибка HTTP: 401 Unauthorized"))

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	err := c.Login(t.Context(), "testuser", "wrongpassword")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestCreateData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockResp := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(bytes.NewBufferString(`{"id":123}`)),
	}

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "/api/data/", req.URL.Path)

			assert.Equal(t, "Bearer test_token", req.Header.Get("Authorization"))

			var reqBody models.DataRequest
			err := json.NewDecoder(req.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "test_data", reqBody.Name)
			assert.Equal(t, models.TextData, reqBody.Type)

			return mockResp, nil
		})

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	c.SetToken("test_token")

	id, err := c.CreateData(t.Context(), "test_data", models.TextData, []byte("test_content"))

	require.NoError(t, err)
	assert.Equal(t, int64(123), id)
}

func TestCreateDataNotAuthenticated(t *testing.T) {
	c := &Client{}

	_, err := c.CreateData(t.Context(), "test_data", models.TextData, []byte("test_content"))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestGetData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBufferString(`{
			"id": 123,
			"name": "test_data",
			"type": "text_data",
			"encrypted_data": "dGVzdF9kYXRh",
			"created_at": "2023-01-01T00:00:00Z",
			"updated_at": "2023-01-01T00:00:00Z"
		}`)),
	}

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "GET", req.Method)
			assert.Equal(t, "/api/data/123", req.URL.Path)

			assert.Equal(t, "Bearer test_token", req.Header.Get("Authorization"))

			return mockResp, nil
		})

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	c.SetToken("test_token")

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	c.SetSalt(salt)

	data, err := c.GetData(t.Context(), 123)

	require.NoError(t, err)
	assert.Equal(t, int64(123), data.ID)
	assert.Equal(t, "test_data", data.Name)
	assert.Equal(t, models.TextData, data.Type)
}

func TestGetDataNotAuthenticated(t *testing.T) {
	c := &Client{}

	_, err := c.GetData(t.Context(), 123)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestGetDataNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		Return(nil, fmt.Errorf("ошибка HTTP: 404 Not Found"))

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	c.SetToken("test_token")

	_, err := c.GetData(t.Context(), 123)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGetAllData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	jsonResponse := `[{
		"id": 123,
		"type": "text_data",
		"name": "test_data_1",
		"encrypted_data": "ZW5jcnlwdGVkX2RhdGFfMQ=="
	},
	{
		"id": 124,
		"type": "login_password",
		"name": "test_data_2",
		"encrypted_data": "ZW5jcnlwdGVkX2RhdGFfMg=="
	}]`
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
		Header:     make(http.Header),
	}

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "GET", req.Method)
			assert.Equal(t, "/api/data/", req.URL.Path)

			return mockResp, nil
		})

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	c.SetSalt(salt)

	tempDir, err := os.MkdirTemp("", "gophkeeper_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c.SetConfigPath(tempDir)
	c.SetUsername("testuser")

	saltFilePath := filepath.Join(tempDir, "testuser.salt")
	err = os.WriteFile(saltFilePath, salt, 0600)
	require.NoError(t, err)

	dataList, err := c.GetAllData()

	require.NoError(t, err)
	assert.Len(t, dataList, 2)
	assert.Equal(t, int64(123), dataList[0].ID)
	assert.Equal(t, models.TextData, dataList[0].Type)
	assert.Equal(t, "test_data_1", dataList[0].Name)
	assert.Equal(t, int64(124), dataList[1].ID)
	assert.Equal(t, models.LoginPassword, dataList[1].Type)
	assert.Equal(t, "test_data_2", dataList[1].Name)
}

func TestDeleteData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := NewMockHTTPClient(ctrl)

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"success":true}`)),
	}

	mockHTTP.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "DELETE", req.Method)
			assert.Equal(t, "/api/data/123/", req.URL.Path)

			assert.Equal(t, "Bearer test_token", req.Header.Get("Authorization"))

			return mockResp, nil
		})

	c := &Client{}
	c.SetHTTPClient(mockHTTP)

	c.SetToken("test_token")

	err := c.DeleteData(t.Context(), 123)

	require.NoError(t, err)
}

func TestDeleteDataNotAuthenticated(t *testing.T) {
	c := &Client{}

	err := c.DeleteData(t.Context(), 123)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}
