package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidKeyLength = errors.New("неверная длина ключа")
	ErrDataTooShort     = errors.New("данные слишком короткие")
	ErrInvalidData      = errors.New("неверные данные")
)

// GenerateKey генерирует ключ шифрования на основе пароля пользователя
func GenerateKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}

// Encrypt шифрует данные с использованием AES-256-GCM
func Encrypt(data []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt расшифровывает данные с использованием AES-256-GCM
func Decrypt(encryptedData []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(encryptedData) < gcm.NonceSize() {
		return nil, ErrDataTooShort
	}

	nonce, ciphertext := encryptedData[:gcm.NonceSize()], encryptedData[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidData
	}

	return plaintext, nil
}

// GenerateSalt создает случайную соль для хеширования
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, salt)
	return salt, err
}

// HashPassword хеширует пароль с использованием Argon2id
func HashPassword(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	saltAndHash := append(salt, hash...)
	return base64.StdEncoding.EncodeToString(saltAndHash), nil
}

// VerifyPassword проверяет соответствие пароля хешу
func VerifyPassword(password, encodedHash string) (bool, error) {
	saltAndHash, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, err
	}

	if len(saltAndHash) < 16+32 {
		return false, ErrInvalidData
	}

	salt := saltAndHash[:16]
	storedHash := saltAndHash[16:]

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return sha256.Sum256(hash) == sha256.Sum256(storedHash), nil
}
