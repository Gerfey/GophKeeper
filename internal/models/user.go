package models

import "time"

type User struct {
	ID        int64     `json:"id"         db:"id"`
	Username  string    `json:"username"   db:"username"      validate:"required,min=3,max=50"`
	Password  string    `json:"-"          db:"password_hash" validate:"required,min=6"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UserCredentials struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
