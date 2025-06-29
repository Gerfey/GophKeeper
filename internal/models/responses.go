package models

type ErrorResponse struct {
	Error string `json:"error"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type RegisterResponse struct {
	UserID int64  `json:"user_id"`
	Token  string `json:"token,omitempty"`
}

type LoginResponse struct {
	UserID int64  `json:"user_id"`
	Token  string `json:"token"`
}

type SyncResponse struct {
	Data []*Data `json:"data"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
