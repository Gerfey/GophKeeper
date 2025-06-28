package models

import "time"

type DataType string

const (
	// LoginPassword тип данных для хранения логина и пароля
	LoginPassword DataType = "login_password"
	// TextData тип данных для хранения текстовой информации
	TextData DataType = "text_data"
	// BinaryData тип данных для хранения бинарной информации
	BinaryData DataType = "binary_data"
	// CardData тип данных для хранения информации о банковских картах
	CardData DataType = "card_data"
)

// Data представляет модель для хранения зашифрованных данных пользователя
type Data struct {
	ID            int64     `json:"id" db:"id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	Type          DataType  `json:"type" db:"data_type"`
	Name          string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	EncryptedData []byte    `json:"encrypted_data" db:"encrypted_data"`
	Metadata      string    `json:"metadata" db:"metadata"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// LoginPasswordData представляет данные типа логин/пароль
type LoginPasswordData struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// TextDataContent представляет текстовые данные
type TextDataContent struct {
	Content string `json:"content" validate:"required"`
}

// CardDataContent представляет данные банковской карты
type CardDataContent struct {
	CardNumber string `json:"card_number" validate:"required,min=16,max=19"`
	CardHolder string `json:"card_holder" validate:"required"`
	ExpiryDate string `json:"expiry_date" validate:"required"`
	CVV        string `json:"cvv" validate:"required,min=3,max=4"`
}

// DataRequest представляет запрос на создание или обновление данных
type DataRequest struct {
	Type          DataType    `json:"type" validate:"required"`
	Name          string      `json:"name" validate:"required,min=1,max=100"`
	Content       interface{} `json:"content,omitempty"`
	EncryptedData []byte      `json:"encrypted_data,omitempty"`
	Metadata      string      `json:"metadata"`
}

// DataResponse представляет ответ с данными (без зашифрованных данных)
type DataResponse struct {
	ID        int64       `json:"id"`
	Type      DataType    `json:"type"`
	Name      string      `json:"name"`
	Content   interface{} `json:"content"`
	Metadata  string      `json:"metadata"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// ToDataResponse преобразует Data в DataResponse
func (d *Data) ToDataResponse() DataResponse {
	return DataResponse{
		ID:        d.ID,
		Type:      d.Type,
		Name:      d.Name,
		Content:   nil,
		Metadata:  d.Metadata,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
