package models

import "time"

// User представляет модель пользователя в системе
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username" validate:"required,min=3,max=50"`
	Password  string    `json:"password,omitempty" db:"password_hash" validate:"required,min=6"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserCredentials представляет данные для аутентификации пользователя
type UserCredentials struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserResponse представляет ответ с данными пользователя (без пароля)
type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// ToUserResponse преобразует User в UserResponse
func (u *User) ToUserResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}
