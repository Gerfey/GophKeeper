package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gerfey/gophkeeper/internal/models"
)

func TestDataToDataResponse(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		data     models.Data
		expected models.DataResponse
	}{
		{
			name: "LoginPassword",
			data: models.Data{
				ID:            1,
				UserID:        10,
				Type:          models.LoginPassword,
				Name:          "Test Login",
				EncryptedData: []byte("encrypted"),
				Content:       models.LoginPasswordData{Login: "user", Password: "pass"},
				Metadata:      "meta",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: models.DataResponse{
				ID:        1,
				Type:      models.LoginPassword,
				Name:      "Test Login",
				Content:   models.LoginPasswordData{Login: "user", Password: "pass"},
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "TextData",
			data: models.Data{
				ID:            2,
				UserID:        10,
				Type:          models.TextData,
				Name:          "Test Text",
				EncryptedData: []byte("encrypted"),
				Content:       models.TextDataContent{Content: "text content"},
				Metadata:      "meta",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: models.DataResponse{
				ID:        2,
				Type:      models.TextData,
				Name:      "Test Text",
				Content:   models.TextDataContent{Content: "text content"},
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "BinaryData",
			data: models.Data{
				ID:            3,
				UserID:        10,
				Type:          models.BinaryData,
				Name:          "Test Binary",
				EncryptedData: []byte("encrypted"),
				Content:       models.BinaryDataContent{FileName: "test.txt", Data: []byte("binary data")},
				Metadata:      "meta",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: models.DataResponse{
				ID:        3,
				Type:      models.BinaryData,
				Name:      "Test Binary",
				Content:   models.BinaryDataContent{FileName: "test.txt", Data: []byte("binary data")},
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "CardData",
			data: models.Data{
				ID:            4,
				UserID:        10,
				Type:          models.CardData,
				Name:          "Test Card",
				EncryptedData: []byte("encrypted"),
				Content: models.CardDataContent{
					CardNumber: "1234567890123456",
					CardHolder: "Test User",
					ExpiryDate: "12/25",
					CVV:        "123",
				},
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: models.DataResponse{
				ID:   4,
				Type: models.CardData,
				Name: "Test Card",
				Content: models.CardDataContent{
					CardNumber: "1234567890123456",
					CardHolder: "Test User",
					ExpiryDate: "12/25",
					CVV:        "123",
				},
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "NilContent",
			data: models.Data{
				ID:            5,
				UserID:        10,
				Type:          models.TextData,
				Name:          "Test Nil",
				EncryptedData: []byte("encrypted"),
				Content:       nil,
				Metadata:      "meta",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: models.DataResponse{
				ID:        5,
				Type:      models.TextData,
				Name:      "Test Nil",
				Content:   nil,
				Metadata:  "meta",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := tc.data.ToDataResponse()

			assert.Equal(t, tc.expected.ID, response.ID)
			assert.Equal(t, tc.expected.Type, response.Type)
			assert.Equal(t, tc.expected.Name, response.Name)
			assert.Equal(t, tc.expected.Content, response.Content)
			assert.Equal(t, tc.expected.Metadata, response.Metadata)
			assert.Equal(t, tc.expected.CreatedAt, response.CreatedAt)
			assert.Equal(t, tc.expected.UpdatedAt, response.UpdatedAt)
		})
	}
}
