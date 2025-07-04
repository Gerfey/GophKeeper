package auth

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("токен недействителен")
	ErrExpiredToken = errors.New("срок действия токена истек")
)

type UserClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenManager interface {
	GenerateToken(userID int64, duration time.Duration) (string, error)
	ValidateToken(token string) (*UserClaims, error)
	GetUserID(token string) (int64, error)
}

type JWTManager struct {
	signingKey []byte
}

func NewJWTManager(signingKey string) *JWTManager {
	return &JWTManager{
		signingKey: []byte(signingKey),
	}
}

func (m *JWTManager) GenerateToken(userID int64, duration time.Duration) (string, error) {
	payload := &UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString(m.signingKey)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&UserClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("неожиданный метод подписи")
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

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

func (m *JWTManager) GetUserID(tokenStr string) (int64, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&UserClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("неожиданный метод подписи")
			}

			return m.signingKey, nil
		},
	)

	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return 0, errors.New("неверные JWT claims")
	}

	return claims.UserID, nil
}
