package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	KeyLength   = 32
	SaltLength  = 16
	Memory      = 64 * 1024
	Iterations  = 1
	Parallelism = 4
)

var (
	ErrInvalidKeyLength = errors.New("неверная длина ключа")
	ErrDataTooShort     = errors.New("данные слишком короткие")
	ErrInvalidData      = errors.New("неверные данные")
)

func GenerateKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, Iterations, Memory, Parallelism, KeyLength)
}

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != KeyLength {
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != KeyLength {
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

	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrDataTooShort
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidData
	}

	return plaintext, nil
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	_, err := rand.Read(salt)

	return salt, err
}

func HashPassword(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, Iterations, Memory, Parallelism, KeyLength)

	saltAndHash := make([]byte, 0, len(salt)+len(hash))
	saltAndHash = append(saltAndHash, salt...)
	saltAndHash = append(saltAndHash, hash...)

	return base64.StdEncoding.EncodeToString(saltAndHash), nil
}

func VerifyPassword(password, encodedHash string) bool {
	saltAndHash, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false
	}

	if len(saltAndHash) < SaltLength {
		return false
	}

	salt := saltAndHash[:SaltLength]
	storedHash := saltAndHash[SaltLength:]

	hash := argon2.IDKey([]byte(password), salt, Iterations, Memory, Parallelism, KeyLength)

	return subtle.ConstantTimeCompare(hash, storedHash) == 1
}
