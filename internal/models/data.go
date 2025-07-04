package models

import "time"

type DataType string

const (
	LoginPassword DataType = "login_password"
	TextData      DataType = "text_data"
	BinaryData    DataType = "binary_data"
	CardData      DataType = "card_data"
)

type Data struct {
	ID            int64     `json:"id"             db:"id"`
	UserID        int64     `json:"user_id"        db:"user_id"`
	Type          DataType  `json:"type"           db:"data_type"`
	Name          string    `json:"name"           db:"name"           validate:"required,min=1,max=100"`
	EncryptedData []byte    `json:"encrypted_data" db:"encrypted_data"`
	Content       any       `json:"-"              db:"-"`
	Metadata      string    `json:"metadata"       db:"metadata"`
	CreatedAt     time.Time `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"     db:"updated_at"`
}

type LoginPasswordData struct {
	Login    string `json:"login"    validate:"required"`
	Password string `json:"password" validate:"required"`
}

type TextDataContent struct {
	Content string `json:"content" validate:"required"`
	Text    string `json:"text"`
}

type CardDataContent struct {
	CardNumber string `json:"card_number" validate:"required,min=16,max=19"`
	CardHolder string `json:"card_holder" validate:"required"`
	ExpiryDate string `json:"expiry_date" validate:"required"`
	CVV        string `json:"cvv"         validate:"required,min=3,max=4"`
	CardExpiry string `json:"card_expiry"`
	CardCVV    string `json:"card_cvv"`
}

type BinaryDataContent struct {
	FileName string `json:"file_name" validate:"required"`
	Data     []byte `json:"data"      validate:"required"`
}

type DataRequest struct {
	Type          DataType `json:"type"                     validate:"required"`
	Name          string   `json:"name"                     validate:"required,min=1,max=100"`
	Content       any      `json:"content,omitempty"`
	EncryptedData []byte   `json:"encrypted_data,omitempty"`
	Metadata      string   `json:"metadata"`
}

type DataResponse struct {
	ID        int64     `json:"id"`
	Type      DataType  `json:"type"`
	Name      string    `json:"name"`
	Content   any       `json:"content"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *Data) ToDataResponse() DataResponse {
	return DataResponse{
		ID:        d.ID,
		Type:      d.Type,
		Name:      d.Name,
		Content:   d.Content,
		Metadata:  d.Metadata,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
