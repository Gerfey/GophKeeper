package crypto_test

import (
	"bytes"
	"testing"

	"github.com/gerfey/gophkeeper/internal/crypto"
)

func TestGenerateKey(t *testing.T) {
	password := []byte("test_password")
	salt := []byte("test_salt_16bytes")

	key := crypto.GenerateKey(password, salt)

	if len(key) != crypto.KeyLength {
		t.Errorf("Длина ключа %d не соответствует ожидаемой %d", len(key), crypto.KeyLength)
	}

	// Проверка детерминированности - один и тот же пароль и соль должны давать один и тот же ключ
	key2 := crypto.GenerateKey(password, salt)
	if !bytes.Equal(key, key2) {
		t.Error("Генерация ключа не детерминирована")
	}

	// Разные пароли должны давать разные ключи
	differentPassword := []byte("different_password")
	differentKey := crypto.GenerateKey(differentPassword, salt)
	if bytes.Equal(key, differentKey) {
		t.Error("Разные пароли дают одинаковые ключи")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	plaintext := []byte("секретное сообщение для тестирования")
	key := make([]byte, crypto.KeyLength)
	for i := range key {
		key[i] = byte(i % 256)
	}

	// Тест успешного шифрования и расшифровки
	ciphertext, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Ошибка шифрования: %v", err)
	}

	decrypted, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Ошибка расшифровки: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Расшифрованный текст не соответствует исходному")
	}

	// Тест с неверным ключом
	wrongKey := make([]byte, crypto.KeyLength)
	_, err = crypto.Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Error("Расшифровка с неверным ключом должна вернуть ошибку")
	}

	// Тест с неверной длиной ключа
	shortKey := make([]byte, crypto.KeyLength-1)
	_, err = crypto.Encrypt(plaintext, shortKey)
	if err != crypto.ErrInvalidKeyLength {
		t.Errorf("Ожидалась ошибка %v, получена %v", crypto.ErrInvalidKeyLength, err)
	}

	_, err = crypto.Decrypt(ciphertext, shortKey)
	if err != crypto.ErrInvalidKeyLength {
		t.Errorf("Ожидалась ошибка %v, получена %v", crypto.ErrInvalidKeyLength, err)
	}

	// Тест с поврежденными данными
	invalidCiphertext := append([]byte{}, ciphertext...)
	if len(invalidCiphertext) > 0 {
		invalidCiphertext[len(invalidCiphertext)-1] ^= 0x01 // Изменяем последний байт
	}
	_, err = crypto.Decrypt(invalidCiphertext, key)
	if err == nil {
		t.Error("Расшифровка поврежденных данных должна вернуть ошибку")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatalf("Ошибка генерации соли: %v", err)
	}

	if len(salt1) != crypto.SaltLength {
		t.Errorf("Длина соли %d не соответствует ожидаемой %d", len(salt1), crypto.SaltLength)
	}

	salt2, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatalf("Ошибка генерации соли: %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("Две последовательно сгенерированные соли одинаковы")
	}
}

func TestHashVerifyPassword(t *testing.T) {
	password := "test_password"

	// Тест успешного хеширования и проверки
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("Ошибка хеширования пароля: %v", err)
	}

	if !crypto.VerifyPassword(password, hash) {
		t.Error("Проверка пароля не удалась для правильного пароля")
	}

	// Тест с неверным паролем
	wrongPassword := "wrong_password"
	if crypto.VerifyPassword(wrongPassword, hash) {
		t.Error("Проверка пароля успешна для неверного пароля")
	}

	// Тест с неверным хешем
	if crypto.VerifyPassword(password, "invalid_hash") {
		t.Error("Проверка пароля успешна для неверного хеша")
	}

	// Тест с пустым паролем
	emptyHash, err := crypto.HashPassword("")
	if err != nil {
		t.Fatalf("Ошибка хеширования пустого пароля: %v", err)
	}

	if !crypto.VerifyPassword("", emptyHash) {
		t.Error("Проверка пароля не удалась для пустого пароля")
	}
}
