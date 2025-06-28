package auth

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrInvalidToken = errors.New("неверный токен")
	ErrExpiredToken = errors.New("токен истек")
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.StandardClaims
}

type TokenManager interface {
	GenerateToken(userID int64, ttl time.Duration) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
}

type JWTManager struct {
	signingKey []byte
}

func NewJWTManager(signingKey string) TokenManager {
	return &JWTManager{
		signingKey: []byte(signingKey),
	}
}

// GenerateToken создает новый JWT токен для пользователя
func (m *JWTManager) GenerateToken(userID int64, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(ttl).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.signingKey)
}

// ValidateToken проверяет и возвращает данные из JWT токена
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return m.signingKey, nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, ErrExpiredToken
	}

	return claims, nil
}
